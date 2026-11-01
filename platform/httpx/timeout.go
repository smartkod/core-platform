package httpx

import (
	"context"
	"time"

	"github.com/labstack/echo/v4"
)

// Timeout bounds how long a handler may work before its context is cancelled.
//
// It deliberately does not use http.TimeoutHandler (which echo's own Timeout
// middleware wraps). That implementation runs the handler in a goroutine,
// replaces the ResponseWriter with a buffering wrapper, and re-panics from the
// handler goroutine. Two consequences make it unusable here:
//
//   - it buffers the whole response in memory, so a streaming file download of
//     any size would be held in RAM before the first byte reaches the client;
//   - it panics with a nil dereference when the wrapped writer is used by
//     handlers that parse multipart bodies or write directly to the response.
//
// Cancelling the request context is what actually protects the server: a
// dependency call that respects ctx stops, and the handler returns an error that
// the envelope renders. The handler still owns its response, so streaming keeps
// working.
func Timeout(d time.Duration) echo.MiddlewareFunc {
	if d <= 0 {
		return func(next echo.HandlerFunc) echo.HandlerFunc { return next }
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx, cancel := context.WithTimeout(c.Request().Context(), d)
			defer cancel()

			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}
