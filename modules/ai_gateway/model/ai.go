package model

import (
"time"

"gorm.io/gorm"
)

// AIProvider AI saÄŸlayÄ±cÄ±
type AIProvider struct {
ID          uint           json:"id" gorm:"primaryKey"
Name        string         json:"name" gorm:"unique;not null" // openai, anthropic, gemini
APIKey      string         json:"-" gorm:"not null"           // API anahtarÄ±
BaseURL     string         json:"base_url"
Models      string         json:"models" gorm:"type:jsonb"    // JSON array
IsActive    bool           json:"is_active" gorm:"default:true"
Priority    int            json:"priority" gorm:"default:1"
CreatedAt   time.Time      json:"created_at"
UpdatedAt   time.Time      json:"updated_at"
DeletedAt   gorm.DeletedAt json:"-" gorm:"index"
}

func (AIProvider) TableName() string {
return "ai_providers"
}

// AIRequest AI isteÄŸi
type AIRequest struct {
Provider    string                 json:"provider" validate:"required" // openai, anthropic, gemini
Model       string                 json:"model" validate:"required"
Prompt      string                 json:"prompt" validate:"required"
MaxTokens   int                    json:"max_tokens"
Temperature float64                json:"temperature"
Options     map[string]interface{} json:"options"
}

// AIResponse AI yanÄ±tÄ±
type AIResponse struct {
ID           string                 json:"id"
Provider     string                 json:"provider"
Model        string                 json:"model"
Content      string                 json:"content"
Usage        map[string]interface{} json:"usage"
FinishReason string                 json:"finish_reason"
CreatedAt    time.Time              json:"created_at"
}

// AICache AI Ã¶nbellek
type AICache struct {
ID          uint           json:"id" gorm:"primaryKey"
Provider    string         json:"provider" gorm:"not null"
Model       string         json:"model" gorm:"not null"
PromptHash  string         json:"prompt_hash" gorm:"index;not null"
Response    string         json:"response" gorm:"type:text"
TokensSaved int            json:"tokens_saved"
CreatedAt   time.Time      json:"created_at"
ExpiresAt   time.Time      json:"expires_at"
DeletedAt   gorm.DeletedAt json:"-" gorm:"index"
}

func (AICache) TableName() string {
return "ai_cache"
}

// AIUsage AI kullanÄ±m istatistikleri
type AIUsage struct {
ID          uint           json:"id" gorm:"primaryKey"
Provider    string         json:"provider" gorm:"not null"
Model       string         json:"model" gorm:"not null"
UserID      uint           json:"user_id" gorm:"not null"
TokensIn    int            json:"tokens_in"
TokensOut   int            json:"tokens_out"
TotalTokens int            json:"total_tokens"
Cost        float64        json:"cost"
CreatedAt   time.Time      json:"created_at"
DeletedAt   gorm.DeletedAt json:"-" gorm:"index"
}

func (AIUsage) TableName() string {
return "ai_usage"
}
