package services

import (
	"sync"

	"github.com/redis/go-redis/v9"
)

var (
	RedisClient *redis.Client
	oneRedis    sync.Once
)

// GetRedisClient returns a singleton Redis client
func ConnectRedis(host string) *redis.Client {
	oneRedis.Do(func() {
		RedisClient = redis.NewClient(&redis.Options{
			Addr: host, // adjust if needed
			// Password: "", // no password set by default
			DB: 0, // default DB
		})
	})
	return RedisClient
}
