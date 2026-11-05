package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func headerProbe(t *testing.T, cfg SecureHeaderConfig) http.Header {
	t.Helper()
	e := echo.New()
	e.Use(SecureHeaders(cfg))
	e.GET("/probe", func(c echo.Context) error { return c.JSON(http.StatusOK, map[string]bool{"ok": true}) })

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/probe", nil))
	return rec.Header()
}

func TestSecureHeadersBaseline(t *testing.T) {
	h := headerProbe(t, SecureHeaderConfig{})

	cases := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "no-referrer",
	}
	for name, want := range cases {
		if got := h.Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}

	csp := h.Get("Content-Security-Policy")
	for _, want := range []string{"default-src 'none'", "frame-ancestors 'none'"} {
		if !strings.Contains(csp, want) {
			t.Errorf("Content-Security-Policy = %q, missing %q", csp, want)
		}
	}
}

// HSTS over plain HTTP would pin a scheme the development server cannot serve,
// so it must only appear in production.
func TestSecureHeadersHSTSOnlyInProduction(t *testing.T) {
	if got := headerProbe(t, SecureHeaderConfig{}).Get("Strict-Transport-Security"); got != "" {
		t.Errorf("development sent Strict-Transport-Security = %q, want none", got)
	}

	got := headerProbe(t, SecureHeaderConfig{Production: true}).Get("Strict-Transport-Security")
	if !strings.Contains(got, "max-age=") {
		t.Errorf("production Strict-Transport-Security = %q, want a max-age", got)
	}
}

// The legacy auditor was removed from browsers and could itself be abused; its
// presence would be a false signal of protection.
func TestSecureHeadersDoesNotSendXSSProtection(t *testing.T) {
	if got := headerProbe(t, SecureHeaderConfig{}).Get("X-XSS-Protection"); got != "" {
		t.Errorf("X-XSS-Protection = %q, want none", got)
	}
}

func TestSecureHeadersHonoursOverride(t *testing.T) {
	const custom = "default-src 'self'"
	if got := headerProbe(t, SecureHeaderConfig{ContentSecurityPolicy: custom}).Get("Content-Security-Policy"); got != custom {
		t.Errorf("Content-Security-Policy = %q, want %q", got, custom)
	}
}

func TestNoStoreMarksWrites(t *testing.T) {
	e := echo.New()
	e.Use(NoStore())
	e.POST("/write", func(c echo.Context) error { return c.NoContent(http.StatusNoContent) })
	e.GET("/read", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/write", nil))
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("POST Cache-Control = %q, want no-store", got)
	}

	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/read", nil))
	if got := rec.Header().Get("Cache-Control"); got == "no-store" {
		t.Error("a GET response should be cacheable; only writes are marked no-store")
	}
}
