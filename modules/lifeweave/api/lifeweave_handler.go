package api

import (
"net/http"
"strconv"

"modules/lifeweave/model"
"modules/lifeweave/service"

"github.com/labstack/echo/v4"
)

type LifeWeaveHandler struct {
lifeweaveService *service.LifeWeaveService
}

func NewLifeWeaveHandler(lifeweaveService *service.LifeWeaveService) *LifeWeaveHandler {
return &LifeWeaveHandler{
lifeweaveService: lifeweaveService,
}
}

// RecordEvent olay kaydeder
func (h *LifeWeaveHandler) RecordEvent(c echo.Context) error {
userID := uint(1) // TODO: JWT'den al

var req model.LifeEvent
if err := c.Bind(&req); err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": "Invalid request",
})
}

req.UserID = userID
event, err := h.lifeweaveService.RecordEvent(userID, &req)
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusCreated, event)
}

// GetEvents kullanıcının olaylarını getirir
func (h *LifeWeaveHandler) GetEvents(c echo.Context) error {
userID := uint(1) // TODO: JWT'den al

events, err := h.lifeweaveService.GetUserEvents(userID)
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, events)
}

// GetEvent olay detayını getirir
func (h *LifeWeaveHandler) GetEvent(c echo.Context) error {
id, err := strconv.ParseUint(c.Param("id"), 10, 32)
if err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": "Invalid event ID",
})
}

event, err := h.lifeweaveService.GetEventByID(uint(id))
if err != nil {
return c.JSON(http.StatusNotFound, map[string]string{
"error": "Event not found",
})
}

return c.JSON(http.StatusOK, event)
}

// UpdateEvent olayı günceller
func (h *LifeWeaveHandler) UpdateEvent(c echo.Context) error {
id, err := strconv.ParseUint(c.Param("id"), 10, 32)
if err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": "Invalid event ID",
})
}

var req model.LifeEvent
if err := c.Bind(&req); err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": "Invalid request",
})
}

event, err := h.lifeweaveService.UpdateEvent(uint(id), &req)
if err != nil {
return c.JSON(http.StatusNotFound, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, event)
}

// DeleteEvent olayı siler
func (h *LifeWeaveHandler) DeleteEvent(c echo.Context) error {
id, err := strconv.ParseUint(c.Param("id"), 10, 32)
if err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": "Invalid event ID",
})
}

if err := h.lifeweaveService.DeleteEvent(uint(id)); err != nil {
return c.JSON(http.StatusNotFound, map[string]string{
"error": "Event not found",
})
}

return c.JSON(http.StatusOK, map[string]string{
"message": "Event deleted successfully",
})
}

// VerifyEvent olayı doğrula
func (h *LifeWeaveHandler) VerifyEvent(c echo.Context) error {
id, err := strconv.ParseUint(c.Param("id"), 10, 32)
if err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": "Invalid event ID",
})
}

feedback := c.QueryParam("feedback")
if feedback == "" {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": "Feedback parameter is required (accurate, partial, inaccurate)",
})
}

if err := h.lifeweaveService.VerifyEvent(uint(id), feedback); err != nil {
return c.JSON(http.StatusNotFound, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, map[string]string{
"message": "Event verified successfully",
})
}

// GetTimeline kullanıcının zaman çizelgesini getirir
func (h *LifeWeaveHandler) GetTimeline(c echo.Context) error {
userID := uint(1) // TODO: JWT'den al

timeline, err := h.lifeweaveService.GetTimeline(userID)
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, timeline)
}
