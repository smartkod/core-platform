package lifeweave

import (
"modules/lifeweave/api"
"modules/lifeweave/model"
"modules/lifeweave/service"

"core-platform/core/logger"
"core-platform/core/module"

"github.com/labstack/echo/v4"
"gorm.io/gorm"
)

type LifeWeaveModule struct {
*module.BaseModule
db      *gorm.DB
logger  *logger.Logger
handler *api.LifeWeaveHandler
service *service.LifeWeaveService
router  *echo.Echo
}

func NewLifeWeaveModule(db *gorm.DB, logger *logger.Logger, echo *echo.Echo) *LifeWeaveModule {
base := module.NewBaseModule("lifeweave", "1.0.0", module.StateEnabled, logger)
base.SetDependsOn([]string{"auth", "database"})

return &LifeWeaveModule{
BaseModule: base,
db:         db,
logger:     logger,
router:     echo,
}
}

func (m *LifeWeaveModule) Initialize() error {
if err := m.BaseModule.Initialize(); err != nil {
return err
}

m.service = service.NewLifeWeaveService(m.db)
m.handler = api.NewLifeWeaveHandler(m.service)

m.logger.Info("LifeWeave module initialized")
return nil
}

func (m *LifeWeaveModule) Start() error {
if err := m.BaseModule.Start(); err != nil {
return err
}

lwGroup := m.router.Group("/api/lifeweave")
lwGroup.POST("/events", m.handler.RecordEvent)
lwGroup.GET("/events", m.handler.GetEvents)
lwGroup.GET("/events/:id", m.handler.GetEvent)
lwGroup.PUT("/events/:id", m.handler.UpdateEvent)
lwGroup.DELETE("/events/:id", m.handler.DeleteEvent)
lwGroup.POST("/events/:id/verify", m.handler.VerifyEvent)
lwGroup.GET("/timeline", m.handler.GetTimeline)

m.logger.Info("LifeWeave module started - routes registered")
return nil
}

func (m *LifeWeaveModule) Stop() error {
m.logger.Info("LifeWeave module stopped")
return m.BaseModule.Stop()
}

func (m *LifeWeaveModule) Models() []interface{} {
return []interface{}{
&model.LifeEvent{},
}
}
