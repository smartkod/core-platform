package context_engine

import (
"modules/context_engine/api"
"modules/context_engine/model"
"modules/context_engine/service"

"core-platform/core/logger"
"core-platform/core/module"

"github.com/labstack/echo/v4"
"gorm.io/gorm"
)

type ContextEngineModule struct {
*module.BaseModule
db      *gorm.DB
logger  *logger.Logger
handler *api.ContextHandler
service *service.ContextService
router  *echo.Echo
}

func NewContextEngineModule(db *gorm.DB, logger *logger.Logger, echo *echo.Echo) *ContextEngineModule {
base := module.NewBaseModule("context_engine", "1.0.0", module.StateEnabled, logger)
base.SetDependsOn([]string{"auth", "database"})

return &ContextEngineModule{
BaseModule: base,
db:         db,
logger:     logger,
router:     echo,
}
}

func (m *ContextEngineModule) Initialize() error {
if err := m.BaseModule.Initialize(); err != nil {
return err
}

m.service = service.NewContextService(m.db)
m.handler = api.NewContextHandler(m.service)

m.logger.Info("Context Engine module initialized")
return nil
}

func (m *ContextEngineModule) Start() error {
if err := m.BaseModule.Start(); err != nil {
return err
}

contextGroup := m.router.Group("/api/context")
contextGroup.POST("/sessions", m.handler.CreateSession)
contextGroup.GET("/sessions", m.handler.GetSessions)
contextGroup.GET("/sessions/:session_id", m.handler.GetSession)
contextGroup.DELETE("/sessions/:session_id", m.handler.DeleteSession)
contextGroup.POST("/sessions/:session_id/messages", m.handler.AddMessage)
contextGroup.GET("/sessions/:session_id/messages", m.handler.GetMessages)
contextGroup.DELETE("/sessions/:session_id/context", m.handler.ClearContext)

m.logger.Info("Context Engine module started - routes registered")
return nil
}

func (m *ContextEngineModule) Stop() error {
m.logger.Info("Context Engine module stopped")
return m.BaseModule.Stop()
}

func (m *ContextEngineModule) Models() []interface{} {
return []interface{}{
&model.ContextSession{},
&model.ContextMessage{},
}
}
