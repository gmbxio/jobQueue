package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq" // Postgres driver
)

// InitDB initializes the connection pool to PostgreSQL and creates the jobs table
func InitDB() *sql.DB {
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}

	const (
		databaseName = "job_db"
	)

	maintenanceConnStr := fmt.Sprintf("postgres://postgres:password@%s:5432/postgres?sslmode=disable", dbHost)
	targetConnStr := fmt.Sprintf("postgres://postgres:password@%s:5432/job_db?sslmode=disable", dbHost)

	maintenanceDB, err := sql.Open("postgres", maintenanceConnStr)
	if err != nil {
		log.Fatalf("Error opening maintenance database connection: %v", err)
	}
	defer maintenanceDB.Close()

	if err := maintenanceDB.Ping(); err != nil {
		log.Fatalf("Could not ping the maintenance database: %v", err)
	}

	var exists bool
	if err := maintenanceDB.QueryRow(`SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`, databaseName).Scan(&exists); err != nil {
		log.Fatalf("Could not check database existence: %v", err)
	}

	if !exists {
		if _, err := maintenanceDB.Exec(`CREATE DATABASE job_db`); err != nil {
			log.Fatalf("Could not create database %s: %v", databaseName, err)
		}
		fmt.Println("Created missing PostgreSQL database job_db.")
	}

	database, err := sql.Open("postgres", targetConnStr)
	if err != nil {
		log.Fatalf("Error opening database connection: %v", err)
	}

	if err := database.Ping(); err != nil {
		log.Fatalf("Could not ping the database: %v", err)
	}

	fmt.Println("Successfully connected to PostgreSQL!")

	if _, err := database.Exec(`CREATE EXTENSION IF NOT EXISTS pgcrypto`); err != nil {
		log.Fatalf("Error enabling pgcrypto extension: %v", err)
	}

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
