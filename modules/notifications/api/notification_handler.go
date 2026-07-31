package api

import (
"net/http"
"strconv"

"modules/notifications/model"
"modules/notifications/service"

"github.com/labstack/echo/v4"
)

type NotificationHandler struct {
notifService *service.NotificationService
}

func NewNotificationHandler(notifService *service.NotificationService) *NotificationHandler {
return &NotificationHandler{
notifService: notifService,
}
}

// GetNotifications kullanıcının bildirimlerini getir
func (h *NotificationHandler) GetNotifications(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1)

limit := 50
if limitStr := c.QueryParam("limit"); limitStr != "" {
if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
limit = val
}
}

notifications, err := h.notifService.GetNotifications(userID, limit)
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, notifications)
}

// GetUnreadCount okunmamış bildirim sayısını getir
func (h *NotificationHandler) GetUnreadCount(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1)

count, err := h.notifService.GetUnreadCount(userID)
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, map[string]interface{}{
"unread_count": count,
})
}

// MarkAsRead bildirimi okundu olarak işaretle
func (h *NotificationHandler) MarkAsRead(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1)

id, err := strconv.ParseUint(c.Param("id"), 10, 32)
if err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": "Invalid notification ID",
})
}

if err := h.notifService.MarkAsRead(uint(id)); err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, map[string]string{
"message": "Notification marked as read",
})
}

// MarkAllAsRead tüm bildirimleri okundu olarak işaretle
func (h *NotificationHandler) MarkAllAsRead(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1)

if err := h.notifService.MarkAllAsRead(userID); err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, map[string]string{
"message": "All notifications marked as read",
})
}

// GetPreferences bildirim tercihlerini getir
func (h *NotificationHandler) GetPreferences(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1)

prefs, err := h.notifService.GetPreferences(userID)
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, prefs)
}

// UpdatePreferences bildirim tercihlerini güncelle
func (h *NotificationHandler) UpdatePreferences(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1)

var req model.UpdatePreferenceRequest
if err := c.Bind(&req); err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": "Invalid request",
})
}

prefs, err := h.notifService.UpdatePreferences(userID, &req)
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, prefs)
}
