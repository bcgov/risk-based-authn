package rules

import (
	"context"
	"errors"
	"fmt"
	"rba/services"
	"rba/util"
	"time"

	"github.com/redis/go-redis/v9"
)

func ResetRateLimitUponSuccess(ctx context.Context, ip string) error {
	key := "rateLimitFailedLogins:" + ip
	// Delete the key; no error if key does not exist
	if err := services.RedisClient.Del(ctx, key).Err(); err != nil {
		return err
	}
	return nil
}

func EvaluateRateLimitFailedLoginsRisk(ctx context.Context, ip string, rollingWindow time.Duration, threshold int) (float64, error) {

	eventType, ok := RequestEventTypeFromContext(ctx)

	if !ok || (eventType != "login_failure" && eventType != "login") {
		return 0, errors.New("invalid event type for rateLimitFailedLogins rule")
	}

	// If login is successful, reset the rate limit failed logins counter
	if eventType == "login" {
		err := ResetRateLimitUponSuccess(ctx, ip)
		if err != nil {
			fmt.Printf("Error resetting rate limit failed logins: %v\n", err)
		}
		return 0, nil
	}

	now := time.Now().UnixMilli()
	windowStart := float64(now - rollingWindow.Milliseconds())
	key := "rateLimitFailedLogins:" + ip

	// Remove old entries
	if err := services.RedisClient.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%f", windowStart)).Err(); err != nil {
		return 0, err
	}

	// Add login failure event
	member := fmt.Sprintf("%d", now)
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

	services.RedisClient.Expire(ctx, key, rollingWindow)

	if count > int64(threshold) {
		return 1, nil
	}

	return 0, nil
}

func parseRateLimitFailedLoginsRule(raw map[string]interface{}) (util.NamedRiskHandler, error) {

	if redisErr := services.PingRedis(); redisErr != nil {
		return util.NamedRiskHandler{}, errors.New("velocity: a valid redis connection is required for this rule. Check redis configuration")
	}

	threshold, ok := raw["threshold"].(int)
	if !ok {
		return util.NamedRiskHandler{}, errors.New("rateLimitFailedLogins: missing or invalid threshold")
	}

	rollingWindowSeconds, ok := raw["rollingWindowSeconds"].(int)
	if !ok {
		return util.NamedRiskHandler{}, errors.New("rateLimitFailedLogins: missing or invalid rollingWindowSeconds")
	}

	strategy, ok := raw["strategy"].(string)
	if !ok || !util.IsValidStrategy(strategy) {
		return util.NamedRiskHandler{}, errors.New("rateLimitFailedLogins: missing or invalid strategy")
	}

	return util.NamedRiskHandler{
		Name:     util.Rules.RateLimitFailedLogins,
		Strategy: strategy,
		Handler: func(ctx context.Context, args map[string]interface{}) util.RiskResult {
			base := util.RiskResult{
				Name:     util.Rules.RateLimitFailedLogins,
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

			score, err := EvaluateRateLimitFailedLoginsRisk(ctx, ip, time.Duration(rollingWindowSeconds)*time.Second, threshold)
			result := base
			result.Score = score
			if err != nil {
				errText := err.Error()
				result.Err = &errText
			}
			return result
		},
	}, nil
}
