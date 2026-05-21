package database

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

func NewRedis(addr, password string) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("redis ping failed: %v", err)
	}
	log.Println("✅ Connected to Redis")
	return rdb
}
