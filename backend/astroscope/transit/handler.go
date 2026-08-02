package transit

import (
"net/http"

"github.com/labstack/echo/v4"
)

type TransitHandler struct {
service *TransitService
}

func NewTransitHandler(service *TransitService) *TransitHandler {
return &TransitHandler{service: service}
}

// GetDaily gÃ¼nlÃ¼k transitleri getirir
func (h *TransitHandler) GetDaily(c echo.Context) error {
userID := uint(1) // TODO: JWT'den al

req := &TransitRequest{
Date: c.QueryParam("date"),
}
if req.Date == "" {
req.Date = time.Now().Format("2006-01-02")
}

response, err := h.service.GetDailyTransits(userID, req)
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, response)
}
