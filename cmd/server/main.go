package main

import (
"github.com/labstack/echo/v4"
"go.uber.org/zap"

"core-platform/core/config"
"core-platform/core/database"
"core-platform/core/logger"
)

func main() {
// Load config
cfg := config.New()

// Initialize logger
log := logger.New(cfg.Logger.Level, cfg.Logger.Format)
defer log.Sync()

log.Info("Starting Core Platform")

// Initialize database
db, err := database.New(&cfg.Database, log)
if err != nil {
log.Fatal("Failed to connect to database", zap.Error(err))
}
defer db.Close()

// Create router
e := echo.New()

// Health check
e.GET("/health", func(c echo.Context) error {
return c.JSON(200, map[string]string{
"status":  "ok",
"service": "Core Platform",
"version": "1.0.0",
})
})

// Start server
e.Logger.Fatal(e.Start(":8081"))
}