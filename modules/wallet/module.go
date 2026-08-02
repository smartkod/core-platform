package wallet

import (
"modules/wallet/api"
"modules/wallet/model"
"modules/wallet/service"

"core-platform/core/logger"
"core-platform/core/module"

"github.com/labstack/echo/v4"
"gorm.io/gorm"
)

type WalletModule struct {
*module.BaseModule
db      *gorm.DB
logger  *logger.Logger
handler *api.WalletHandler
service *service.WalletService
router  *echo.Echo
}

func NewWalletModule(db *gorm.DB, logger *logger.Logger, echo *echo.Echo) *WalletModule {
base := module.NewBaseModule("wallet", "1.0.0", module.StateEnabled, logger)
base.SetDependsOn([]string{"auth", "database"})

return &WalletModule{
BaseModule: base,
db:         db,
logger:     logger,
router:     echo,
}
}

func (m *WalletModule) Initialize() error {
if err := m.BaseModule.Initialize(); err != nil {
return err
}

m.service = service.NewWalletService(m.db)
m.handler = api.NewWalletHandler(m.service)

m.logger.Info("Wallet module initialized")
return nil
}

func (m *WalletModule) Start() error {
if err := m.BaseModule.Start(); err != nil {
return err
}

walletGroup := m.router.Group("/api/wallet")
walletGroup.GET("", m.handler.GetWallet)
walletGroup.GET("/balance", m.handler.GetBalance)
walletGroup.POST("/earn", m.handler.EarnToken)
walletGroup.POST("/spend", m.handler.SpendToken)
walletGroup.GET("/transactions", m.handler.GetTransactions)
walletGroup.POST("/daily-bonus", m.handler.ClaimDailyBonus)

m.logger.Info("Wallet module started - routes registered")
return nil
}

func (m *WalletModule) Stop() error {
m.logger.Info("Wallet module stopped")
return m.BaseModule.Stop()
}

func (m *WalletModule) Models() []interface{} {
return []interface{}{
&model.Wallet{},
&model.WalletTransaction{},
}
}
