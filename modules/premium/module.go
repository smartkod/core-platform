package premium

import (
"modules/premium/model"

"core-platform/core/logger"
"core-platform/core/module"

"github.com/labstack/echo/v4"
"gorm.io/gorm"
)

type PremiumModule struct {
*module.BaseModule
db     *gorm.DB
logger *logger.Logger
router *echo.Echo
}

func NewPremiumModule(db *gorm.DB, logger *logger.Logger, echo *echo.Echo) *PremiumModule {
base := module.NewBaseModule("premium", "1.0.0", module.StateEnabled, logger)
base.SetDependsOn([]string{"auth", "database", "wallet"})

return &PremiumModule{
BaseModule: base,
db:         db,
logger:     logger,
router:     echo,
}
}

func (m *PremiumModule) Initialize() error {
if err := m.BaseModule.Initialize(); err != nil {
return err
}
m.logger.Info("Premium module initialized")
return nil
}

func (m *PremiumModule) Start() error {
if err := m.BaseModule.Start(); err != nil {
return err
}

premiumGroup := m.router.Group("/api/premium")
premiumGroup.GET("/plans", func(c echo.Context) error {
return c.JSON(200, []map[string]interface{}{
{"id": 1, "name": "Aylik", "price": 49.90, "currency": "TRY", "duration": 30},
{"id": 2, "name": "Yillik", "price": 499.90, "currency": "TRY", "duration": 365},
{"id": 3, "name": "Omur Boyu", "price": 999.90, "currency": "TRY", "duration": 0},
})
})
premiumGroup.GET("/me", func(c echo.Context) error {
return c.JSON(200, map[string]interface{}{
"is_active":      false,
"plan_name":      "",
"days_remaining": 0,
"is_lifetime":    false,
})
})
premiumGroup.POST("/purchase", func(c echo.Context) error {
var req map[string]interface{}
if err := c.Bind(&req); err != nil {
return c.JSON(400, map[string]string{"error": "Invalid request"})
}
return c.JSON(200, map[string]interface{}{
"is_active":      true,
"plan_id":        req["plan_id"],
"message":        "Premium satin alindi!",
"days_remaining": 30,
})
})
premiumGroup.POST("/cancel", func(c echo.Context) error {
return c.JSON(200, map[string]string{
"message": "Premium aboneligi iptal edildi.",
})
})

m.logger.Info("Premium module started - routes registered")
return nil
}

func (m *PremiumModule) Stop() error {
m.logger.Info("Premium module stopped")
return m.BaseModule.Stop()
}

func (m *PremiumModule) Models() []interface{} {
return []interface{}{
&model.PremiumPlan{},
&model.UserPremium{},
}
}
