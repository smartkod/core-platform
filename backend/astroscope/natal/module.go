package natal

import (
"core-platform/core/module"
"core-platform/core/logger"

"github.com/labstack/echo/v4"
"gorm.io/gorm"
)

type NatalModule struct {
*module.BaseModule
db      *gorm.DB
logger  *logger.Logger
handler *NatalHandler
service *NatalService
router  *echo.Echo
}

func NewNatalModule(db *gorm.DB, logger *logger.Logger, echo *echo.Echo) *NatalModule {
base := module.NewBaseModule("natal", "1.0.0", module.StateEnabled, logger)
base.SetDependsOn([]string{"auth", "database"})

return &NatalModule{
BaseModule: base,
db:         db,
logger:     logger,
router:     echo,
}
}

func (m *NatalModule) Initialize() error {
if err := m.BaseModule.Initialize(); err != nil {
return err
}

m.service = NewNatalService(m.db)
m.handler = NewNatalHandler(m.service)

m.logger.Info("Natal module initialized")
return nil
}

func (m *NatalModule) Start() error {
if err := m.BaseModule.Start(); err != nil {
return err
}

group := m.router.Group("/api/natal")
group.POST("/calculate", m.handler.Calculate)
group.GET("/my-chart", m.handler.GetByUserID)

m.logger.Info("Natal module started - routes registered")
return nil
}

func (m *NatalModule) Stop() error {
m.logger.Info("Natal module stopped")
return m.BaseModule.Stop()
}
