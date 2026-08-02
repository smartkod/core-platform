package api

import (
"net/http"

"modules/premium/model"
"modules/premium/service"

"github.com/labstack/echo/v4"
)

type PremiumHandler struct {
premiumService *service.PremiumService
}

func NewPremiumHandler(premiumService *service.PremiumService) *PremiumHandler {
return &PremiumHandler{
premiumService: premiumService,
}
}

// GetPlans premium planlarÄ±nÄ± getir
func (h *PremiumHandler) GetPlans(c echo.Context) error {
plans, err := h.premiumService.GetPlans()
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}
return c.JSON(http.StatusOK, plans)
}

// GetMyPremium kullanÄ±cÄ±nÄ±n premium durumunu getir
func (h *PremiumHandler) GetMyPremium(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1)

premium, err := h.premiumService.GetUserPremium(userID)
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}
return c.JSON(http.StatusOK, premium)
}

// PurchasePremium premium satÄ±n al
func (h *PremiumHandler) PurchasePremium(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1)

var req model.PurchaseRequest
if err := c.Bind(&req); err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": "Invalid request",
})
}

premium, err := h.premiumService.PurchasePremium(userID, &req)
if err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": err.Error(),
})
}
return c.JSON(http.StatusOK, premium)
}

// CancelPremium premium iptal et
func (h *PremiumHandler) CancelPremium(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1)

err := h.premiumService.CancelPremium(userID)
if err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": err.Error(),
})
}
return c.JSON(http.StatusOK, map[string]string{
"message": "Premium subscription canceled successfully",
})
}
