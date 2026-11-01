package security

import (
	"testing"
	"time"
)

func TestRefundForgetsOneAttempt(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)

	if !rl.Allow("k") || !rl.Allow("k") {
		t.Fatal("both attempts should be allowed")
	}
	if rl.Allow("k") {
		t.Fatal("third attempt should be denied")
	}

	// A refund is what a successful request does.
	rl.Refund("k")
	if !rl.Allow("k") {
		t.Fatal("after a refund one more attempt should be allowed")
	}
}

func TestRefundNeverGoesNegative(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)

	// Refunding a key that was never used must not create credit that could be
	// spent later; otherwise traffic on one path could pre-pay another.
	rl.Refund("unused")

	if !rl.Allow("unused") {
		t.Fatal("a refunded-but-unused key should still allow its first attempt")
	}
	if !rl.Allow("unused") {
		t.Fatal("second attempt should be allowed")
	}
	if rl.Allow("unused") {
		t.Fatal("third attempt should be denied: the refund must not have granted credit")
	}
}

func TestRefundOnMissingKeyIsSafe(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)
	rl.Refund("never-seen") // must not panic
	rl.Refund("never-seen")
}

// TestSuccessfulAttemptsDoNotAccumulate documents the property that matters in
// production: ten successful logins in a row must not lock a user out. Without
// it, everyone behind one NAT shares a budget and the limit punishes honest
// traffic while barely slowing an attacker.
func TestSuccessfulAttemptsDoNotAccumulate(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute)

	for i := 0; i < 50; i++ {
		if !rl.Allow("user") {
			t.Fatalf("successful attempt %d was denied", i+1)
		}
		rl.Refund("user") // what the middleware does when the handler succeeds
	}
}

// TestFailedAttemptsStillExhaustBudget is the counterpart: brute force must
// still be stopped.
func TestFailedAttemptsStillExhaustBudget(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute)

	for i := 0; i < 3; i++ {
		if !rl.Allow("attacker") {
			t.Fatalf("attempt %d should have been allowed before the limit", i+1)
		}
		// No refund: the handler failed.
	}
	if rl.Allow("attacker") {
		t.Fatal("the fourth failed attempt must be denied")
	}
}
