package queue

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/go-redis/redis/v8"
)

var Ctx = context.Background()

func InitRedis() *redis.Client {
	var rdb *redis.Client

	// Prefer a full Redis URL if provided (Railway / Render internal URL)
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		fmt.Println("Cloud environment detected. Connecting via REDIS_URL...")
		opt, err := redis.ParseURL(redisURL)
		if err != nil {
			log.Fatalf("Failed to parse REDIS_URL: %v", err)
		}
		rdb = redis.NewClient(opt)
	} else {
		// Build from individual env vars (Render Blueprint / local Docker)
		host := os.Getenv("REDIS_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("REDIS_PORT")
		if port == "" {
			port = "6379"
		}
		fmt.Printf("Connecting to Redis at %s:%s...\n", host, port)
		rdb = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", host, port),
			Password: "",
			DB:       0,
		})
	}

	if err := rdb.Ping(Ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	fmt.Println("Successfully connected to Redis!")
	return rdb
}
