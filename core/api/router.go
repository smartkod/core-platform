package api

import (
"net/http"

"core-platform/core/health"
"core-platform/core/logger"

"github.com/labstack/echo/v4"
"github.com/labstack/echo/v4/middleware"
"go.uber.org/zap"
)

type Router struct {
echo    *echo.Echo
logger  *logger.Logger
health  *health.HealthChecker
}

func NewRouter(logger *logger.Logger, healthChecker *health.HealthChecker) *Router {
e := echo.New()

// Middleware
e.Use(middleware.Logger())
e.Use(middleware.Recover())
e.Use(middleware.CORS())

r := &Router{
echo:   e,
logger: logger,
health: healthChecker,
}

logger.Info("Router created")
return r
}

// SetupRoutes route'larÄ± ayarlar
func (r *Router) SetupRoutes() {
r.logger.Info("Setting up default routes...")

// Root endpoint
r.echo.GET("/", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]string{
"message": "Core Platform v0.5.0",
"status":  "running",
})
})

// Health check endpoint - basit versiyon
r.echo.GET("/health", func(c echo.Context) error {
if r.health == nil {
return c.JSON(http.StatusOK, map[string]string{
"status":  "ok",
"service": "Core Platform",
"version": "0.5.0",
})
}
return c.JSON(http.StatusOK, r.health.Check())
})

r.logger.Info("Default routes registered: /, /health")
}

// SetHealthChecker health checker'Ä± ayarlar
func (r *Router) SetHealthChecker(healthChecker *health.HealthChecker) {
r.health = healthChecker
r.logger.Info("Health checker updated")

// Health check route'u gÃ¼ncelle - override
r.echo.GET("/health", func(c echo.Context) error {
return c.JSON(http.StatusOK, r.health.Check())
})
}

// Start sunucuyu baÅŸlatÄ±r
func (r *Router) Start(address string) error {
r.logger.Info("Starting server",
zap.String("address", address),
)
return r.echo.Start(address)
}

// GetEcho echo instance'Ä±nÄ± dÃ¶ndÃ¼rÃ¼r
func (r *Router) GetEcho() *echo.Echo {
return r.echo
}
