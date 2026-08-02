package model

import (
"time"

"gorm.io/gorm"
)

// PromptTemplate prompt ÅŸablonu
type PromptTemplate struct {
ID          uint           json:"id" gorm:"primaryKey"
Name        string         json:"name" gorm:"unique;not null"
Category    string         json:"category" gorm:"not null" // astrology, education, fitness, finance
Description string         json:"description"
Template    string         json:"template" gorm:"type:text;not null"
Variables   string         json:"variables" gorm:"type:jsonb" // JSON array
Version     string         json:"version" gorm:"default:1.0.0"
IsActive    bool           json:"is_active" gorm:"default:true"
UsageCount  int            json:"usage_count" gorm:"default:0"
CreatedBy   uint           json:"created_by"
CreatedAt   time.Time      json:"created_at"
UpdatedAt   time.Time      json:"updated_at"
DeletedAt   gorm.DeletedAt json:"-" gorm:"index"
}

func (PromptTemplate) TableName() string {
return "prompt_templates"
}

// PromptExecution prompt Ã§alÄ±ÅŸtÄ±rma kaydÄ±
type PromptExecution struct {
ID           uint           json:"id" gorm:"primaryKey"
TemplateID   uint           json:"template_id" gorm:"not null"
UserID       uint           json:"user_id" gorm:"not null"
Input        string         json:"input" gorm:"type:jsonb"
Output       string         json:"output" gorm:"type:text"
TokensIn     int            json:"tokens_in"
TokensOut    int            json:"tokens_out"
Duration     int64          json:"duration" // ms
Success      bool           json:"success"
ErrorMessage string         json:"error_message"
CreatedAt    time.Time      json:"created_at"
DeletedAt    gorm.DeletedAt json:"-" gorm:"index"
}

func (PromptExecution) TableName() string {
return "prompt_executions"
}

// CreatePromptRequest prompt oluÅŸturma isteÄŸi
type CreatePromptRequest struct {
Name        string            json:"name" validate:"required"
Category    string            json:"category" validate:"required"
Description string            json:"description"
Template    string            json:"template" validate:"required"
Variables   []string          json:"variables"
Version     string            json:"version"
}

// ExecutePromptRequest prompt Ã§alÄ±ÅŸtÄ±rma isteÄŸi
type ExecutePromptRequest struct {
TemplateName string                 json:"template_name" validate:"required"
Variables    map[string]interface{} json:"variables"
}

// PromptResponse prompt yanÄ±tÄ±
type PromptResponse struct {
ID          uint     json:"id"
Name        string   json:"name"
Category    string   json:"category"
Description string   json:"description"
Template    string   json:"template"
Variables   []string json:"variables"
Version     string   json:"version"
IsActive    bool     json:"is_active"
UsageCount  int      json:"usage_count"
CreatedAt   time.Time json:"created_at"
}

// ExecutePromptResponse prompt Ã§alÄ±ÅŸtÄ±rma yanÄ±tÄ±
type ExecutePromptResponse struct {
Result      string json:"result"
TokensIn    int    json:"tokens_in"
TokensOut   int    json:"tokens_out"
Duration    int64  json:"duration" // ms
TemplateID  uint   json:"template_id"
}
