package notifications

import (
"modules/notifications/api"
"modules/notifications/model"
"modules/notifications/service"

"core-platform/core/logger"
"core-platform/core/module"

"github.com/labstack/echo/v4"
"gorm.io/gorm"
)

type NotificationsModule struct {
*module.BaseModule
db      *gorm.DB
logger  *logger.Logger
handler *api.NotificationHandler
service *service.NotificationService
router  *echo.Echo
}

func NewNotificationsModule(db *gorm.DB, logger *logger.Logger, echo *echo.Echo) *NotificationsModule {
base := module.NewBaseModule("notifications", "1.0.0", module.StateEnabled, logger)
base.SetDependsOn([]string{"auth", "database"})

return &NotificationsModule{
BaseModule: base,
db:         db,
logger:     logger,
router:     echo,
}
}

func (m *NotificationsModule) Initialize() error {
if err := m.BaseModule.Initialize(); err != nil {
return err
}

m.service = service.NewNotificationService(m.db)
m.handler = api.NewNotificationHandler(m.service)

m.logger.Info("Notifications module initialized")
return nil
}

func (m *NotificationsModule) Start() error {
if err := m.BaseModule.Start(); err != nil {
return err
}

notifGroup := m.router.Group("/api/notifications")
notifGroup.GET("", m.handler.GetNotifications)
notifGroup.GET("/unread", m.handler.GetUnreadCount)
notifGroup.PUT("/:id/read", m.handler.MarkAsRead)
notifGroup.PUT("/read-all", m.handler.MarkAllAsRead)
notifGroup.GET("/preferences", m.handler.GetPreferences)
notifGroup.PUT("/preferences", m.handler.UpdatePreferences)

m.logger.Info("Notifications module started - routes registered")
return nil
}

func (m *NotificationsModule) Stop() error {
m.logger.Info("Notifications module stopped")
return m.BaseModule.Stop()
}

func (m *NotificationsModule) Models() []interface{} {
return []interface{}{
&model.Notification{},
&model.NotificationPreference{},
}
}
