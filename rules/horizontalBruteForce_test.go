package rules

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"rba/services"
	"rba/util"

	"github.com/redis/go-redis/v9"
)

// TestMain runs before all tests. It sets up a Redis client using env vars.
func TestMain(m *testing.M) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
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

func TestParseHorizontalBruteForceRule(t *testing.T) {
	raw := map[string]interface{}{
		"intervalSeconds":  10,
		"distinctAccounts": 2,
		"strategy":         util.Strategies.Override,
	}

	handler, err := parseHorizontalBruteForceRule(raw)
	if err != nil {
		t.Fatalf("unexpected error parsing rule: %v", err)
	}

	if handler.Name != util.Rules.HorizontalBruteForce {
		t.Errorf("expected rule name %s, got %s", util.Rules.HorizontalBruteForce, handler.Name)
	}
	if handler.Strategy != util.Strategies.Override {
		t.Errorf("expected strategy %s, got %s", util.Strategies.Override, handler.Strategy)
	}
}

func TestEvaluateHorizontalBruteForceRisk(t *testing.T) {
	ctx := context.Background()
	ip := "1.2.3.4"
	interval := 2 * time.Second
	distinctAccounts := 3

	// clear key before test
	if err := services.RedisClient.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("failed to flush redis: %v", err)
	}

	// First attempt on "alice" should not exceed threshold
	score, err := EvaluateHorizontalBruteForceRisk(ctx, ip, "alice", interval, distinctAccounts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score != 0.0 {
		t.Errorf("expected score 0.0, got %v", score)
	}

	// Multiple attempts on the same account ("alice") should still count as 1 distinct account
	for i := 0; i < 3; i++ {
		_, _ = EvaluateHorizontalBruteForceRisk(ctx, ip, "alice", interval, distinctAccounts)
	}
	score, _ = EvaluateHorizontalBruteForceRisk(ctx, ip, "alice", interval, distinctAccounts)
	if score != 0.0 {
		t.Errorf("expected score 0.0 for repeated alice attempts, got %v", score)
	}

	// Add a second distinct account ("bob") from the same IP
	score, _ = EvaluateHorizontalBruteForceRisk(ctx, ip, "bob", interval, distinctAccounts)
	if score != 0.0 {
		t.Errorf("expected score 0.0 when alice+bob are within distinctAccounts threshold, got %v", score)
	}

	// Add a third distinct account ("charlie") from the same IP
	score, _ = EvaluateHorizontalBruteForceRisk(ctx, ip, "charlie", interval, distinctAccounts)
	if score != 1.0 {
		t.Errorf("expected score 1.0 when alice+bob+charlie exceed distinctAccounts threshold, got %v", score)
	}
}
