package prompt_engine

import (
"modules/prompt_engine/api"
"modules/prompt_engine/model"
"modules/prompt_engine/service"

"core-platform/core/logger"
"core-platform/core/module"

"github.com/labstack/echo/v4"
"gorm.io/gorm"
)

type PromptEngineModule struct {
*module.BaseModule
db      *gorm.DB
logger  *logger.Logger
handler *api.PromptHandler
service *service.PromptService
router  *echo.Echo
}

func NewPromptEngineModule(db *gorm.DB, logger *logger.Logger, echo *echo.Echo) *PromptEngineModule {
base := module.NewBaseModule("prompt_engine", "1.0.0", module.StateEnabled, logger)
base.SetDependsOn([]string{"auth", "database"})

return &PromptEngineModule{
BaseModule: base,
db:         db,
logger:     logger,
router:     echo,
}
}

func (m *PromptEngineModule) Initialize() error {
if err := m.BaseModule.Initialize(); err != nil {
return err
}

m.service = service.NewPromptService(m.db)
m.handler = api.NewPromptHandler(m.service)

m.logger.Info("Prompt Engine module initialized")
return nil
}

func (m *PromptEngineModule) Start() error {
if err := m.BaseModule.Start(); err != nil {
return err
}

promptGroup := m.router.Group("/api/prompts")
promptGroup.POST("", m.handler.CreatePrompt)
promptGroup.POST("/execute", m.handler.ExecutePrompt)
promptGroup.GET("", m.handler.GetTemplates)
promptGroup.GET("/:name", m.handler.GetTemplateByName)
promptGroup.DELETE("/:name", m.handler.DeleteTemplate)

m.logger.Info("Prompt Engine module started - routes registered")
return nil
}

func (m *PromptEngineModule) Stop() error {
m.logger.Info("Prompt Engine module stopped")
return m.BaseModule.Stop()
}

func (m *PromptEngineModule) Models() []interface{} {
return []interface{}{
&model.PromptTemplate{},
&model.PromptExecution{},
}
}
