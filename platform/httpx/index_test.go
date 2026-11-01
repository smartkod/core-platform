package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

// The discovery index exists so a browser visit to "/" explains the API instead
// of returning a bare 404. These tests pin the two properties that make it
// useful: it lists real endpoints, and it does not list echo's internals.

func indexServer(t *testing.T) *echo.Echo {
	t.Helper()
	e := echo.New()
	e.HideBanner = true

	e.GET("/healthz", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	e.GET("/readyz", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	e.GET("/version", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	e.POST("/api/v1/auth/login", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	e.GET("/api/v1/me", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	// Echo's catch-all registration, which must not appear in the listing.
	e.RouteNotFound("", func(c echo.Context) error { return echo.ErrNotFound })
	e.RouteNotFound("/*", func(c echo.Context) error { return echo.ErrNotFound })

	e.GET("/", Index("demo", "1.0.0", "development", RoutesOf(e)))
	return e
}

func indexDoc(t *testing.T, e *echo.Echo) map[string]any {
	t.Helper()
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
}

func TestIndexListsEndpoints(t *testing.T) {
	doc := indexDoc(t, indexServer(t))

	if doc["name"] != "demo" {
		t.Errorf("name = %v, want demo", doc["name"])
	}
	if doc["env"] != "development" {
		t.Errorf("env = %v, want development", doc["env"])
	}

	health, ok := doc["health"].([]any)
	if !ok || len(health) != 3 {
		t.Fatalf("health = %v, want 3 entries", doc["health"])
	}

	api, ok := doc["api"].([]any)
	if !ok || len(api) != 2 {
		t.Fatalf("api = %v, want 2 entries", doc["api"])
	}
}

func TestIndexHidesEchoInternals(t *testing.T) {
	doc := indexDoc(t, indexServer(t))

	raw, _ := json.Marshal(doc)
	if strings.Contains(string(raw), "echo_route_not_found") {
		t.Fatal("echo's internal route registration leaked into the index")
	}
}

func TestIndexHasGuidance(t *testing.T) {
	doc := indexDoc(t, indexServer(t))

	msg, _ := doc["message"].(string)
	if !strings.Contains(msg, "JSON API") {
		t.Errorf("message = %q, want it to explain that this is an API", msg)
	}

	auth, ok := doc["auth"].(map[string]any)
	if !ok || auth["flow"] == "" {
		t.Error("auth guidance is missing; a caller cannot tell how to authenticate")
	}
}
