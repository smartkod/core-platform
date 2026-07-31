package rewarded_ads

import (
"modules/rewarded_ads/model"

"core-platform/core/logger"
"core-platform/core/module"

"github.com/labstack/echo/v4"
"gorm.io/gorm"
)

type RewardedAdsModule struct {
*module.BaseModule
db     *gorm.DB
logger *logger.Logger
router *echo.Echo
}

func NewRewardedAdsModule(db *gorm.DB, logger *logger.Logger, echo *echo.Echo) *RewardedAdsModule {
base := module.NewBaseModule("rewarded_ads", "1.0.0", module.StateEnabled, logger)
base.SetDependsOn([]string{"auth", "database", "wallet"})

return &RewardedAdsModule{
BaseModule: base,
db:         db,
logger:     logger,
router:     echo,
}
}

func (m *RewardedAdsModule) Initialize() error {
if err := m.BaseModule.Initialize(); err != nil {
return err
}
m.logger.Info("Rewarded Ads module initialized")
return nil
}

func (m *RewardedAdsModule) Start() error {
if err := m.BaseModule.Start(); err != nil {
return err
}

adGroup := m.router.Group("/api/ads")
adGroup.GET("/status", func(c echo.Context) error {
return c.JSON(200, map[string]interface{}{
"can_watch":        true,
"watched_today":    3,
"remaining_today":  2,
"token_per_ad":     10,
"daily_limit":      5,
"daily_token_limit": 50,
})
})
adGroup.POST("/watch", func(c echo.Context) error {
return c.JSON(200, map[string]interface{}{
"success":      true,
"message":      "Reklam izlendi! 10 jeton kazandiniz.",
"reward_value": 10,
"total_today":  4,
"remaining":    1,
})
})

m.logger.Info("Rewarded Ads module started - routes registered")
return nil
}

func (m *RewardedAdsModule) Stop() error {
m.logger.Info("Rewarded Ads module stopped")
return m.BaseModule.Stop()
}

func (m *RewardedAdsModule) Models() []interface{} {
return []interface{}{
&model.AdReward{},
&model.DailyAdLimit{},
}
}
