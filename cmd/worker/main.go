package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gmbxio/jobQueue/internal/db"
	"github.com/gmbxio/jobQueue/internal/models"
	"github.com/gmbxio/jobQueue/internal/queue"
)

// GroqRequest matches the Groq API /chat/completions schema
type GroqRequest struct {
	Model    string        `json:"model"`
	Messages []GroqMessage `json:"messages"`
}

type GroqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// GroqResponse matches what Groq returns
type GroqResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// JobPayload is what the client sends in the payload field
type JobPayload struct {
	Prompt string `json:"prompt"`
	Model  string `json:"model"`
}

// processJobs is the core worker loop run by each goroutine
func processJobs(id int, groqAPIKey string) {
	groqURL := "https://api.groq.com/openai/v1/chat/completions"

	dbConn := db.InitDB()
	defer dbConn.Close()

	redisClient := queue.InitRedis()
	defer redisClient.Close()

	log.Printf("[Worker %d] Ready — listening to Redis queue...\n", id)

	for {
		// Block until a job ID arrives in the Redis queue
		result, err := redisClient.BRPop(queue.Ctx, 0, "tasks_queue").Result()
		if err != nil {
			log.Printf("[Worker %d] Redis pop error: %v\n", id, err)
			time.Sleep(1 * time.Second)
			continue
		}

		jobID := result[1]
		fmt.Printf("\n[Worker %d][+] Picked up Job ID: %s\n", id, jobID)

		// 1. Mark job as running
		_, _ = dbConn.Exec(
			`UPDATE jobs SET status = 'running', updated_at = NOW() WHERE id = $1`,
			jobID,
		)

		// 2. Fetch the stored payload from PostgreSQL
		var payloadBytes []byte
		err = dbConn.QueryRow(`SELECT payload FROM jobs WHERE id = $1`, jobID).Scan(&payloadBytes)
		if err != nil {
			log.Printf("[Worker %d][-] Failed to fetch payload for job %s: %v\n", id, jobID, err)
			_, _ = dbConn.Exec(
				`UPDATE jobs SET status = 'failed', error_message = $1, updated_at = NOW() WHERE id = $2`,
				fmt.Sprintf("Failed to fetch payload: %v", err), jobID,
			)
			continue
		}

		// 3. Decode the stored payload
		var payload JobPayload
		if err := json.Unmarshal(payloadBytes, &payload); err != nil {
			log.Printf("[Worker %d][-] Failed to parse payload for job %s: %v\n", id, jobID, err)
			_, _ = dbConn.Exec(
				`UPDATE jobs SET status = 'failed', error_message = $1, updated_at = NOW() WHERE id = $2`,
				fmt.Sprintf("Invalid payload format: %v", err), jobID,
			)
			continue
		}

		// Validate required fields
		if payload.Prompt == "" {
			errMsg := "Payload missing required field: prompt"
			log.Printf("[Worker %d][-] Job %s: %s\n", id, jobID, errMsg)
			_, _ = dbConn.Exec(
				`UPDATE jobs SET status = 'failed', error_message = $1, updated_at = NOW() WHERE id = $2`,
				errMsg, jobID,
			)
			continue
		}
		if !models.IsSupported(payload.Model) {
			errMsg := fmt.Sprintf("Unsupported model: %s", payload.Model)
			log.Printf("[Worker %d][-] Job %s: %s\n", id, jobID, errMsg)
			_, _ = dbConn.Exec(
				`UPDATE jobs SET status = 'failed', error_message = $1, updated_at = NOW() WHERE id = $2`,
				errMsg, jobID,
			)
			continue
		}

		fmt.Printf("[Worker %d]    -> Forwarding to Groq | model: %s | prompt: %.60s...\n",
			id, payload.Model, payload.Prompt)

		// 4. Build the Groq request
		groqReq := GroqRequest{
			Model: payload.Model,
			Messages: []GroqMessage{
				{Role: "user", Content: payload.Prompt},
			},
		}

		groqReqBytes, _ := json.Marshal(groqReq)

		// 5. POST to Groq API
		startTime := time.Now()

		httpReq, err := http.NewRequest("POST", groqURL, bytes.NewBuffer(groqReqBytes))
		if err != nil {
			_, _ = dbConn.Exec(
				`UPDATE jobs SET status = 'failed', error_message = $1, updated_at = NOW() WHERE id = $2`,
				fmt.Sprintf("Failed to build request: %v", err), jobID,
			)
			continue
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+groqAPIKey)

		client := &http.Client{Timeout: 60 * time.Second}
		resp, err := client.Do(httpReq)
		if err != nil {
			errMsg := fmt.Sprintf("Groq API unreachable: %v", err)
			_, _ = dbConn.Exec(
				`UPDATE jobs SET status = 'failed', error_message = $1, updated_at = NOW() WHERE id = $2`,
				errMsg, jobID,
			)
			fmt.Printf("[Worker %d][X] Job %s FAILED: Groq unreachable.\n", id, jobID)
			continue
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		duration := time.Since(startTime).Seconds()

		// 6. Handle non-200 from Groq
		if resp.StatusCode != http.StatusOK {
			errMsg := fmt.Sprintf("Groq returned status %d: %s", resp.StatusCode, string(bodyBytes))
			_, _ = dbConn.Exec(
				`UPDATE jobs SET status = 'failed', error_message = $1, updated_at = NOW() WHERE id = $2`,
				errMsg, jobID,
			)
			fmt.Printf("[Worker %d][X] Job %s FAILED: Groq error.\n", id, jobID)
			continue
		}

		// 7. Parse Groq's response
		var groqResp GroqResponse
		if err := json.Unmarshal(bodyBytes, &groqResp); err != nil {
			errMsg := fmt.Sprintf("Failed to parse Groq response: %v", err)
			_, _ = dbConn.Exec(
				`UPDATE jobs SET status = 'failed', error_message = $1, updated_at = NOW() WHERE id = $2`,
				errMsg, jobID,
			)
			fmt.Printf("[Worker %d][X] Job %s FAILED: Could not parse Groq output.\n", id, jobID)
			continue
		}

		// Extract the response text
		responseText := ""
		if len(groqResp.Choices) > 0 {
			responseText = groqResp.Choices[0].Message.Content
		}

		// 8. Build final result with metadata
		finalResult := map[string]interface{}{
			"model":                           groqResp.Model,
			"response":                        responseText,
			"prompt_tokens":                   groqResp.Usage.PromptTokens,
			"completion_tokens":               groqResp.Usage.CompletionTokens,
			"total_tokens":                    groqResp.Usage.TotalTokens,
			"queue_execution_latency_seconds": duration,
		}
		finalResultBytes, _ := json.Marshal(finalResult)

		// 9. Mark job as completed in PostgreSQL
		_, err = dbConn.Exec(
			`UPDATE jobs SET status = 'completed', result = $1, error_message = NULL, updated_at = NOW() WHERE id = $2`,
			finalResultBytes, jobID,
		)
		if err != nil {
			log.Printf("[Worker %d][-] Failed to update completed state for job %s: %v\n", id, jobID, err)
		} else {
			fmt.Printf("[Worker %d][✓] Job %s completed in %.2fs | model: %s | tokens: %d\n",
				id, jobID, duration, groqResp.Model, groqResp.Usage.TotalTokens)
		}
	}
}

func main() {
	fmt.Println("JobQueue Worker Engine initialized...")

	// Read Groq API key from environment variable
	groqAPIKey := os.Getenv("GROQ_API_KEY")

	// Read concurrency from env, default to 3
	concurrency := 3
	if val := os.Getenv("WORKER_CONCURRENCY"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			concurrency = n
		}
	}

	if groqAPIKey == "" {
		log.Println("WARNING: GROQ_API_KEY is not set; worker is running in health-check-only mode. Set GROQ_API_KEY to process jobs.")
	} else {
		fmt.Printf("Starting %d concurrent worker goroutines...\n", concurrency)

		for i := 1; i <= concurrency; i++ {
			go func(id int) {
				processJobs(id, groqAPIKey)
			}(i)
		}
	}

	// Minimal health-check HTTP server so Render's free "web" service type
	// keeps this process alive. The real work happens in the goroutines above.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "worker ok")
	})
	fmt.Printf("Health-check server listening on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
