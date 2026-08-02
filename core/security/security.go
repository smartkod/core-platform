package security

import (
    "time"

    "github.com/labstack/echo/v4"
)

// Security ana security modÃ¼lÃ¼
type Security struct {
    rateLimiter *RateLimiter
    csrf        *CSRF
    validator   *Validator
    config      Config
}

// Config security yapÄ±landÄ±rmasÄ±
type Config struct {
    RateLimit       int
    RateLimitWindow time.Duration
    EnableCSRF      bool
    EnableCORS      bool
    CORSConfig      CORSConfig
}

// DefaultConfig varsayÄ±lan yapÄ±landÄ±rma
func DefaultConfig() Config {
    return Config{
        RateLimit:       100,
        RateLimitWindow: time.Minute,
        EnableCSRF:      true,
        EnableCORS:      true,
        CORSConfig:      DefaultCORSConfig(),
    }
}

// NewSecurity yeni security modÃ¼lÃ¼ oluÅŸturur
func NewSecurity(config Config) *Security {
    return &Security{
        rateLimiter: NewRateLimiter(config.RateLimit, config.RateLimitWindow),
        csrf:        NewCSRF(),
        validator:   NewValidator(),
        config:      config,
    }
}

// Setup security middleware'lerini ayarlar
func (s *Security) Setup(e *echo.Echo) {
    if s.config.EnableCORS {
        e.Use(CORS(s.config.CORSConfig))
    }
    e.Use(s.rateLimiter.Middleware())
    if s.config.EnableCSRF {
        e.Use(s.csrf.Middleware())
    }
}

// GetValidator validator'Ä± dÃ¶ndÃ¼rÃ¼r
func (s *Security) GetValidator() *Validator {
    return s.validator
}

// GetCSRF CSRF'yi dÃ¶ndÃ¼rÃ¼r
func (s *Security) GetCSRF() *CSRF {
    return s.csrf
}

// GetRateLimiter rate limiter'Ä± dÃ¶ndÃ¼rÃ¼r
func (s *Security) GetRateLimiter() *RateLimiter {
    return s.rateLimiter
}
