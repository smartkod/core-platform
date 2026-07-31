package community

import (
"modules/community/api"
"modules/community/model"

"core-platform/core/logger"
"core-platform/core/module"

"github.com/labstack/echo/v4"
"gorm.io/gorm"
)

type CommunityModule struct {
*module.BaseModule
db      *gorm.DB
logger  *logger.Logger
handler *api.CommunityHandler
router  *echo.Echo
}

func NewCommunityModule(db *gorm.DB, logger *logger.Logger, echo *echo.Echo) *CommunityModule {
base := module.NewBaseModule("community", "1.0.0", module.StateEnabled, logger)
base.SetDependsOn([]string{"auth", "database"})

return &CommunityModule{
BaseModule: base,
db:         db,
logger:     logger,
router:     echo,
}
}

func (m *CommunityModule) Initialize() error {
if err := m.BaseModule.Initialize(); err != nil {
return err
}

m.handler = api.NewCommunityHandler()
m.logger.Info("Community module initialized")
return nil
}

func (m *CommunityModule) Start() error {
if err := m.BaseModule.Start(); err != nil {
return err
}

communityGroup := m.router.Group("/api/community")
communityGroup.POST("/posts", m.handler.CreatePost)
communityGroup.GET("/posts", m.handler.GetPosts)
communityGroup.GET("/posts/:id", m.handler.GetPost)
communityGroup.POST("/posts/:id/comments", m.handler.AddComment)
communityGroup.POST("/posts/:id/like", m.handler.LikePost)
communityGroup.POST("/comments/:id/helpful", m.handler.MarkHelpful)
communityGroup.GET("/patterns", m.handler.GetPatterns)

m.logger.Info("Community module started - routes registered")
return nil
}

func (m *CommunityModule) Stop() error {
m.logger.Info("Community module stopped")
return m.BaseModule.Stop()
}

func (m *CommunityModule) Models() []interface{} {
return []interface{}{
&model.CommunityPost{},
&model.CommunityComment{},
&model.EventPattern{},
}
}
