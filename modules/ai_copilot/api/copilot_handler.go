package api

import (
"net/http"

"modules/ai_copilot/model"
"modules/ai_copilot/service"

"github.com/labstack/echo/v4"
)

type CopilotHandler struct {
copilotService *service.CopilotService
}

func NewCopilotHandler(copilotService *service.CopilotService) *CopilotHandler {
return &CopilotHandler{
copilotService: copilotService,
}
}

// Ask soru sor
func (h *CopilotHandler) Ask(c echo.Context) error {
userID := uint(1) // TODO: JWT'den al

var req model.CopilotRequest
if err := c.Bind(&req); err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": "Invalid request",
})
}

if req.Query == "" {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": "Query is required",
})
}

response, err := h.copilotService.Ask(userID, &req)
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, response)
}

// GetHistory sohbet geÃ§miÅŸini getir
func (h *CopilotHandler) GetHistory(c echo.Context) error {
userID := uint(1) // TODO: JWT'den al

limit := 20
// TODO: Query param'dan limit al

messages, err := h.copilotService.GetHistory(userID, limit)
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, messages)
}
