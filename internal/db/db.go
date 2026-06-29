package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func InitDB() *sql.DB {
	var database *sql.DB
	var err error

	// Railway provides DATABASE_URL — use it directly
	railwayURL := os.Getenv("DATABASE_URL")
	if railwayURL != "" {
		fmt.Println("Cloud environment detected. Connecting to managed PostgreSQL...")
		database, err = sql.Open("postgres", railwayURL)
		if err != nil {
			log.Fatalf("Error connecting to cloud database: %v", err)
		}
	} else {
		// Local Docker fallback
		fmt.Println("Local environment detected. Connecting to local PostgreSQL...")
		dbHost := os.Getenv("DB_HOST")
		if dbHost == "" {
			dbHost = "localhost"
		}

		maintenanceConnStr := fmt.Sprintf("postgres://postgres:password@%s:5432/postgres?sslmode=disable", dbHost)
		targetConnStr := fmt.Sprintf("postgres://postgres:password@%s:5432/job_db?sslmode=disable", dbHost)

		maintenanceDB, _ := sql.Open("postgres", maintenanceConnStr)
		var exists bool
		_ = maintenanceDB.QueryRow(`SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'job_db')`).Scan(&exists)
		if !exists {
			maintenanceDB.Exec(`CREATE DATABASE job_db`)
			fmt.Println("Created missing PostgreSQL database job_db.")
		}
		maintenanceDB.Close()

		database, err = sql.Open("postgres", targetConnStr)
		if err != nil {
			log.Fatalf("Error opening database connection: %v", err)
		}
	}

	if err = database.Ping(); err != nil {
		log.Fatalf("Could not ping database: %v", err)
	}

	fmt.Println("Successfully connected to PostgreSQL!")

	// Enable pgcrypto for gen_random_uuid()
	database.Exec(`CREATE EXTENSION IF NOT EXISTS pgcrypto`)

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS jobs (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		task_type VARCHAR(50) NOT NULL,
		payload JSONB NOT NULL,
		status VARCHAR(20) DEFAULT 'pending',
		result JSONB,
		error_message TEXT,
		retry_count INT DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);`

	if _, err := database.Exec(createTableSQL); err != nil {
		log.Fatalf("Error creating jobs table: %v", err)
	}

	fmt.Println("PostgreSQL database schema verified.")
	return database
}