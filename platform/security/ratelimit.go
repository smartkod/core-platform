// Package security holds the rate limiter.
//
// The limiter is in-memory with TTL cleanup. It is correct for a single
// instance, which is the V1 deployment shape; a shared store is a later
// concern and is deliberately not abstracted yet (L0.md 8.1 L0-8).
package security

import (
	"sync"
	"time"

	"github.com/labstack/echo/v4"

	"core-platform/contract/core"
)

type bucket struct {
	count int
	reset time.Time
}

// RateLimiter is a fixed-window limiter keyed by a caller-provided string.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	limit   int
	window  time.Duration
	now     func() time.Time
}

// NewRateLimiter builds a limiter. A cleanup goroutine removes expired
// buckets so memory cannot grow without bound.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		buckets: map[string]*bucket{},
		limit:   limit,
		window:  window,
		now:     func() time.Time { return time.Now().UTC() },
	}
	go rl.cleanup()
	return rl
}

func (r *RateLimiter) cleanup() {
	t := time.NewTicker(r.window)
	defer t.Stop()
	for range t.C {
		now := r.now()
		r.mu.Lock()
		for k, b := range r.buckets {
			if now.After(b.reset) {
				delete(r.buckets, k)
			}
		}
		r.mu.Unlock()
	}
}

// Allow records a hit for key and reports whether it is within the limit.
func (r *RateLimiter) Allow(key string) bool {
	now := r.now()
	r.mu.Lock()
	defer r.mu.Unlock()

	b, ok := r.buckets[key]
	if !ok || now.After(b.reset) {
		r.buckets[key] = &bucket{count: 1, reset: now.Add(r.window)}
		return true
	}
	if b.count >= r.limit {
		return false
	}
	b.count++
	return true
}

// Refund gives back one unit for key.
//
// It exists so only failed attempts consume the budget. Without this, a shared
// office NAT or a mobile carrier gateway would exhaust a login limit for every
// user behind it, while a brute-force attacker would be treated identically —
// the limit would punish the honest case and barely slow the hostile one.
func (r *RateLimiter) Refund(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if b, ok := r.buckets[key]; ok && b.count > 0 {
		b.count--
	}
}

// Middleware returns an echo middleware keyed by an arbitrary function.
func (r *RateLimiter) Middleware(keyFn func(echo.Context) string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if !r.Allow(keyFn(c)) {
				return core.ErrRateLimited
			}
			return next(c)
		}
	}
}

// ByIP is the default key function.
func ByIP(c echo.Context) string { return c.RealIP() }
