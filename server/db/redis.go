package db

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/OnnaSoft/lipstick/server/config"
	"github.com/redis/go-redis/v9"
)

var (
	defaultRedisClient *redis.Client
	ctx                = context.Background()
)

func NewRedisConnection() (*redis.Client, error) {
	conf, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	client := redis.NewClient(&redis.Options{
		Addr:         conf.Redis.Host + ":" + string(rune(conf.Redis.Port)),
		Password:     conf.Redis.Password,
		DB:           conf.Redis.Database,
		PoolSize:     conf.Redis.PoolSize,
		MinIdleConns: conf.Redis.MinIdleConns,
		PoolTimeout:  time.Duration(conf.Redis.PoolTimeout) * time.Second,
	})

	_, err = client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	if os.Getenv("DEBUG") == "true" {
		log.Println("Redis connected with DEBUG mode enabled")
	}

	return client, nil
}

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
