package rules

import (
	"context"
	"rba/services"
	"rba/util"
	"testing"
	"time"
)

func TestRateLimitFailedLoginsRule(t *testing.T) {
	raw := map[string]interface{}{
		"intervalSeconds": 60,
		"threshold":       5,
		"strategy":        util.Strategies.Override,
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
	ctx := WithRequestEventType(context.Background(), "login_failure")
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
		t.Errorf("expected score 0, got %v", score)
	}

	time.Sleep(5 * time.Millisecond)

	// second attempt
	score, _ = EvaluateRateLimitFailedLoginsRisk(ctx, ip, rollingWindow, threshold)

	if score != 0.0 {
		t.Errorf("expected score 0, got %v", score)
	}

	time.Sleep(5 * time.Millisecond)

	// third attempt should exceed threshold
	score, _ = EvaluateRateLimitFailedLoginsRisk(ctx, ip, rollingWindow, threshold)

	if score != 1 {
		t.Errorf("expected score 1, got %v", score)
	}

	ctx = WithRequestEventType(context.Background(), "login")

	// successful login attempt should reset rate limit
	score, _ = EvaluateRateLimitFailedLoginsRisk(ctx, ip, rollingWindow, threshold)

	if score != 0 {
		t.Errorf("expected score 0, got %v", score)
	}

	// fourth attempt
	score, _ = EvaluateRateLimitFailedLoginsRisk(ctx, ip, rollingWindow, threshold)

	if score != 0 {
		t.Errorf("expected score 0, got %v", score)
	}
}
