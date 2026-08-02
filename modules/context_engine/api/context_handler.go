package api

import (
"net/http"
"strconv"

"modules/context_engine/model"
"modules/context_engine/service"

"github.com/labstack/echo/v4"
)

type ContextHandler struct {
contextService *service.ContextService
}

func NewContextHandler(contextService *service.ContextService) *ContextHandler {
return &ContextHandler{
contextService: contextService,
}
}

// CreateSession yeni oturum oluÅŸturur
func (h *ContextHandler) CreateSession(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1)

var req model.CreateSessionRequest
if err := c.Bind(&req); err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": "Invalid request",
})
}

session, err := h.contextService.CreateSession(userID, &req)
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusCreated, session)
}

// GetSessions kullanÄ±cÄ±nÄ±n tÃ¼m oturumlarÄ±nÄ± getirir
func (h *ContextHandler) GetSessions(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1)

sessions, err := h.contextService.GetSessions(userID)
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, sessions)
}

// GetSession oturum bilgilerini getirir
func (h *ContextHandler) GetSession(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1)

sessionID := c.Param("session_id")
session, err := h.contextService.GetSession(userID, sessionID)
if err != nil {
return c.JSON(http.StatusNotFound, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, session)
}

// AddMessage oturuma mesaj ekler
func (h *ContextHandler) AddMessage(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1)

sessionID := c.Param("session_id")
var req model.AddMessageRequest
if err := c.Bind(&req); err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": "Invalid request",
})
}

message, err := h.contextService.AddMessage(userID, sessionID, &req)
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusCreated, message)
}

// GetMessages oturumun mesajlarÄ±nÄ± getirir
func (h *ContextHandler) GetMessages(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1)

sessionID := c.Param("session_id")
limit := 50
if limitStr := c.QueryParam("limit"); limitStr != "" {
if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
limit = val
}
}

messages, err := h.contextService.GetMessages(userID, sessionID, limit)
if err != nil {
return c.JSON(http.StatusNotFound, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, messages)
}

// DeleteSession oturumu siler
func (h *ContextHandler) DeleteSession(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1)

sessionID := c.Param("session_id")
if err := h.contextService.DeleteSession(userID, sessionID); err != nil {
return c.JSON(http.StatusNotFound, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, map[string]string{
"message": "Session deleted successfully",
})
}

// ClearContext oturumun context'ini temizler
func (h *ContextHandler) ClearContext(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1)

sessionID := c.Param("session_id")
if err := h.contextService.ClearContext(userID, sessionID); err != nil {
return c.JSON(http.StatusNotFound, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, map[string]string{
"message": "Context cleared successfully",
})
}
