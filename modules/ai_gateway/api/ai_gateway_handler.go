package api

import (
"net/http"

"modules/ai_gateway/model"
"modules/ai_gateway/service"

"github.com/labstack/echo/v4"
)

type AIGatewayHandler struct {
aiService *service.AIGatewayService
}

func NewAIGatewayHandler(aiService *service.AIGatewayService) *AIGatewayHandler {
return &AIGatewayHandler{
aiService: aiService,
}
}

// Chat AI ile sohbet
func (h *AIGatewayHandler) Chat(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1)

var req model.AIRequest
if err := c.Bind(&req); err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": "Invalid request",
})
}

// VarsayÄ±lan deÄŸerler
if req.MaxTokens <= 0 {
req.MaxTokens = 1000
}
if req.Temperature <= 0 {
req.Temperature = 0.7
}

response, err := h.aiService.Chat(userID, &req)
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, response)
}

// GetProviders aktif provider'larÄ± getirir
func (h *AIGatewayHandler) GetProviders(c echo.Context) error {
providers, err := h.aiService.GetProviders()
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}
return c.JSON(http.StatusOK, providers)
}
