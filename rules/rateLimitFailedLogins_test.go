package rules

import (
	"context"
	"fmt"
	"rba/services"
	"rba/util"
	"testing"
	"time"
)

func TestRateLimitFailedLoginsRule(t *testing.T) {
	raw := map[string]interface{}{
		"rollingWindowSeconds": 60,
		"threshold":            5,
		"strategy":             util.Strategies.Override,
	}

	handler, err := parseRateLimitFailedLoginsRule(raw)
	if err != nil {
		t.Fatalf("unexpected error parsing rule: %v", err)
	}

	if handler.Name != util.Rules.RateLimitFailedLogins {
		t.Errorf("expected rule name %s, got %s", util.Rules.RateLimitFailedLogins, handler.Name)
	}
	if handler.Strategy != util.Strategies.Override {
		t.Errorf("expected strategy %s, got %s", util.Strategies.Override, handler.Strategy)
	}
}

func TestRateLimitFailedLoginsThreshold(t *testing.T) {
	ctx := context.Background()
	ip := "1.1.1.1"
	rollingWindow := 10 * time.Second
	threshold := 2

	// clear key before test
	if err := services.RedisClient.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("failed to flush redis: %v", err)
	}

	// first attempt
	score, _ := EvaluateRateLimitFailedLoginsRisk(ctx, ip, rollingWindow, threshold)

	if score != 0.0 {
		t.Errorf("expected score 0.0, got %v", score)
	}

	// second attempt
	score, _ = EvaluateRateLimitFailedLoginsRisk(ctx, ip, rollingWindow, threshold)

	if score != 0.0 {
		t.Errorf("expected score 0.0, got %v", score)
	}

	key := "rateLimitFailedLogins:" + ip

	now := time.Now().UnixMilli()
	windowStart := float64(now - rollingWindow.Milliseconds())

	PrintAllItemsFromKey(ctx, key)

	count, _ := services.RedisClient.ZCount(ctx, key, fmt.Sprintf("%f", windowStart), "+inf").Result()

	fmt.Print("Current count: ", count, "\n")

	// third attempt should exceed threshold
	score, _ = EvaluateRateLimitFailedLoginsRisk(ctx, ip, rollingWindow, threshold)

	if score != 1.0 {
		t.Errorf("expected score 1.0, got %v", score)
	}
}
