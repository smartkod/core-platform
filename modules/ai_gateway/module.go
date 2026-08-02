package ai_gateway

import (
"os"

"modules/ai_gateway/api"
"modules/ai_gateway/model"
"modules/ai_gateway/service"

"core-platform/core/logger"
"core-platform/core/module"

"github.com/labstack/echo/v4"
"gorm.io/gorm"
)

type AIGatewayModule struct {
*module.BaseModule
db      *gorm.DB
logger  *logger.Logger
handler *api.AIGatewayHandler
service *service.AIGatewayService
router  *echo.Echo
}

func NewAIGatewayModule(db *gorm.DB, logger *logger.Logger, echo *echo.Echo) *AIGatewayModule {
base := module.NewBaseModule("ai_gateway", "1.0.0", module.StateEnabled, logger)
base.SetDependsOn([]string{"auth", "database"})

// OpenAI API key ve base URL
openAIKey := os.Getenv("OPENAI_API_KEY")
openAIBaseURL := os.Getenv("OPENAI_BASE_URL")
if openAIBaseURL == "" {
openAIBaseURL = "https://api.openai.com"
}

return &AIGatewayModule{
BaseModule: base,
db:         db,
logger:     logger,
router:     echo,
service:    service.NewAIGatewayService(db, openAIKey, openAIBaseURL),
}
}

func (m *AIGatewayModule) Initialize() error {
if err := m.BaseModule.Initialize(); err != nil {
return err
}

m.handler = api.NewAIGatewayHandler(m.service)

m.logger.Info("AI Gateway module initialized")
return nil
}

func (m *AIGatewayModule) Start() error {
if err := m.BaseModule.Start(); err != nil {
return err
}

aiGroup := m.router.Group("/api/ai")
aiGroup.POST("/chat", m.handler.Chat)
aiGroup.GET("/providers", m.handler.GetProviders)

m.logger.Info("AI Gateway module started - routes registered")
return nil
}

func (m *AIGatewayModule) Stop() error {
m.logger.Info("AI Gateway module stopped")
return m.BaseModule.Stop()
}

func (m *AIGatewayModule) Models() []interface{} {
return []interface{}{
&model.AIProvider{},
&model.AICache{},
&model.AIUsage{},
}
}
