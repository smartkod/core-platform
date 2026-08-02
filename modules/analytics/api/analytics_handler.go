package api

import (
"net/http"

"modules/analytics/service"

"github.com/labstack/echo/v4"
)

type AnalyticsHandler struct {
analyticsService *service.AnalyticsService
}

func NewAnalyticsHandler(analyticsService *service.AnalyticsService) *AnalyticsHandler {
return &AnalyticsHandler{
analyticsService: analyticsService,
}
}

// GetDashboard dashboard verilerini getirir
func (h *AnalyticsHandler) GetDashboard(c echo.Context) error {
dashboard, err := h.analyticsService.GetDashboard()
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}
return c.JSON(http.StatusOK, dashboard)
}
