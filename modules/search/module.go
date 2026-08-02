package search

import (
"modules/search/api"
"modules/search/model"
"modules/search/service"

"core-platform/core/logger"
"core-platform/core/module"

"github.com/labstack/echo/v4"
"gorm.io/gorm"
)

type SearchModule struct {
*module.BaseModule
db      *gorm.DB
logger  *logger.Logger
handler *api.SearchHandler
service *service.SearchService
router  *echo.Echo
}

func NewSearchModule(db *gorm.DB, logger *logger.Logger, echo *echo.Echo) *SearchModule {
base := module.NewBaseModule("search", "1.0.0", module.StateEnabled, logger)
base.SetDependsOn([]string{"auth", "database"})

return &SearchModule{
BaseModule: base,
db:         db,
logger:     logger,
router:     echo,
}
}

func (m *SearchModule) Initialize() error {
if err := m.BaseModule.Initialize(); err != nil {
return err
}

m.service = service.NewSearchService(m.db)
m.handler = api.NewSearchHandler(m.service)

m.logger.Info("Search module initialized")
return nil
}

func (m *SearchModule) Start() error {
if err := m.BaseModule.Start(); err != nil {
return err
}

searchGroup := m.router.Group("/api/search")
searchGroup.GET("", m.handler.Search)
searchGroup.GET("/history", m.handler.GetHistory)
searchGroup.GET("/popular", m.handler.GetPopular)

m.logger.Info("Search module started - routes registered")
return nil
}

func (m *SearchModule) Stop() error {
m.logger.Info("Search module stopped")
return m.BaseModule.Stop()
}

func (m *SearchModule) Models() []interface{} {
return []interface{}{
&model.SearchHistory{},
}
}
