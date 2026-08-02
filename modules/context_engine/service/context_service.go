package service

import (
"crypto/rand"
"encoding/hex"
"errors"
"fmt"
"strings"
"time"

"modules/context_engine/model"
"modules/context_engine/repository"

"gorm.io/gorm"
)

type ContextService struct {
contextRepo *repository.ContextRepository
maxTokens   int
}

func NewContextService(db *gorm.DB) *ContextService {
return &ContextService{
contextRepo: repository.NewContextRepository(db),
maxTokens:   4000, // Maksimum context token sayÄ±sÄ±
}
}

// CreateSession yeni oturum oluÅŸturur
func (s *ContextService) CreateSession(userID uint, req *model.CreateSessionRequest) (*model.SessionResponse, error) {
sessionID := s.generateSessionID()

session := &model.ContextSession{
UserID:    userID,
SessionID: sessionID,
Title:     req.Title,
IsActive:  true,
}

if err := s.contextRepo.CreateSession(session); err != nil {
return nil, err
}

return s.toSessionResponse(session), nil
}

// GetSession oturum bilgilerini getirir
func (s *ContextService) GetSession(userID uint, sessionID string) (*model.SessionResponse, error) {
session, err := s.contextRepo.FindSession(userID, sessionID)
if err != nil {
return nil, errors.New("session not found")
}
return s.toSessionResponse(session), nil
}

// GetSessions kullanÄ±cÄ±nÄ±n tÃ¼m oturumlarÄ±nÄ± getirir
func (s *ContextService) GetSessions(userID uint) ([]model.SessionResponse, error) {
sessions, err := s.contextRepo.FindSessions(userID)
if err != nil {
return nil, err
}

responses := make([]model.SessionResponse, len(sessions))
for i, session := range sessions {
responses[i] = *s.toSessionResponse(&session)
}
return responses, nil
}

// AddMessage oturuma mesaj ekler
func (s *ContextService) AddMessage(userID uint, sessionID string, req *model.AddMessageRequest) (*model.MessageResponse, error) {
session, err := s.contextRepo.FindSession(userID, sessionID)
if err != nil {
return nil, errors.New("session not found")
}

if !session.IsActive {
return nil, errors.New("session is not active")
}

// Token sayÄ±sÄ±nÄ± hesapla (basit)
tokens := len(strings.Split(req.Content, " "))

message := &model.ContextMessage{
SessionID: session.ID,
Role:      req.Role,
Content:   req.Content,
Tokens:    tokens,
}

if err := s.contextRepo.AddMessage(message); err != nil {
return nil, err
}

// Oturum bilgilerini gÃ¼ncelle
session.MessageCount++
session.TokenCount += tokens

// Context'i gÃ¼ncelle
s.updateContext(session)

if err := s.contextRepo.UpdateSession(session); err != nil {
return nil, err
}

return s.toMessageResponse(message), nil
}

// GetMessages oturumun mesajlarÄ±nÄ± getirir
func (s *ContextService) GetMessages(userID uint, sessionID string, limit int) ([]model.MessageResponse, error) {
session, err := s.contextRepo.FindSession(userID, sessionID)
if err != nil {
return nil, errors.New("session not found")
}

messages, err := s.contextRepo.GetMessages(session.ID, limit)
if err != nil {
return nil, err
}

responses := make([]model.MessageResponse, len(messages))
for i, msg := range messages {
responses[i] = *s.toMessageResponse(&msg)
}
return responses, nil
}

// DeleteSession oturumu siler
func (s *ContextService) DeleteSession(userID uint, sessionID string) error {
return s.contextRepo.DeleteSession(userID, sessionID)
}

// ClearContext oturumun context'ini temizler
func (s *ContextService) ClearContext(userID uint, sessionID string) error {
session, err := s.contextRepo.FindSession(userID, sessionID)
if err != nil {
return errors.New("session not found")
}

session.Context = ""
session.Summary = ""
session.MessageCount = 0
session.TokenCount = 0

return s.contextRepo.UpdateSession(session)
}

// updateContext oturumun context'ini gÃ¼nceller
func (s *ContextService) updateContext(session *model.ContextSession) {
// Basit context yÃ¶netimi
messages, _ := s.contextRepo.GetMessages(session.ID, 10)

var contextBuilder strings.Builder
for _, msg := range messages {
contextBuilder.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
}

session.Context = contextBuilder.String()

// Token limitini kontrol et
if session.TokenCount > s.maxTokens {
// Token limiti aÅŸÄ±ldÄ±, eski mesajlarÄ± Ã¶zetle
session.Summary = "Context Ã¶zetlenmiÅŸ: " + session.Context[:200]
session.Context = session.Summary
}
}

// generateSessionID benzersiz session ID oluÅŸturur
func (s *ContextService) generateSessionID() string {
bytes := make([]byte, 16)
rand.Read(bytes)
return hex.EncodeToString(bytes)
}

// toSessionResponse model'den response oluÅŸturur
func (s *ContextService) toSessionResponse(session *model.ContextSession) *model.SessionResponse {
return &model.SessionResponse{
ID:           session.ID,
SessionID:    session.SessionID,
Title:        session.Title,
Context:      session.Context,
Summary:      session.Summary,
MessageCount: session.MessageCount,
TokenCount:   session.TokenCount,
IsActive:     session.IsActive,
CreatedAt:    session.CreatedAt,
UpdatedAt:    session.UpdatedAt,
}
}

// toMessageResponse model'den response oluÅŸturur
func (s *ContextService) toMessageResponse(msg *model.ContextMessage) *model.MessageResponse {
return &model.MessageResponse{
ID:        msg.ID,
Role:      msg.Role,
Content:   msg.Content,
Tokens:    msg.Tokens,
CreatedAt: msg.CreatedAt,
}
}
