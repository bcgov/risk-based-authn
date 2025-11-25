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

// EvaluatePasswordSprayRisk checks Redis for suspicious login failures
func EvaluatePasswordSprayRisk(ctx context.Context, ip string, interval time.Duration, attemptsAllowed int, distinctAccounts int) (float64, error) {
    now := time.Now().UnixMilli()
    windowStart := float64(now - interval.Milliseconds())
    key := fmt.Sprintf("passwordSpray:%s", ip)

    // Remove old entries
    if err := services.RedisClient.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%f", windowStart)).Err(); err != nil {
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

    services.RedisClient.Expire(ctx, key, interval)

    // Threshold check
    if count >= int64(attemptsAllowed) && count >= int64(distinctAccounts) {
        return 1.0, nil
    }
    return 0.0, nil
}

func parsePasswordSprayRule(raw map[string]interface{}) (util.NamedRiskHandler, error) {
    interval, ok := raw["intervalSeconds"].(int)
    if redisErr := services.PingRedis(); redisErr != nil {
        return util.NamedRiskHandler{}, errors.New("passwordSpray: a valid redis connection is required for this rule. Check redis configuration")
    }
    if !ok {
        return util.NamedRiskHandler{}, errors.New("passwordSpray: missing or invalid intervalSeconds")
    }

    attemptsAllowed, ok := raw["attemptsAllowed"].(int)
    if !ok {
        return util.NamedRiskHandler{}, errors.New("passwordSpray: missing or invalid attemptsAllowed")
    }

    distinctAccounts, ok := raw["distinctAccounts"].(int)
    if !ok {
        return util.NamedRiskHandler{}, errors.New("passwordSpray: missing or invalid distinctAccounts")
    }

    strategy, ok := raw["strategy"].(string)
    if !ok || !util.IsValidStrategy(strategy) {
        return util.NamedRiskHandler{}, errors.New("passwordSpray: missing or invalid strategy")
    }

    return util.NamedRiskHandler{
        Name:     util.Rules.PasswordSpray,
        Strategy: strategy,
        Handler: func(ctx context.Context, args map[string]interface{}) util.RiskResult {
            base := util.RiskResult{
                Name:     util.Rules.PasswordSpray,
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

            score, redisErr := EvaluatePasswordSprayRisk(ctx, ip, time.Duration(interval)*time.Second, attemptsAllowed, distinctAccounts)
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
