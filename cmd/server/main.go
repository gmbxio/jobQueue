package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/go-redis/redis/v8"
	"github.com/gmbxio/jobQueue/internal/db"
	"github.com/gmbxio/jobQueue/internal/queue"
	"github.com/gorilla/mux"
)

// Global variables so handlers.go can access them
var (
	dbConn      *sql.DB
	redisClient *redis.Client
)

func main() {
	fmt.Println("JobQueue API Server is waking up...")

	// 1. Initialize PostgreSQL
	dbConn = db.InitDB()
	defer dbConn.Close()

	// 2. Initialize Redis
	redisClient = queue.InitRedis()
	defer redisClient.Close()

	// 3. Set up the HTTP Router
	r := mux.NewRouter()

	// Health check
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("System is healthy! Databases connected."))
	}).Methods("GET")

	// Job API routes
	r.HandleFunc("/jobs", CreateJobHandler).Methods("POST")
	r.HandleFunc("/jobs", ListJobsHandler).Methods("GET")
	r.HandleFunc("/jobs/{id}", GetJobStatusHandler).Methods("GET")
	r.HandleFunc("/jobs/{id}", DeleteJobHandler).Methods("DELETE")

	// Serve frontend dashboard — must be last
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./cmd/server/static/")))

	// 4. Start the server
	fmt.Println("Server is running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", r))
}