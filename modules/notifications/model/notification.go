package model

import (
"time"

"gorm.io/gorm"
)

// Notification bildirim modeli
type Notification struct {
ID          uint           json:"id" gorm:"primaryKey"
UserID      uint           json:"user_id" gorm:"not null;index"
Type        string         json:"type" gorm:"not null" // email, push, inapp, sms
Title       string         json:"title" gorm:"not null"
Content     string         json:"content" gorm:"not null"
Data        string         json:"data" gorm:"type:jsonb" // Ek veriler
IsRead      bool           json:"is_read" gorm:"default:false"
IsSent      bool           json:"is_sent" gorm:"default:false"
SentAt      *time.Time     json:"sent_at"
ReadAt      *time.Time     json:"read_at"
CreatedAt   time.Time      json:"created_at"
UpdatedAt   time.Time      json:"updated_at"
DeletedAt   gorm.DeletedAt json:"-" gorm:"index"
}

func (Notification) TableName() string {
return "notifications"
}

// NotificationPreference bildirim tercihleri
type NotificationPreference struct {
ID          uint           json:"id" gorm:"primaryKey"
UserID      uint           json:"user_id" gorm:"uniqueIndex;not null"
Email       bool           json:"email" gorm:"default:true"
Push        bool           json:"push" gorm:"default:true"
InApp       bool           json:"in_app" gorm:"default:true"
SMS         bool           json:"sms" gorm:"default:false"
Marketing   bool           json:"marketing" gorm:"default:true"
Transaction bool           json:"transaction" gorm:"default:true"
System      bool           json:"system" gorm:"default:true"
CreatedAt   time.Time      json:"created_at"
UpdatedAt   time.Time      json:"updated_at"
DeletedAt   gorm.DeletedAt json:"-" gorm:"index"
}

func (NotificationPreference) TableName() string {
return "notification_preferences"
}

// CreateNotificationRequest bildirim oluÅŸturma isteÄŸi
type CreateNotificationRequest struct {
UserID  uint   json:"user_id" validate:"required"
Type    string json:"type" validate:"required" // email, push, inapp, sms
Title   string json:"title" validate:"required"
Content string json:"content" validate:"required"
Data    string json:"data"
}

// NotificationResponse bildirim yanÄ±tÄ±
type NotificationResponse struct {
ID        uint       json:"id"
Type      string     json:"type"
Title     string     json:"title"
Content   string     json:"content"
Data      string     json:"data"
IsRead    bool       json:"is_read"
IsSent    bool       json:"is_sent"
CreatedAt time.Time  json:"created_at"
ReadAt    *time.Time json:"read_at,omitempty"
}

// NotificationPreferenceResponse bildirim tercihi yanÄ±tÄ±
type NotificationPreferenceResponse struct {
Email       bool json:"email"
Push        bool json:"push"
InApp       bool json:"in_app"
SMS         bool json:"sms"
Marketing   bool json:"marketing"
Transaction bool json:"transaction"
System      bool json:"system"
}

// UpdatePreferenceRequest bildirim tercihi gÃ¼ncelleme isteÄŸi
type UpdatePreferenceRequest struct {
Email       *bool json:"email"
Push        *bool json:"push"
InApp       *bool json:"in_app"
SMS         *bool json:"sms"
Marketing   *bool json:"marketing"
Transaction *bool json:"transaction"
System      *bool json:"system"
}
