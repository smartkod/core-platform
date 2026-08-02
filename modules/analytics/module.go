package analytics

import (
"modules/analytics/api"
"modules/analytics/model"
"modules/analytics/service"

"core-platform/core/logger"
"core-platform/core/module"

"github.com/labstack/echo/v4"
"gorm.io/gorm"
)

type AnalyticsModule struct {
*module.BaseModule
db      *gorm.DB
logger  *logger.Logger
handler *api.AnalyticsHandler
service *service.AnalyticsService
router  *echo.Echo
}

func NewAnalyticsModule(db *gorm.DB, logger *logger.Logger, echo *echo.Echo) *AnalyticsModule {
base := module.NewBaseModule("analytics", "1.0.0", module.StateEnabled, logger)
base.SetDependsOn([]string{"auth", "database"})

return &AnalyticsModule{
BaseModule: base,
db:         db,
logger:     logger,
router:     echo,
}
}

func (m *AnalyticsModule) Initialize() error {
if err := m.BaseModule.Initialize(); err != nil {
return err
}

m.service = service.NewAnalyticsService(m.db)
m.handler = api.NewAnalyticsHandler(m.service)

m.logger.Info("Analytics module initialized")
return nil
}

func (m *AnalyticsModule) Start() error {
if err := m.BaseModule.Start(); err != nil {
return err
}

analyticsGroup := m.router.Group("/api/analytics")
analyticsGroup.GET("/dashboard", m.handler.GetDashboard)

m.logger.Info("Analytics module started - routes registered")
return nil
}

func (m *AnalyticsModule) Stop() error {
m.logger.Info("Analytics module stopped")
return m.BaseModule.Stop()
}

func (m *AnalyticsModule) Models() []interface{} {
return []interface{}{
&model.UserActivity{},
&model.DailyStats{},
}
}
