package auth

import (
"core-platform/modules/auth/api"
"core-platform/modules/auth/model"
"core-platform/modules/auth/service"

coreapi "core-platform/core/api"
"core-platform/core/logger"
"core-platform/core/module"

"go.uber.org/zap"
"gorm.io/gorm"
)

// AuthModule auth modülü
type AuthModule struct {
*module.BaseModule
db          *gorm.DB
logger      *logger.Logger
handler     *api.AuthHandler
authService *service.AuthService
router      *coreapi.Router
jwtSecret   string
}

// NewAuthModule yeni auth modülü oluşturur
func NewAuthModule(db *gorm.DB, logger *logger.Logger, router *coreapi.Router, jwtSecret string) *AuthModule {
base := module.NewBaseModule("auth", "1.0.0", module.StateEnabled, logger)
base.SetDependsOn([]string{})

return &AuthModule{
BaseModule: base,
db:         db,
logger:     logger,
router:     router,
jwtSecret:  jwtSecret,
}
}

// Initialize modülü başlatır
func (m *AuthModule) Initialize() error {
if err := m.BaseModule.Initialize(); err != nil {
return err
}

// Servis ve handler'ı oluştur
m.authService = service.NewAuthService(m.db, m.jwtSecret)
m.handler = api.NewAuthHandler(m.authService)

m.logger.Info("Auth module initialized")
return nil
}

// Start modülü başlatır ve route'ları ekler
func (m *AuthModule) Start() error {
m.logger.Info("Auth module Start() called - registering routes...")

if err := m.BaseModule.Start(); err != nil {
m.logger.Error("BaseModule.Start() failed", zap.Error(err))
return err
}

// Route'ları ekle - doğrudan echo instance'ına
echo := m.router.GetEcho()
if echo == nil {
m.logger.Error("Echo instance is nil!")
return nil
}

m.logger.Info("Echo instance found, registering auth routes...")

authGroup := echo.Group("/api/auth")
authGroup.POST("/register", m.handler.Register)
authGroup.POST("/login", m.handler.Login)

m.logger.Info("Auth routes registered: /api/auth/register, /api/auth/login")

// Debug: Tüm route'ları listele
m.logger.Info("Current routes:")
for _, route := range echo.Routes() {
m.logger.Info("Route",
zap.String("method", route.Method),
zap.String("path", route.Path),
)
}

return nil
}

// Stop modülü durdurur
func (m *AuthModule) Stop() error {
m.logger.Info("Auth module stopped")
return m.BaseModule.Stop()
}

// User modelini dışa aktar
type User = model.User
