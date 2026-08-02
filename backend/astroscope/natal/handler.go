package natal

import (
"net/http"

"github.com/labstack/echo/v4"
)

type NatalHandler struct {
service *NatalService
}

func NewNatalHandler(service *NatalService) *NatalHandler {
return &NatalHandler{service: service}
}

// Calculate doÄŸum haritasÄ± hesaplar
func (h *NatalHandler) Calculate(c echo.Context) error {
userID := uint(1) // TODO: JWT'den al

var req NatalRequest
if err := c.Bind(&req); err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": "GeÃ§ersiz istek",
})
}

response, err := h.service.CalculateNatal(userID, &req)
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, response)
}

// GetByUserID kullanÄ±cÄ±nÄ±n doÄŸum haritasÄ±nÄ± getirir
func (h *NatalHandler) GetByUserID(c echo.Context) error {
userID := uint(1) // TODO: JWT'den al

chart, err := h.service.GetByUserID(userID)
if err != nil {
return c.JSON(http.StatusNotFound, map[string]string{
"error": "DoÄŸum haritasÄ± bulunamadÄ±",
})
}

return c.JSON(http.StatusOK, chart)
}
