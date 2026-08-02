package users

import (
"modules/users/api"
"modules/users/model"
"modules/users/service"

"core-platform/core/logger"
"core-platform/core/module"

"github.com/labstack/echo/v4"
"gorm.io/gorm"
)

// UsersModule users modÃ¼lÃ¼
type UsersModule struct {
*module.BaseModule
db          *gorm.DB
logger      *logger.Logger
handler     *api.UserHandler
userService *service.UserService
router      *echo.Echo
}

// NewUsersModule yeni users modÃ¼lÃ¼ oluÅŸturur
func NewUsersModule(db *gorm.DB, logger *logger.Logger, echo *echo.Echo) *UsersModule {
base := module.NewBaseModule("users", "1.0.0", module.StateEnabled, logger)
base.SetDependsOn([]string{"auth", "database"})

return &UsersModule{
BaseModule: base,
db:         db,
logger:     logger,
router:     echo,
}
}

// Initialize modÃ¼lÃ¼ baÅŸlatÄ±r
func (m *UsersModule) Initialize() error {
if err := m.BaseModule.Initialize(); err != nil {
return err
}

// Servis ve handler'Ä± oluÅŸtur
m.userService = service.NewUserService(m.db)
m.handler = api.NewUserHandler(m.userService)

m.logger.Info("Users module initialized")
return nil
}

// Start modÃ¼lÃ¼ baÅŸlatÄ±r ve route'larÄ± ekler
func (m *UsersModule) Start() error {
if err := m.BaseModule.Start(); err != nil {
return err
}

// Route'larÄ± ekle
userGroup := m.router.Group("/api/users")
userGroup.GET("/me", m.handler.GetMe)
userGroup.PUT("/me", m.handler.UpdateMe)
userGroup.GET("", m.handler.GetAllUsers)
userGroup.GET("/:id", m.handler.GetUserByID)
userGroup.DELETE("/:id", m.handler.DeleteUser)

m.logger.Info("Users module started - routes registered")
return nil
}

// Stop modÃ¼lÃ¼ durdurur
func (m *UsersModule) Stop() error {
m.logger.Info("Users module stopped")
return m.BaseModule.Stop()
}

// User modelini dÄ±ÅŸa aktar
type User = model.User
