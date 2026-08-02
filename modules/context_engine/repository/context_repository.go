package repository

import (
"modules/context_engine/model"

"gorm.io/gorm"
)

type ContextRepository struct {
db *gorm.DB
}

func NewContextRepository(db *gorm.DB) *ContextRepository {
return &ContextRepository{
db: db,
}
}

// Session operations
func (r *ContextRepository) CreateSession(session *model.ContextSession) error {
return r.db.Create(session).Error
}

func (r *ContextRepository) FindSession(userID uint, sessionID string) (*model.ContextSession, error) {
var session model.ContextSession
err := r.db.Where("user_id = ? AND session_id = ?", userID, sessionID).First(&session).Error
if err != nil {
return nil, err
}
return &session, nil
}

func (r *ContextRepository) FindSessions(userID uint) ([]model.ContextSession, error) {
var sessions []model.ContextSession
err := r.db.Where("user_id = ? AND is_active = ?", userID, true).
Order("updated_at DESC").
Find(&sessions).Error
return sessions, err
}

func (r *ContextRepository) UpdateSession(session *model.ContextSession) error {
return r.db.Save(session).Error
}

func (r *ContextRepository) DeleteSession(userID uint, sessionID string) error {
return r.db.Where("user_id = ? AND session_id = ?", userID, sessionID).
Delete(&model.ContextSession{}).Error
}

// Message operations
func (r *ContextRepository) AddMessage(message *model.ContextMessage) error {
return r.db.Create(message).Error
}

func (r *ContextRepository) GetMessages(sessionID uint, limit int) ([]model.ContextMessage, error) {
var messages []model.ContextMessage
err := r.db.Where("session_id = ?", sessionID).
Order("created_at ASC").
Limit(limit).
Find(&messages).Error
return messages, err
}
