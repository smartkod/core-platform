// Package httpx builds the HTTP server: router, middleware chain, single error
// envelope and the health endpoints.
package httpx

import (
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"core-platform/contract/core"
)

// ErrorBody is the error object inside the envelope.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// Envelope is the only response shape the HTTP layer produces for errors.
type Envelope struct {
	Success   bool      `json:"success"`
	Error     ErrorBody `json:"error"`
	RequestID string    `json:"request_id"`
}

// codeOf maps a sentinel to its stable public code and status.
//
// This is the single place where internal errors become HTTP responses. 5xx
// bodies never carry an internal message (L0.md 4.4).
func codeOf(err error) (string, int) {
	switch {
	case errors.Is(err, core.ErrUnauthorized):
		return "UNAUTHORIZED", http.StatusUnauthorized
	case errors.Is(err, core.ErrForbidden):
		return "FORBIDDEN", http.StatusForbidden
	case errors.Is(err, core.ErrNotFound):
		return "NOT_FOUND", http.StatusNotFound
	case errors.Is(err, core.ErrValidation):
		return "VALIDATION_ERROR", http.StatusUnprocessableEntity
	case errors.Is(err, core.ErrConflict):
		return "CONFLICT", http.StatusConflict
	case errors.Is(err, core.ErrRateLimited):
		return "RATE_LIMITED", http.StatusTooManyRequests
	case errors.Is(err, core.ErrNotImplemented):
		return "NOT_IMPLEMENTED", http.StatusNotImplemented
	case errors.Is(err, core.ErrProviderFailure):
		return "PROVIDER_ERROR", http.StatusBadGateway
	case errors.Is(err, core.ErrInsufficientBalance):
		return "INSUFFICIENT_BALANCE", http.StatusPaymentRequired
	case errors.Is(err, core.ErrEntitlementRequired):
		return "ENTITLEMENT_REQUIRED", http.StatusPaymentRequired
	}
	return "INTERNAL", http.StatusInternalServerError
}

// ErrorHandler installs the envelope as echo's error handler.
func ErrorHandler(l core.Logger) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}

		code, status := codeOf(err)
		msg := err.Error()

		// Echo's own errors (404 route not found, 405) keep their status but
		// get the same stable code format as platform sentinels.
		var he *echo.HTTPError
		if errors.As(err, &he) {
			if status == http.StatusInternalServerError {
				status = he.Code
				code = statusCode(he.Code)
				msg = http.StatusText(he.Code)
			}
		}

		if status >= 500 {
			msg = "internal server error"
			l.Error("request failed",
				"code", code, "status", status,
				"request_id", RequestIDFrom(c), "err", err.Error())
		}

		_ = c.JSON(status, Envelope{
			Success:   false,
			Error:     ErrorBody{Code: code, Message: msg},
			RequestID: RequestIDFrom(c),
		})
	}
}

// statusCode turns an HTTP status into the same SCREAMING_SNAKE code style the
// platform sentinels use, so a client never sees two formats.
func statusCode(status int) string {
	text := http.StatusText(status)
	if text == "" {
		return "ERROR"
	}
	return strings.ToUpper(strings.ReplaceAll(text, " ", "_"))
}

// RespondOK is the success envelope helper.
func RespondOK(c echo.Context, status int, data any) error {
	return c.JSON(status, struct {
		Success   bool   `json:"success"`
		Data      any    `json:"data,omitempty"`
		RequestID string `json:"request_id"`
	}{
		Success:   true,
		Data:      data,
		RequestID: RequestIDFrom(c),
	})
}
