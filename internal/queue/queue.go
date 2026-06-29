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

	// Railway provides REDIS_URL — use it directly
	redisURL := os.Getenv("REDIS_URL")
	if redisURL != "" {
		fmt.Println("Cloud environment detected. Connecting to managed Redis...")
		opt, err := redis.ParseURL(redisURL)
		if err != nil {
			log.Fatalf("Failed to parse REDIS_URL: %v", err)
		}
		rdb = redis.NewClient(opt)
	} else {
		// Local Docker fallback
		fmt.Println("Local environment detected. Connecting to local Redis...")
		redisHost := os.Getenv("REDIS_HOST")
		if redisHost == "" {
			redisHost = "localhost"
		}
		rdb = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:6379", redisHost),
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
