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

	// Prefer a full connection URL if provided (Railway / Render internal URL)
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		fmt.Println("Cloud environment detected. Connecting via DATABASE_URL...")
		database, err = sql.Open("postgres", dbURL)
		if err != nil {
			log.Fatalf("Error connecting to cloud database: %v", err)
		}
	} else {
		// Build connection string from individual env vars (Render Blueprint / local Docker)
		host := os.Getenv("DB_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("DB_PORT")
		if port == "" {
			port = "5432"
		}
		user := os.Getenv("DB_USER")
		if user == "" {
			user = "postgres"
		}
		password := os.Getenv("DB_PASSWORD")
		if password == "" {
			password = "password"
		}
		dbName := os.Getenv("DB_NAME")
		if dbName == "" {
			dbName = "job_db"
		}

		// On Render managed Postgres, SSL is required
		sslMode := "disable"
		if os.Getenv("DB_HOST") != "" && os.Getenv("DB_HOST") != "localhost" {
			sslMode = "require"
		}

		connStr := fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=%s",
			user, password, host, port, dbName, sslMode,
		)
		fmt.Printf("Connecting to PostgreSQL at %s:%s/%s...\n", host, port, dbName)

		database, err = sql.Open("postgres", connStr)
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