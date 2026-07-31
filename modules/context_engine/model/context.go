package model

import (
"time"

"gorm.io/gorm"
)

// ContextSession kullanıcı oturumu
type ContextSession struct {
ID          uint           json:"id" gorm:"primaryKey"
UserID      uint           json:"user_id" gorm:"not null;index"
SessionID   string         json:"session_id" gorm:"unique;not null"
Title       string         json:"title"
Context     string         json:"context" gorm:"type:text"
Summary     string         json:"summary" gorm:"type:text"
MessageCount int           json:"message_count" gorm:"default:0"
TokenCount  int            json:"token_count" gorm:"default:0"
IsActive    bool           json:"is_active" gorm:"default:true"
ExpiresAt   *time.Time     json:"expires_at"
CreatedAt   time.Time      json:"created_at"
UpdatedAt   time.Time      json:"updated_at"
DeletedAt   gorm.DeletedAt json:"-" gorm:"index"
}

func (ContextSession) TableName() string {
return "context_sessions"
}

// ContextMessage konuşma mesajı
type ContextMessage struct {
ID          uint           json:"id" gorm:"primaryKey"
SessionID   uint           json:"session_id" gorm:"not null;index"
Role        string         json:"role" gorm:"not null" // user, assistant, system
Content     string         json:"content" gorm:"type:text;not null"
Tokens      int            json:"tokens" gorm:"default:0"
CreatedAt   time.Time      json:"created_at"
DeletedAt   gorm.DeletedAt json:"-" gorm:"index"
}

func (ContextMessage) TableName() string {
return "context_messages"
}

// CreateSessionRequest oturum oluşturma isteği
type CreateSessionRequest struct {
Title string json:"title"
}

// AddMessageRequest mesaj ekleme isteği
type AddMessageRequest struct {
Role    string json:"role" validate:"required" // user, assistant, system
Content string json:"content" validate:"required"
}

// SessionResponse oturum yanıtı
type SessionResponse struct {
ID           uint      json:"id"
SessionID    string    json:"session_id"
Title        string    json:"title"
Context      string    json:"context"
Summary      string    json:"summary"
MessageCount int       json:"message_count"
TokenCount   int       json:"token_count"
IsActive     bool      json:"is_active"
CreatedAt    time.Time json:"created_at"
UpdatedAt    time.Time json:"updated_at"
}

// MessageResponse mesaj yanıtı
type MessageResponse struct {
ID        uint      json:"id"
Role      string    json:"role"
Content   string    json:"content"
Tokens    int       json:"tokens"
CreatedAt time.Time json:"created_at"
}
