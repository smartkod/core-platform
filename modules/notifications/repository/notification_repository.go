package repository

import (
"modules/notifications/model"

"gorm.io/gorm"
)

type NotificationRepository struct {
db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
return &NotificationRepository{
db: db,
}
}

// CreateNotification bildirim oluÅŸtur
func (r *NotificationRepository) CreateNotification(notification *model.Notification) error {
return r.db.Create(notification).Error
}

// GetUserNotifications kullanÄ±cÄ±nÄ±n bildirimlerini getir
func (r *NotificationRepository) GetUserNotifications(userID uint, limit int) ([]model.Notification, error) {
var notifications []model.Notification
err := r.db.Where("user_id = ?", userID).
Order("created_at DESC").
Limit(limit).
Find(&notifications).Error
return notifications, err
}

// GetUnreadCount okunmamÄ±ÅŸ bildirim sayÄ±sÄ±nÄ± getir
func (r *NotificationRepository) GetUnreadCount(userID uint) (int64, error) {
var count int64
err := r.db.Model(&model.Notification{}).
Where("user_id = ? AND is_read = ?", userID, false).
Count(&count).Error
return count, err
}

// MarkAsRead bildirimi okundu olarak iÅŸaretle
func (r *NotificationRepository) MarkAsRead(notificationID uint) error {
return r.db.Model(&model.Notification{}).
Where("id = ?", notificationID).
Updates(map[string]interface{}{
"is_read": true,
"read_at": gorm.Expr("NOW()"),
}).Error
}

// MarkAllAsRead tÃ¼m bildirimleri okundu olarak iÅŸaretle
func (r *NotificationRepository) MarkAllAsRead(userID uint) error {
return r.db.Model(&model.Notification{}).
Where("user_id = ? AND is_read = ?", userID, false).
Updates(map[string]interface{}{
"is_read": true,
"read_at": gorm.Expr("NOW()"),
}).Error
}

// GetPreferences kullanÄ±cÄ±nÄ±n bildirim tercihlerini getir
func (r *NotificationRepository) GetPreferences(userID uint) (*model.NotificationPreference, error) {
var prefs model.NotificationPreference
err := r.db.Where("user_id = ?", userID).First(&prefs).Error
if err != nil {
return nil, err
}
return &prefs, nil
}

// CreateOrUpdatePreferences bildirim tercihlerini oluÅŸtur veya gÃ¼ncelle
func (r *NotificationRepository) CreateOrUpdatePreferences(prefs *model.NotificationPreference) error {
return r.db.Save(prefs).Error
}
