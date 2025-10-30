package rules

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"rba/services"
	"rba/util"
	"slices"
	"time"

	"github.com/redis/go-redis/v9"
)

var x = 1

func EvaluateVelocityRisk(ctx context.Context, ip string, interval time.Duration, limit int) (float64, error) {
	now := time.Now().UnixMilli()
	windowStart := float64(now - interval.Milliseconds())
	key := fmt.Sprintf("velocity:%s", ip)

	// Remove old entries
	if err := services.RedisClient.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%f", windowStart)).Err(); err != nil {
		return 0, err
	}

	// Add login event
	member := fmt.Sprintf("%d-%d", now, rand.Intn(1000000))
	if err := services.RedisClient.ZAdd(ctx, key, redis.Z{
		Score:  float64(now),
		Member: member,
	}).Err(); err != nil {
		return 0, err
	}

	// Count current entries
	count, err := services.RedisClient.ZCount(ctx, key, fmt.Sprintf("%f", windowStart), "+inf").Result()
	if err != nil {
		return 0, err
	}

	services.RedisClient.Expire(ctx, key, interval)

	if count > int64(limit) {
		return 1.0, nil
	}

	return 0.0, nil
}

func parseVelocityRule(raw map[string]interface{}) (util.NamedRiskHandler, error) {
	interval, ok := raw["intervalSeconds"].(int)
	if !ok {
		return util.NamedRiskHandler{}, errors.New("velocity: missing or invalid intervalSeconds")
	}

	limit, ok := raw["limit"].(int)
	if !ok {
		return util.NamedRiskHandler{}, errors.New("velocity: missing or invalid limit")
	}

	strategy, ok := raw["strategy"].(string)
	if !ok || !slices.Contains(util.Strategies, strategy) {
		return util.NamedRiskHandler{}, errors.New("velocity: missing or invalid strategy")
	}

	return util.NamedRiskHandler{
		Name:     "velocity",
		Strategy: strategy,
		Handler: func(ctx context.Context, args map[string]interface{}) util.RiskResult {
			now := time.Now().UnixMilli()
			println(now)
			base := util.RiskResult{
				Name:     "velocity",
				Strategy: strategy,
				Score:    0,
				Err:      nil,
			}

			ip, err := util.GetStringField(args, "ip")
			if err != nil {
				errText := "missing ip"
				result := base
				result.Err = &errText
				return result
			}

			score, redisErr := EvaluateVelocityRisk(ctx, ip, time.Duration(interval)*time.Second, limit)
			result := base
			result.Score = score
			if redisErr != nil {
				errText := redisErr.Error()
				result.Err = &errText
			}
			return result
		},
	}, nil
}
