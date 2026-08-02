package repository

import (
"modules/ai_copilot/model"

"gorm.io/gorm"
)

type CopilotRepository struct {
db *gorm.DB
}

func NewCopilotRepository(db *gorm.DB) *CopilotRepository {
return &CopilotRepository{
db: db,
}
}

// Session operations
func (r *CopilotRepository) FindByUserID(userID uint) (*model.CopilotSession, error) {
var session model.CopilotSession
err := r.db.Where("user_id = ?", userID).First(&session).Error
if err != nil {
return nil, err
}
return &session, nil
}

func (r *CopilotRepository) CreateSession(session *model.CopilotSession) error {
return r.db.Create(session).Error
}

func (r *CopilotRepository) UpdateSession(session *model.CopilotSession) error {
return r.db.Save(session).Error
}

// Message operations
func (r *CopilotRepository) CreateMessage(message *model.CopilotMessage) error {
return r.db.Create(message).Error
}

func (r *CopilotRepository) GetMessages(sessionID uint, limit int) ([]model.CopilotMessage, error) {
var messages []model.CopilotMessage
err := r.db.Where("session_id = ?", sessionID).
Order("created_at DESC").
Limit(limit).
Find(&messages).Error
return messages, err
}
