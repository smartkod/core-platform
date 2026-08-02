package api

import (
"net/http"

"core-platform/modules/auth/model"
"core-platform/modules/auth/service"

"github.com/labstack/echo/v4"
)

// AuthHandler auth handler
type AuthHandler struct {
authService *service.AuthService
}

// NewAuthHandler yeni auth handler oluÅŸturur
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
return &AuthHandler{
authService: authService,
}
}

// Register kullanÄ±cÄ± kaydÄ±
func (h *AuthHandler) Register(c echo.Context) error {
var req model.RegisterRequest
if err := c.Bind(&req); err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
}

user, err := h.authService.Register(&req)
if err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
}

return c.JSON(http.StatusCreated, user)
}

// Login kullanÄ±cÄ± giriÅŸi
func (h *AuthHandler) Login(c echo.Context) error {
var req model.LoginRequest
if err := c.Bind(&req); err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
}

response, err := h.authService.Login(&req)
if err != nil {
return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
}

return c.JSON(http.StatusOK, response)
}
