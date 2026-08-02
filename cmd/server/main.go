package main

import (
"github.com/labstack/echo/v4"
"go.uber.org/zap"

"core-platform/core/config"
"core-platform/core/database"
"core-platform/core/logger"
"core-platform/core/module"
)

func main() {
cfg := config.New()
log := logger.New(cfg.Logger.Level, cfg.Logger.Format)
defer log.Sync()

log.Info("Starting Core Platform")

db, err := database.New(&cfg.Database, log)
if err != nil {
log.Fatal("Failed to connect to database", zap.Error(err))
}
defer db.Close()

manager := module.NewManager(log, &cfg.Modules)
e := echo.New()

// UTF-8 middleware
e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
return func(c echo.Context) error {
c.Response().Header().Set("Content-Type", "application/json; charset=utf-8")
return next(c)
}
})

e.GET("/health", func(c echo.Context) error {
return c.JSON(200, map[string]string{
"status":  "ok",
"service": "Core Platform",
"version": "1.0.0",
})
})

log.Info("Core Platform started")
e.Logger.Fatal(e.Start(":8081"))
}