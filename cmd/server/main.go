package main

import (
"net/http"

"github.com/labstack/echo/v4"
"github.com/labstack/echo/v4/middleware"
)

func main() {
e := echo.New()
e.Use(middleware.Logger())
e.Use(middleware.Recover())
e.Use(middleware.CORS())

// Health
e.GET("/health", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
})

// Users test
e.GET("/api/users/me", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]string{
"id":    "1",
"email": "test@test.com",
"name":  "Test User",
})
})

e.GET("/api/users", func(c echo.Context) error {
return c.JSON(http.StatusOK, []map[string]string{
{"id": "1", "email": "test@test.com", "name": "Test User"},
{"id": "2", "email": "admin@test.com", "name": "Admin User"},
})
})

e.Start(":8081")
}
