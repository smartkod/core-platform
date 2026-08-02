package model

import (
"time"

"gorm.io/gorm"
)

// CopilotSession AI Copilot oturumu
type CopilotSession struct {
ID          uint           json:"id" gorm:"primaryKey"
UserID      uint           json:"user_id" gorm:"uniqueIndex;not null"
SessionID   string         json:"session_id" gorm:"unique;not null"
Context     string         json:"context" gorm:"type:jsonb" // KullanÄ±cÄ± baÄŸlamÄ±
Preferences string         json:"preferences" gorm:"type:jsonb"
CreatedAt   time.Time      json:"created_at"
UpdatedAt   time.Time      json:"updated_at"
DeletedAt   gorm.DeletedAt json:"-" gorm:"index"
}

func (CopilotSession) TableName() string {
return "copilot_sessions"
}

// CopilotMessage AI Copilot mesajÄ±
type CopilotMessage struct {
ID          uint           json:"id" gorm:"primaryKey"
SessionID   uint           json:"session_id" gorm:"not null;index"
Role        string         json:"role" gorm:"not null" // user, assistant
Content     string         json:"content" gorm:"type:text;not null"
Context     string         json:"context" gorm:"type:jsonb" // Astrolojik baÄŸlam
Tokens      int            json:"tokens" gorm:"default:0"
CreatedAt   time.Time      json:"created_at"
DeletedAt   gorm.DeletedAt json:"-" gorm:"index"
}

func (CopilotMessage) TableName() string {
return "copilot_messages"
}

// CopilotRequest AI Copilot isteÄŸi
type CopilotRequest struct {
Query string json:"query" validate:"required"
}

// CopilotResponse AI Copilot yanÄ±tÄ±
type CopilotResponse struct {
ID          string                 json:"id"
Answer      string                 json:"answer"
Context     map[string]interface{} json:"context"
Sources     []string               json:"sources"
Confidence  int                    json:"confidence"
CreatedAt   time.Time              json:"created_at"
}

// CopilotContext AI Copilot baÄŸlamÄ±
type CopilotContext struct {
UserProfile    map[string]interface{} json:"user_profile"
BirthChart     map[string]interface{} json:"birth_chart"
CurrentTransits map[string]interface{} json:"current_transits"
RecentEvents   []string               json:"recent_events"
LearningLevel  string                 json:"learning_level"
Preferences    map[string]interface{} json:"preferences"
}
