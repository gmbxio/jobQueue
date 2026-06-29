package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type JobRequest struct {
	TaskType string          `json:"task_type"`
	Payload  json.RawMessage `json:"payload"`
}

type JobResponse struct {
	ID       string          `json:"id"`
	TaskType string          `json:"task_type"`
	Payload  json.RawMessage `json:"payload"`
	Status   string          `json:"status"`
	Result   json.RawMessage `json:"result,omitempty"`
	ErrorMsg string          `json:"error_message,omitempty"`
}

var ctx = context.Background()

func CreateJobHandler(w http.ResponseWriter, r *http.Request) {
	var req JobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	var jobID string
	query := `INSERT INTO jobs (task_type, payload, status) VALUES ($1, $2, 'pending') RETURNING id`
	err := dbConn.QueryRow(query, req.TaskType, req.Payload).Scan(&jobID)
	if err != nil {
		http.Error(w, "Failed to create job record", http.StatusInternalServerError)
		return
	}

	err = redisClient.LPush(ctx, "tasks_queue", jobID).Err()
	if err != nil {
		http.Error(w, "Failed to enqueue task", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"job_id":  jobID,
		"status":  "pending",
		"message": "Job safely queued!",
	})
}

func GetJobStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var job JobResponse
	query := `
		SELECT
			id,
			task_type,
			payload,
			status,
			COALESCE(result, 'null'::jsonb),
			COALESCE(error_message, '')
		FROM jobs
		WHERE id = $1`

	err := dbConn.QueryRow(query, id).Scan(
		&job.ID, &job.TaskType, &job.Payload, &job.Status, &job.Result, &job.ErrorMsg,
	)
	if err != nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}
// ListJobsHandler (GET /jobs) - Returns all jobs with their statuses
func ListJobsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := dbConn.Query(`
		SELECT id, task_type, status, created_at, updated_at 
		FROM jobs 
		ORDER BY created_at DESC
	`)
	if err != nil {
		http.Error(w, "Failed to fetch jobs", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type JobSummary struct {
		ID        string `json:"id"`
		TaskType  string `json:"task_type"`
		Status    string `json:"status"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	}

	jobs := []JobSummary{}
	for rows.Next() {
		var job JobSummary
		err := rows.Scan(&job.ID, &job.TaskType, &job.Status, &job.CreatedAt, &job.UpdatedAt)
		if err != nil {
			continue
		}
		jobs = append(jobs, job)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total": len(jobs),
		"jobs":  jobs,
	})
}

//Delete jobs
func DeleteJobHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	result, err := dbConn.Exec(`DELETE FROM jobs WHERE id = $1`, id)
	if err != nil {
		http.Error(w, "Failed to delete job", http.StatusInternalServerError)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}