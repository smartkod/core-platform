package httpx

import (
	"net/http"
	"sort"
	"strings"

	"github.com/labstack/echo/v4"
)

// RouteInfo is one entry in the discovery index.
type RouteInfo struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

// Index serves a discovery document at the API root.
//
// A backend with no HTML front page answers 404 at "/" by default, which reads
// as "broken" to anyone poking at it in a browser. This turns that dead end
// into an answer: which endpoints exist, in what order to call them.
//
// It lists only method and path. That is not a secret — every path is already
// in the public documentation — and it saves a developer from guessing.
func Index(name, version, env string, routes func() []RouteInfo) echo.HandlerFunc {
	return func(c echo.Context) error {
		all := routes()

		health := make([]string, 0, 4)
		api := make([]string, 0, len(all))
		for _, r := range all {
			line := r.Method + " " + r.Path
			switch {
			case r.Path == "/" || strings.HasSuffix(r.Path, "/*"):
				continue // the index itself and the catch-all
			case r.Method == "GET" && (r.Path == "/healthz" || r.Path == "/readyz" || r.Path == "/version"):
				health = append(health, line)
			case strings.HasPrefix(r.Path, "/api/"):
				api = append(api, line)
			}
		}
		sort.Strings(health)
		sort.Strings(api)

		return c.JSON(http.StatusOK, map[string]any{
			"name":    name,
			"version": version,
			"env":     env,
			"message": "This is a JSON API, not a web page. Try one of the endpoints below.",
			"health":  health,
			"api":     api,
			"auth": map[string]string{
				"flow":    "POST /api/v1/auth/register or /login to get access_token, then send it as 'Authorization: Bearer <token>'",
				"refresh": "POST /api/v1/auth/refresh with the refresh_token to rotate it",
			},
		})
	}
}

// RoutesOf converts echo's route table into the index format.
//
// Echo registers its own "route not found" handlers as entries in that table
// with a synthetic method name; they are implementation detail, not endpoints,
// so they are filtered out rather than shown to a caller.
func RoutesOf(e *echo.Echo) func() []RouteInfo {
	return func() []RouteInfo {
		src := e.Routes()
		out := make([]RouteInfo, 0, len(src))
		for _, r := range src {
			if r.Method == "" || strings.HasPrefix(r.Method, "echo") {
				continue
			}
			out = append(out, RouteInfo{Method: r.Method, Path: r.Path})
		}
		return out
	}
}
