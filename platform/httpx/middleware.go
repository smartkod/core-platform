package httpx

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/labstack/echo/v4"
)

// HeaderRequestID is echoed back on every response so a caller can correlate a
// failure with the logs (L0.md 4.4).
const HeaderRequestID = "X-Request-ID"

const requestIDKey = "request_id"

// RequestIDFrom returns the current request's id, or "" outside a request.
func RequestIDFrom(c echo.Context) string {
	if v, ok := c.Get(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// RequestID assigns an id, reusing an inbound one when the caller supplied it.
func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			id := c.Request().Header.Get(HeaderRequestID)
			if id == "" {
				id = newID()
			}
			c.Set(requestIDKey, id)
			c.Response().Header().Set(HeaderRequestID, id)
			return next(c)
		}
	}
}

// NotFound is the handler for unmatched routes; the error handler turns it
// into the standard envelope.
func NotFound(c echo.Context) error {
	return echo.NewHTTPError(http.StatusNotFound)
}

func newID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b[:])
}
