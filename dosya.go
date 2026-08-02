package main

import (
"core-platform/core/api"
"core-platform/core/localization"

"github.com/labstack/echo/v4"
"github.com/labstack/echo/v4/middleware"
)

func main() {
// Echo instance
e := echo.New()

// Middleware
e.Use(middleware.Logger())
e.Use(middleware.Recover())
e.Use(middleware.CORS())

// Localization
localizer, err := localization.NewLocalizer(localization.DefaultConfig())
if err != nil {
panic("Failed to initialize localization: " + err.Error())
}
e.Use(localizer.GetMiddleware().Middleware())

// Health check (Ã§oklu dil desteÄŸi ile)
e.GET("/health", func(c echo.Context) error {
msg := localization.T(c, "welcome", nil)
return api.SuccessResponse(c, map[string]interface{}{
"status":   "ok",
"service":  "Core Platform",
"version":  "0.9.0",
"language": localization.GetLocale(c),
"message":  msg,
})
})

// Test route
e.GET("/", func(c echo.Context) error {
greeting := localization.T(c, "greeting", map[string]string{"Name": "Developer"})
return api.SuccessResponse(c, map[string]interface{}{
"message":  greeting,
"language": localization.GetLocale(c),
"supported": localizer.GetSupportedLocales(),
})
})

// --- AUTH ROUTES ---
authGroup := e.Group("/api/auth")
authGroup.POST("/register", func(c echo.Context) error {
msg := localization.T(c, "auth.register_success", nil)
return api.SuccessResponse(c, map[string]string{
"message": msg,
"email":   "test@test.com",
})
})
authGroup.POST("/login", func(c echo.Context) error {
msg := localization.T(c, "auth.login_success", nil)
return api.SuccessResponse(c, map[string]string{
"message": msg,
"token":   "fake-jwt-token",
})
})

// --- USERS ROUTES ---
userGroup := e.Group("/api/users")

userGroup.GET("/me", func(c echo.Context) error {
msg := localization.T(c, "user.profile", nil)
return api.SuccessResponse(c, map[string]interface{}{
"id":      1,
"email":   "test@test.com",
"name":    "Test User",
"message": msg,
})
})

// Start server
e.Logger.Fatal(e.Start(":8081"))
}
