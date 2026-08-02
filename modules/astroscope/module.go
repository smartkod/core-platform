package astroscope

import (
"modules/astroscope/api"
"modules/astroscope/model"
"modules/astroscope/service"

"core-platform/core/logger"
"core-platform/core/module"

"github.com/labstack/echo/v4"
"gorm.io/gorm"
)

type AstroScopeModule struct {
*module.BaseModule
db      *gorm.DB
logger  *logger.Logger
handler *api.AstrologyHandler
service *service.AstrologyService
router  *echo.Echo
}

func NewAstroScopeModule(db *gorm.DB, logger *logger.Logger, echo *echo.Echo) *AstroScopeModule {
base := module.NewBaseModule("astroscope", "1.0.0", module.StateEnabled, logger)
base.SetDependsOn([]string{"auth", "database", "ai_gateway", "prompt_engine"})

return &AstroScopeModule{
BaseModule: base,
db:         db,
logger:     logger,
router:     echo,
}
}

func (m *AstroScopeModule) Initialize() error {
if err := m.BaseModule.Initialize(); err != nil {
return err
}

m.service = service.NewAstrologyService(m.db)
m.handler = api.NewAstrologyHandler(m.service)

m.logger.Info("AstroScope module initialized")
return nil
}

func (m *AstroScopeModule) Start() error {
if err := m.BaseModule.Start(); err != nil {
return err
}

astroGroup := m.router.Group("/api/astroscope")
astroGroup.GET("/horoscope", m.handler.GetHoroscope)
astroGroup.POST("/compatibility", m.handler.GetCompatibility)
astroGroup.GET("/signs", m.handler.GetSigns)

m.logger.Info("AstroScope module started - routes registered")
return nil
}

func (m *AstroScopeModule) Stop() error {
m.logger.Info("AstroScope module stopped")
return m.BaseModule.Stop()
}

func (m *AstroScopeModule) Models() []interface{} {
return []interface{}{
&model.ZodiacSign{},
&model.Horoscope{},
&model.BirthChart{},
}
}
