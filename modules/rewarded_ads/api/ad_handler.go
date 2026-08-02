package api

import (
"net/http"

"modules/rewarded_ads/model"
"modules/rewarded_ads/service"

"github.com/labstack/echo/v4"
)

type AdHandler struct {
adService *service.AdService
}

func NewAdHandler(adService *service.AdService) *AdHandler {
return &AdHandler{
adService: adService,
}
}

// GetAdStatus reklam durumunu getirir
func (h *AdHandler) GetAdStatus(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1)

status, err := h.adService.GetAdStatus(userID)
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, status)
}

// WatchAd reklam izleme
func (h *AdHandler) WatchAd(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1)

var req model.AdWatchRequest
if err := c.Bind(&req); err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": "Invalid request",
})
}

response, err := h.adService.WatchAd(userID, &req)
if err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, response)
}
