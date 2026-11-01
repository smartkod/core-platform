// Package health serves liveness, readiness and version endpoints.
//
// Liveness must not touch the database: a DB outage should not cause the
// orchestrator to restart a healthy process. Readiness does check the DB, so
// traffic is withdrawn while the dependency is unavailable (L0.md 4.4).
package health

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// Checker reports whether a dependency is ready.
type Checker func(ctx context.Context) error

// Handler holds the readiness checks and build metadata.
type Handler struct {
	name      string
	version   string
	commit    string
	buildTime string
	checkers  map[string]Checker
}

// New builds the health handler.
func New(name, version, commit, buildTime string) *Handler {
	return &Handler{
		name: name, version: version, commit: commit, buildTime: buildTime,
		checkers: map[string]Checker{},
	}
}

// Add registers a named readiness check.
func (h *Handler) Add(name string, c Checker) { h.checkers[name] = c }

// Register mounts the three endpoints.
func (h *Handler) Register(g *echo.Group) {
	g.GET("/healthz", h.Live)
	g.GET("/readyz", h.Ready)
	g.GET("/version", h.Version)
}

// Live answers 200 whenever the process can serve HTTP.
func (h *Handler) Live(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Ready answers 200 only when every registered check passes.
func (h *Handler) Ready(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 3*time.Second)
	defer cancel()

	failed := map[string]string{}
	for name, check := range h.checkers {
		if err := check(ctx); err != nil {
			failed[name] = err.Error()
		}
	}

	if len(failed) > 0 {
		return c.JSON(http.StatusServiceUnavailable, map[string]any{
			"status": "unavailable",
			"failed": failed,
		})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ready"})
}

// Version returns build metadata stamped at link time.
func (h *Handler) Version(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"name":       h.name,
		"version":    h.version,
		"commit":     h.commit,
		"build_time": h.buildTime,
	})
}
