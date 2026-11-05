package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

// A browser preflight only succeeds when every custom request header appears in
// Access-Control-Allow-Headers. Getting this wrong does not break Bearer API
// clients, so it fails only in the browser — the hardest place to notice.
func TestCORSPreflightAllowsClientHeaders(t *testing.T) {
	e := echo.New()
	e.Use(CORS([]string{"http://localhost:3000"}))
	e.POST("/probe", func(c echo.Context) error { return c.NoContent(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodOptions, "/probe", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers",
		"content-type, "+CSVHeaderCSRF+", "+CSVHeaderClientType)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", rec.Code)
	}

	allowed := strings.ToLower(rec.Header().Get("Access-Control-Allow-Headers"))
	for _, want := range []string{"content-type", strings.ToLower(CSVHeaderCSRF), strings.ToLower(CSVHeaderClientType)} {
		if !strings.Contains(allowed, want) {
			t.Errorf("Access-Control-Allow-Headers = %q, missing %q", allowed, want)
		}
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Errorf("allow-origin = %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

// An origin that is not on the list must not be echoed back: the browser then
// blocks the response, which is the intended "no cross-origin access" default.
func TestCORSRejectsUnknownOrigin(t *testing.T) {
	e := echo.New()
	e.Use(CORS([]string{"http://localhost:3000"}))
	e.GET("/probe", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Origin", "http://evil.example")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("allow-origin = %q, want empty", got)
	}
}
