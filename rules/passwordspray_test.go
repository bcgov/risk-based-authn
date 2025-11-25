package rules

import (
	"testing"
	"time"
)

func TestDetectPasswordSpraySuspicious(t *testing.T) {
	rule:= PasswordSprayRule {
		Name: "password_spray",
		Timeframe: 5 * time.Minute,
		AttemptsAllowed: 3,
		DistinctAccounts: 2,
	}

	ip:= "1.2.3.4"
	now:= time.Now()

	attempts:= []LoginAttempt{
		{Username: "alice", IP: ip, Timestamp: now, Success: false},
		{Username: "bob", IP: ip, Timestamp: now, Success: false},
		{Username: "carol", IP: ip, Timestamp: now, Success: false},
	}
	if !detectPasswordSpray(ip, attempts, rule) {
		t.Errorf("Expected spray detection to trigger")
	}
}

func TestDetectPasswordSprayNormalTraffic(t *testing.T){
	rule:= PasswordSprayRule {
		Name: "password_spray",
		Timeframe: 5 * time.Minute,
		AttemptsAllowed: 3,
		DistinctAccounts: 2,
	}

	ip:= "1.2.3.4"
	now:= time.Now()

	attempts:= []LoginAttempt{
		{Username: "alice", IP: ip, Timestamp: now, Success: false},
		{Username: "alice", IP: ip, Timestamp: now, Success: false},
		{Username: "alice", IP: ip, Timestamp: now, Success: false},
	}

	if detectPasswordSpray(ip, attempts, rule){
		t.Errorf("Did not expect spray detection for single account brute force")
	}
}