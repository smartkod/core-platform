package security

import (
    "sync"
    "time"

    "github.com/labstack/echo/v4"
)

// RateLimiter rate limiting yapÄ±landÄ±rmasÄ±
type RateLimiter struct {
    mu       sync.RWMutex
    requests map[string][]time.Time
    limit    int
    window   time.Duration
}

// NewRateLimiter yeni rate limiter oluÅŸturur
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
    return &RateLimiter{
        requests: make(map[string][]time.Time),
        limit:    limit,
        window:   window,
    }
}

// Middleware rate limiting middleware'i
func (rl *RateLimiter) Middleware() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            key := c.RealIP()
            
            rl.mu.Lock()
            defer rl.mu.Unlock()
            
            // Eski kayÄ±tlarÄ± temizle
            now := time.Now()
            var valid []time.Time
            for _, t := range rl.requests[key] {
                if now.Sub(t) < rl.window {
                    valid = append(valid, t)
                }
            }
            rl.requests[key] = valid
            
            // Limit kontrolÃ¼
            if len(rl.requests[key]) >= rl.limit {
                return c.JSON(429, map[string]string{
                    "error": "Too many requests. Please try again later.",
                })
            }
            
            // Request'i kaydet
            rl.requests[key] = append(rl.requests[key], now)
            
            return next(c)
        }
    }
}
