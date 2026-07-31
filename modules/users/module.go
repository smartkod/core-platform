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

// UsersModule users modülü
type UsersModule struct {
*module.BaseModule
db          *gorm.DB
logger      *logger.Logger
handler     *api.UserHandler
userService *service.UserService
router      *echo.Echo
}

// NewUsersModule yeni users modülü oluşturur
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

// Initialize modülü başlatır
func (m *UsersModule) Initialize() error {
if err := m.BaseModule.Initialize(); err != nil {
return err
}

// Servis ve handler'ı oluştur
m.userService = service.NewUserService(m.db)
m.handler = api.NewUserHandler(m.userService)

m.logger.Info("Users module initialized")
return nil
}

// Start modülü başlatır ve route'ları ekler
func (m *UsersModule) Start() error {
if err := m.BaseModule.Start(); err != nil {
return err
}

// Route'ları ekle
userGroup := m.router.Group("/api/users")
userGroup.GET("/me", m.handler.GetMe)
userGroup.PUT("/me", m.handler.UpdateMe)
userGroup.GET("", m.handler.GetAllUsers)
userGroup.GET("/:id", m.handler.GetUserByID)
userGroup.DELETE("/:id", m.handler.DeleteUser)

m.logger.Info("Users module started - routes registered")
return nil
}

// Stop modülü durdurur
func (m *UsersModule) Stop() error {
m.logger.Info("Users module stopped")
return m.BaseModule.Stop()
}

// User modelini dışa aktar
type User = model.User
