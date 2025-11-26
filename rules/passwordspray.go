package rules

import (
    "context"
    "errors"
    "fmt"
    "rba/services"
    "rba/util"
    "time"

   // "github.com/redis/go-redis/v9"
)

// EvaluatePasswordSprayRisk checks Redis for suspicious login failures
// Counts distinct accounts per IP, not repeated attempts on the same account.
func EvaluatePasswordSprayRisk(
    ctx context.Context,
    ip string,
    account string,
    interval time.Duration,
    attemptsAllowed int,   // kept for signature consistency, but not used in spray detection
    distinctAccounts int,
) (float64, error) {

    // Track distinct accounts per IP using a Redis set
    distinctKey := fmt.Sprintf("passwordSpray:distinct:%s", ip)
    if err := services.RedisClient.SAdd(ctx, distinctKey, account).Err(); err != nil {
        return 0, err
    }
    if err := services.RedisClient.Expire(ctx, distinctKey, interval).Err(); err != nil {
        return 0, err
    }

    // Fetch distinct account count
    distinctCount, err := services.RedisClient.SCard(ctx, distinctKey).Result()
    if err != nil {
        return 0, err
    }

    // Threshold check: only distinct accounts matter
    if int(distinctCount) > distinctAccounts {
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

            account, err := util.GetStringField(args, "account")
            if err != nil {
                errText := "missing account"
                result := base
                result.Err = &errText
                return result
            }

            score, redisErr := EvaluatePasswordSprayRisk(
                ctx,
                ip,
                account,
                time.Duration(interval)*time.Second,
                attemptsAllowed,   // not used in spray detection
                distinctAccounts,
            )

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
