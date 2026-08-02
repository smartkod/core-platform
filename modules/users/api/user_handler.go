package api

import (
"net/http"
"strconv"

"modules/users/model"
"modules/users/service"

"github.com/labstack/echo/v4"
)

// UserHandler kullanÄ±cÄ± handler'Ä±
type UserHandler struct {
userService *service.UserService
}

// NewUserHandler yeni handler oluÅŸturur
func NewUserHandler(userService *service.UserService) *UserHandler {
return &UserHandler{
userService: userService,
}
}

// GetMe kullanÄ±cÄ± profilini getirir
func (h *UserHandler) GetMe(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1) // GeÃ§ici

user, err := h.userService.GetUserByID(userID)
if err != nil {
return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
}

return c.JSON(http.StatusOK, user)
}

// UpdateMe kullanÄ±cÄ± profilini gÃ¼nceller
func (h *UserHandler) UpdateMe(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1) // GeÃ§ici

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

// GetAllUsers tÃ¼m kullanÄ±cÄ±larÄ± getirir (admin)
func (h *UserHandler) GetAllUsers(c echo.Context) error {
users, err := h.userService.GetAllUsers()
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
}

return c.JSON(http.StatusOK, users)
}

// GetUserByID kullanÄ±cÄ± detayÄ±nÄ± getirir (admin)
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

// DeleteUser kullanÄ±cÄ± siler (admin)
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
