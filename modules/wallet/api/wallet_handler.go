package api

import (
"net/http"
"strconv"

"modules/wallet/model"
"modules/wallet/service"

"github.com/labstack/echo/v4"
)

type WalletHandler struct {
walletService *service.WalletService
}

func NewWalletHandler(walletService *service.WalletService) *WalletHandler {
return &WalletHandler{
walletService: walletService,
}
}

// GetWallet cÃ¼zdan bilgilerini getirir
func (h *WalletHandler) GetWallet(c echo.Context) error {
userID := uint(1) // TODO: JWT'den al

wallet, err := h.walletService.GetOrCreateWallet(userID)
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, wallet)
}

// GetBalance bakiyeyi getirir
func (h *WalletHandler) GetBalance(c echo.Context) error {
userID := uint(1) // TODO: JWT'den al

balance, err := h.walletService.GetBalance(userID)
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, map[string]interface{}{
"balance": balance,
})
}

// EarnToken jeton kazandÄ±rÄ±r
func (h *WalletHandler) EarnToken(c echo.Context) error {
userID := uint(1) // TODO: JWT'den al

var req model.EarnRequest
if err := c.Bind(&req); err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": "Invalid request",
})
}

tx, err := h.walletService.EarnToken(userID, &req)
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, tx)
}

// SpendToken jeton harcar
func (h *WalletHandler) SpendToken(c echo.Context) error {
userID := uint(1) // TODO: JWT'den al

var req model.SpendRequest
if err := c.Bind(&req); err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": "Invalid request",
})
}

tx, err := h.walletService.SpendToken(userID, &req)
if err != nil {
if err.Error() == "insufficient balance" {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": "Yetersiz bakiye",
})
}
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, tx)
}

// GetTransactions iÅŸlem geÃ§miÅŸini getirir
func (h *WalletHandler) GetTransactions(c echo.Context) error {
userID := uint(1) // TODO: JWT'den al

limit := 50
if limitStr := c.QueryParam("limit"); limitStr != "" {
if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
limit = val
}
}

transactions, err := h.walletService.GetTransactions(userID, limit)
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, transactions)
}

// ClaimDailyBonus gÃ¼nlÃ¼k bonus al
func (h *WalletHandler) ClaimDailyBonus(c echo.Context) error {
userID := uint(1) // TODO: JWT'den al

tx, err := h.walletService.ClaimDailyBonus(userID)
if err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, tx)
}
