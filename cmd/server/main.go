package main

import (
"net/http"

"github.com/labstack/echo/v4"
"github.com/labstack/echo/v4/middleware"
)

func main() {
e := echo.New()

// Middleware
e.Use(middleware.Logger())
e.Use(middleware.Recover())
e.Use(middleware.CORS())

// Test routes
e.GET("/test", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]string{
"message": "Test route works!",
})
})

e.GET("/health", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]string{
"status": "ok",
"service": "Test Server",
})
})

e.POST("/api/auth/register", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]string{
"message": "Register works!",
})
})

e.POST("/api/auth/login", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]string{
"message": "Login works!",
})
})

// Start server
e.Logger.Fatal(e.Start(":8081"))
}
