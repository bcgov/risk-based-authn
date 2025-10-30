package services

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/redis/go-redis/v9"
)

var (
	RedisClient *redis.Client
	oneRedis    sync.Once
)

// GetRedisClient returns a singleton Redis client
func ConnectRedis(host string) (*redis.Client, error) {
	var err error

	oneRedis.Do(func() {
		RedisClient = redis.NewClient(&redis.Options{
			Addr: host,
			DB:   0, // default DB
		})

		ctx := context.Background()
		if pingErr := RedisClient.Ping(ctx).Err(); pingErr != nil {
			err = fmt.Errorf("failed to connect to Redis at %s: %w", host, pingErr)
		}
	})

	return RedisClient, err
}

func PingRedis() error {
	ctx := context.Background()
	if RedisClient == nil {
		return errors.New("Redis unavailable")
	}
	return RedisClient.Ping(ctx).Err()
}
