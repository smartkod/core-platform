package api

import (
"net/http"
"strconv"

"modules/users/model"
"modules/users/service"

"github.com/labstack/echo/v4"
)

// UserHandler kullanıcı handler'ı
type UserHandler struct {
userService *service.UserService
}

// NewUserHandler yeni handler oluşturur
func NewUserHandler(userService *service.UserService) *UserHandler {
return &UserHandler{
userService: userService,
}
}

// GetMe kullanıcı profilini getirir
func (h *UserHandler) GetMe(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1) // Geçici

user, err := h.userService.GetUserByID(userID)
if err != nil {
return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
}

return c.JSON(http.StatusOK, user)
}

// UpdateMe kullanıcı profilini günceller
func (h *UserHandler) UpdateMe(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1) // Geçici

var req model.UpdateUserRequest
if err := c.Bind(&req); err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
}

user, err := h.userService.UpdateUser(userID, &req)
if err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
}

return c.JSON(http.StatusOK, user)
}

// GetAllUsers tüm kullanıcıları getirir (admin)
func (h *UserHandler) GetAllUsers(c echo.Context) error {
users, err := h.userService.GetAllUsers()
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
}

return c.JSON(http.StatusOK, users)
}

// GetUserByID kullanıcı detayını getirir (admin)
func (h *UserHandler) GetUserByID(c echo.Context) error {
id, err := strconv.ParseUint(c.Param("id"), 10, 32)
if err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID"})
}

user, err := h.userService.GetUserByID(uint(id))
if err != nil {
return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
}

return c.JSON(http.StatusOK, user)
}

// DeleteUser kullanıcı siler (admin)
func (h *UserHandler) DeleteUser(c echo.Context) error {
id, err := strconv.ParseUint(c.Param("id"), 10, 32)
if err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID"})
}

if err := h.userService.DeleteUser(uint(id)); err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
}

return c.JSON(http.StatusOK, map[string]string{"message": "User deleted successfully"})
}
