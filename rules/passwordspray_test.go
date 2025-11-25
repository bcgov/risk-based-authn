package rules

import (
    "context"
    "fmt"
    "os"
    "testing"
    "time"

    "github.com/redis/go-redis/v9"
    "rba/services"
    "rba/util"
)

// TestMain runs before all tests. It sets up a Redis client using env vars.
func TestMain(m *testing.M) {
    addr := os.Getenv("REDIS_ADDR")
    if addr == "" {
        addr = "localhost:6379" // default if not provided
    }

    services.RedisClient = redis.NewClient(&redis.Options{
        Addr: addr,
    })

    ctx := context.Background()
    if err := services.RedisClient.Ping(ctx).Err(); err != nil {
        panic(fmt.Sprintf("failed to connect to redis at %s: %v", addr, err))
    }

    // wipe all keys before tests
    if err := services.RedisClient.FlushDB(ctx).Err(); err != nil {
        panic(fmt.Sprintf("failed to flush redis before tests: %v", err))
    }

    code := m.Run()
    services.RedisClient.Close()
    os.Exit(code)
}

func TestParsePasswordSprayRule(t *testing.T) {
    raw := map[string]interface{}{
        "intervalSeconds":  10,
        "attemptsAllowed":  3,
        "distinctAccounts": 2,
        "strategy":         util.Strategies.Override,
    }

    handler, err := parsePasswordSprayRule(raw)
    if err != nil {
        t.Fatalf("unexpected error parsing rule: %v", err)
    }

    if handler.Name != util.Rules.PasswordSpray {
        t.Errorf("expected rule name %s, got %s", util.Rules.PasswordSpray, handler.Name)
    }
    if handler.Strategy != util.Strategies.Override {
        t.Errorf("expected strategy %s, got %s", util.Strategies.Override, handler.Strategy)
    }
}

func TestEvaluatePasswordSprayRisk(t *testing.T) {
    ctx := context.Background()
    ip := "1.2.3.4"
    key := fmt.Sprintf("passwordSpray:%s", ip)

    // clear key before test
    if err := services.RedisClient.Del(ctx, key).Err(); err != nil {
        t.Fatalf("failed to clear redis key: %v", err)
    }

    interval := 2 * time.Second
    attemptsAllowed := 2
    distinctAccounts := 1

    // First attempt should not exceed threshold
    score, err := EvaluatePasswordSprayRisk(ctx, ip, interval, attemptsAllowed, distinctAccounts)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if score != 0.0 {
        t.Errorf("expected score 0.0, got %v", score)
    }

    // Second attempt should exceed threshold
    score, err = EvaluatePasswordSprayRisk(ctx, ip, interval, attemptsAllowed, distinctAccounts)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if score != 1.0 {
        t.Errorf("expected score 1.0 after threshold exceeded, got %v", score)
    }
}
