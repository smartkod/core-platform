package main

import (
"net/http"
"time"

"core-platform/core/security"

"github.com/labstack/echo/v4"
"github.com/labstack/echo/v4/middleware"
)

func main() {
// Echo instance
e := echo.New()

// Logger ve Recover
e.Use(middleware.Logger())
e.Use(middleware.Recover())

// Security middleware
secConfig := security.DefaultConfig()
secConfig.RateLimit = 100                    // Dakikada 100 istek
secConfig.RateLimitWindow = time.Minute
secConfig.EnableCSRF = false                // Şimdilik devre dışı
secConfig.EnableCORS = true

sec := security.NewSecurity(secConfig)
sec.Setup(e)

// Health check
e.GET("/health", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]string{
"status":  "ok",
"service": "Core Platform",
"version": "0.7.0",
})
})

// Test route
e.GET("/", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]string{
"message": "Core Platform v0.7.0",
"status":  "running",
})
})

// --- AUTH ROUTES ---
authGroup := e.Group("/api/auth")
authGroup.POST("/register", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]string{
"message": "Register endpoint working!",
"email":   "test@test.com",
})
})
authGroup.POST("/login", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]string{
"message": "Login endpoint working!",
"token":   "fake-jwt-token",
})
})

// --- USERS ROUTES ---
userGroup := e.Group("/api/users")

// GET /api/users/me - Kullanıcı profili
userGroup.GET("/me", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]interface{}{
"id":         1,
"email":      "test@test.com",
"name":       "Test User",
"language":   "tr",
"avatar":     "",
"is_active":  true,
"created_at": "2026-07-31T10:00:00Z",
"updated_at": "2026-07-31T10:00:00Z",
})
})

// PUT /api/users/me - Kullanıcı güncelle
userGroup.PUT("/me", func(c echo.Context) error {
var req map[string]string
if err := c.Bind(&req); err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
}

return c.JSON(http.StatusOK, map[string]interface{}{
"message":  "User updated successfully",
"name":     req["name"],
"language": req["language"],
})
})

// GET /api/users - Tüm kullanıcılar (admin)
userGroup.GET("", func(c echo.Context) error {
return c.JSON(http.StatusOK, []map[string]interface{}{
{
"id":         1,
"email":      "test@test.com",
"name":       "Test User",
"language":   "tr",
"is_active":  true,
},
{
"id":         2,
"email":      "admin@test.com",
"name":       "Admin User",
"language":   "en",
"is_active":  true,
},
})
})

// GET /api/users/:id - Kullanıcı detayı (admin)
userGroup.GET("/:id", func(c echo.Context) error {
id := c.Param("id")
return c.JSON(http.StatusOK, map[string]interface{}{
"id":         id,
"email":      "user" + id + "@test.com",
"name":       "User " + id,
"language":   "tr",
"is_active":  true,
})
})

// DELETE /api/users/:id - Kullanıcı sil (admin)
userGroup.DELETE("/:id", func(c echo.Context) error {
id := c.Param("id")
return c.JSON(http.StatusOK, map[string]string{
"message": "User " + id + " deleted successfully",
})
})

// Start server
e.Logger.Fatal(e.Start(":8081"))
}
