package security

import (
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
)

// CORSConfig CORS yapılandırması
type CORSConfig struct {
    AllowOrigins     []string
    AllowMethods     []string
    AllowHeaders     []string
    ExposeHeaders    []string
    AllowCredentials bool
    MaxAge           int
}

// DefaultCORSConfig varsayılan CORS yapılandırması
func DefaultCORSConfig() CORSConfig {
    return CORSConfig{
        AllowOrigins:     []string{"*"},
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowHeaders:     []string{"Authorization", "Content-Type", "X-CSRF-Token"},
        ExposeHeaders:    []string{"Content-Length"},
        AllowCredentials: true,
        MaxAge:           300,
    }
}

// CORS CORS middleware'ini döndürür
func CORS(config CORSConfig) echo.MiddlewareFunc {
    return middleware.CORSWithConfig(middleware.CORSConfig{
        AllowOrigins:     config.AllowOrigins,
        AllowMethods:     config.AllowMethods,
        AllowHeaders:     config.AllowHeaders,
        ExposeHeaders:    config.ExposeHeaders,
        AllowCredentials: config.AllowCredentials,
        MaxAge:           config.MaxAge,
    })
}
