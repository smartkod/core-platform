package security

import (
	"testing"
	"time"
)

func TestRateLimiterAllowsUpToLimit(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute)

	for i := 0; i < 3; i++ {
		if !rl.Allow("user") {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	if rl.Allow("user") {
		t.Fatal("request 4 should be denied")
	}
}

func TestRateLimiterIsolatesKeys(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)

	if !rl.Allow("a") {
		t.Fatal("first request for a should be allowed")
	}
	if !rl.Allow("b") {
		t.Fatal("first request for b should be allowed")
	}
	if rl.Allow("a") {
		t.Fatal("second request for a should be denied")
	}
}

func TestRateLimiterResetsAfterWindow(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rl.now = func() time.Time { return now }

	if !rl.Allow("k") {
		t.Fatal("first request should be allowed")
	}
	if rl.Allow("k") {
		t.Fatal("second request inside the window should be denied")
	}

	now = now.Add(2 * time.Minute)
	if !rl.Allow("k") {
		t.Fatal("request after the window should be allowed")
	}
}
