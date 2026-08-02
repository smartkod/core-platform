package ai_copilot

import (
"modules/ai_copilot/api"
"modules/ai_copilot/model"
"modules/ai_copilot/service"

"core-platform/core/logger"
"core-platform/core/module"

"github.com/labstack/echo/v4"
"gorm.io/gorm"
)

type AICopilotModule struct {
*module.BaseModule
db      *gorm.DB
logger  *logger.Logger
handler *api.CopilotHandler
service *service.CopilotService
router  *echo.Echo
}

func NewAICopilotModule(db *gorm.DB, logger *logger.Logger, echo *echo.Echo) *AICopilotModule {
base := module.NewBaseModule("ai_copilot", "1.0.0", module.StateEnabled, logger)
base.SetDependsOn([]string{"auth", "database", "ai_gateway"})

return &AICopilotModule{
BaseModule: base,
db:         db,
logger:     logger,
router:     echo,
}
}

func (m *AICopilotModule) Initialize() error {
if err := m.BaseModule.Initialize(); err != nil {
return err
}

m.service = service.NewCopilotService(m.db)
m.handler = api.NewCopilotHandler(m.service)

m.logger.Info("AI Copilot module initialized")
return nil
}

func (m *AICopilotModule) Start() error {
if err := m.BaseModule.Start(); err != nil {
return err
}

copilotGroup := m.router.Group("/api/copilot")
copilotGroup.POST("/ask", m.handler.Ask)
copilotGroup.GET("/history", m.handler.GetHistory)

m.logger.Info("AI Copilot module started - routes registered")
return nil
}

func (m *AICopilotModule) Stop() error {
m.logger.Info("AI Copilot module stopped")
return m.BaseModule.Stop()
}

func (m *AICopilotModule) Models() []interface{} {
return []interface{}{
&model.CopilotSession{},
&model.CopilotMessage{},
}
}
