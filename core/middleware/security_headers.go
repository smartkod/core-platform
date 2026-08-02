package middleware

import (
"github.com/labstack/echo/v4"
)

// SecurityHeaders gÃƒÂ¼venlik baÃ…Å¸lÃ„Â±klarÃ„Â± ekler
func SecurityHeaders() echo.MiddlewareFunc {
return func(next echo.HandlerFunc) echo.HandlerFunc {
return func(c echo.Context) error {
// HSTS (HTTP Strict Transport Security)
c.Response().Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")

// X-Content-Type-Options
c.Response().Header().Set("X-Content-Type-Options", "nosniff")

// X-Frame-Options
c.Response().Header().Set("X-Frame-Options", "DENY")

// X-XSS-Protection
c.Response().Header().Set("X-XSS-Protection", "1; mode=block")

// Content-Security-Policy
c.Response().Header().Set("Content-Security-Policy", "default-src 'self'")

// Referrer-Policy
c.Response().Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

// Permissions-Policy
c.Response().Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

return next(c)
}
}
}