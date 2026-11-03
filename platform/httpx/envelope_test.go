package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"core-platform/contract/core"
)

type noopLogger struct{}

func (noopLogger) Info(string, ...any)     {}
func (noopLogger) Warn(string, ...any)     {}
func (noopLogger) Error(string, ...any)    {}
func (noopLogger) With(...any) core.Logger { return noopLogger{} }

// newServer wires the same handler stack the kernel uses, minus the database.
func newServer(handler echo.HandlerFunc) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = ErrorHandler(noopLogger{})
	e.GET("/probe", handler)
	return e
}

func do(t *testing.T, e *echo.Echo, path string) (*httptest.ResponseRecorder, Envelope) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	var env Envelope
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	return rec, env
}

func TestSentinelMapping(t *testing.T) {
	cases := []struct {
		err    error
		code   string
		status int
	}{
		{core.ErrUnauthorized, "UNAUTHORIZED", http.StatusUnauthorized},
		{core.ErrForbidden, "FORBIDDEN", http.StatusForbidden},
		{core.ErrNotFound, "NOT_FOUND", http.StatusNotFound},
		{core.ErrValidation, "VALIDATION_ERROR", http.StatusUnprocessableEntity},
		{core.ErrConflict, "CONFLICT", http.StatusConflict},
		{core.ErrRateLimited, "RATE_LIMITED", http.StatusTooManyRequests},
		{core.ErrNotImplemented, "NOT_IMPLEMENTED", http.StatusNotImplemented},
		{core.ErrProviderFailure, "PROVIDER_ERROR", http.StatusBadGateway},
		{core.ErrInvalidState, "INVALID_STATE", http.StatusConflict},
		{core.ErrNotMember, "NOT_A_MEMBER", http.StatusForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			e := newServer(func(echo.Context) error { return tc.err })
			rec, env := do(t, e, "/probe")

			if rec.Code != tc.status {
				t.Errorf("status = %d, want %d", rec.Code, tc.status)
			}
			if env.Success {
				t.Error("success should be false")
			}
			if env.Error.Code != tc.code {
				t.Errorf("code = %q, want %q", env.Error.Code, tc.code)
			}
		})
	}
}

func TestInternalErrorHidesMessage(t *testing.T) {
	secret := "connection string with password=hunter2"
	e := newServer(func(echo.Context) error { return errString(secret) })

	rec, env := do(t, e, "/probe")

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "hunter2") {
		t.Fatal("internal error message leaked to the client")
	}
	if env.Error.Message != "internal server error" {
		t.Errorf("message = %q, want generic text", env.Error.Message)
	}
}

type errString string

func (e errString) Error() string { return string(e) }

func TestRequestIDIsReturned(t *testing.T) {
	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = ErrorHandler(noopLogger{})
	e.Use(RequestID())
	e.GET("/probe", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"id": RequestIDFrom(c)})
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	got := rec.Header().Get(HeaderRequestID)
	if got == "" {
		t.Fatal("X-Request-ID header missing")
	}
	if !strings.Contains(rec.Body.String(), got) {
		t.Error("handler did not see the same request id")
	}
}

func TestInboundRequestIDIsPreserved(t *testing.T) {
	e := echo.New()
	e.HideBanner = true
	e.Use(RequestID())
	e.GET("/probe", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set(HeaderRequestID, "caller-supplied-id")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if got := rec.Header().Get(HeaderRequestID); got != "caller-supplied-id" {
		t.Errorf("request id = %q, want caller-supplied-id", got)
	}
}
