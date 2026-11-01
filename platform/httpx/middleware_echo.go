package httpx

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Recover converts a panic into a 500 without killing the process.
func Recover() echo.MiddlewareFunc { return middleware.Recover() }

// BodyLimit rejects oversized requests before they reach a handler.
func BodyLimit(limit string) echo.MiddlewareFunc {
	return middleware.BodyLimit(limit)
}

// SecureHeaders sets the baseline security headers.
func SecureHeaders() echo.MiddlewareFunc { return middleware.Secure() }

// CORS installs an allow-list CORS policy. An empty list means no cross-origin
// access, which is the safe default.
func CORS(origins []string) echo.MiddlewareFunc {
	return middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: origins,
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{
			"Authorization", "Content-Type", HeaderRequestID, "Idempotency-Key",
		},
		ExposeHeaders:    []string{HeaderRequestID},
		AllowCredentials: len(origins) > 0,
		MaxAge:           600,
	})
}
