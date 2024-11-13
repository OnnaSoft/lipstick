package db

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/juliotorresmoreno/lipstick/server/config"
	"github.com/redis/go-redis/v9"
)

var (
	defaultRedisClient *redis.Client
	ctx                = context.Background()
)

// NewRedisConnection creates a new Redis client with connection pool support based on the configuration.
func NewRedisConnection() (*redis.Client, error) {
	// Get configuration
	conf, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Create Redis client with pool options from configuration
	client := redis.NewClient(&redis.Options{
		Addr:         conf.Redis.Host + ":" + string(rune(conf.Redis.Port)),
		Password:     conf.Redis.Password,                                 // Password for Redis (empty if not set)
		DB:           conf.Redis.Database,                                 // Default DB index
		PoolSize:     conf.Redis.PoolSize,                                 // Max number of connections
		MinIdleConns: conf.Redis.MinIdleConns,                             // Min idle connections
		PoolTimeout:  time.Duration(conf.Redis.PoolTimeout) * time.Second, // Pool timeout in seconds
	})

	// Test the connection
	_, err = client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	// Enable debug logs if the DEBUG environment variable is set
	if os.Getenv("DEBUG") == "true" {
		log.Println("Redis connected with DEBUG mode enabled")
	}

	return client, nil
}

// GetRedisConnection provides the default Redis connection, creating it if necessary.
func GetRedisConnection() (*redis.Client, error) {
	if defaultRedisClient == nil {
		client, err := NewRedisConnection()
		if err != nil {
			return nil, err
		}
		defaultRedisClient = client
	}
	return defaultRedisClient, nil
}

// CloseRedisConnection closes the Redis connection pool.
func CloseRedisConnection() {
	if defaultRedisClient != nil {
		err := defaultRedisClient.Close()
		if err != nil {
			log.Printf("Error closing Redis connection: %v", err)
		} else {
			log.Println("Redis connection closed successfully")
		}
	}
}
