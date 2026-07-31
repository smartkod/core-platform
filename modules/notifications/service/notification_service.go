package service

import (
"errors"
"time"

"modules/notifications/model"
"modules/notifications/repository"

"gorm.io/gorm"
)

type NotificationService struct {
notifRepo *repository.NotificationRepository
}

func NewNotificationService(db *gorm.DB) *NotificationService {
return &NotificationService{
notifRepo: repository.NewNotificationRepository(db),
}
}

// SendNotification bildirim gönder
func (s *NotificationService) SendNotification(req *model.CreateNotificationRequest) (*model.NotificationResponse, error) {
// Bildirim oluştur
notification := &model.Notification{
UserID:  req.UserID,
Type:    req.Type,
Title:   req.Title,
Content: req.Content,
Data:    req.Data,
IsRead:  false,
IsSent:  false,
}

if err := s.notifRepo.CreateNotification(notification); err != nil {
return nil, err
}

// Bildirimi gönder (mock)
// TODO: Gerçek bildirim gönderimi (Firebase, SMTP, SMS API)
now := time.Now()
notification.IsSent = true
notification.SentAt = &now

return s.toResponse(notification), nil
}

// GetNotifications kullanıcının bildirimlerini getir
func (s *NotificationService) GetNotifications(userID uint, limit int) ([]model.NotificationResponse, error) {
notifications, err := s.notifRepo.GetUserNotifications(userID, limit)
if err != nil {
return nil, err
}

responses := make([]model.NotificationResponse, len(notifications))
for i, notif := range notifications {
responses[i] = *s.toResponse(&notif)
}
return responses, nil
}

// GetUnreadCount okunmamış bildirim sayısını getir
func (s *NotificationService) GetUnreadCount(userID uint) (int64, error) {
return s.notifRepo.GetUnreadCount(userID)
}

// MarkAsRead bildirimi okundu olarak işaretle
func (s *NotificationService) MarkAsRead(notificationID uint) error {
return s.notifRepo.MarkAsRead(notificationID)
}

// MarkAllAsRead tüm bildirimleri okundu olarak işaretle
func (s *NotificationService) MarkAllAsRead(userID uint) error {
return s.notifRepo.MarkAllAsRead(userID)
}

// GetPreferences kullanıcının bildirim tercihlerini getir
func (s *NotificationService) GetPreferences(userID uint) (*model.NotificationPreferenceResponse, error) {
prefs, err := s.notifRepo.GetPreferences(userID)
if err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
// Varsayılan tercihleri oluştur
defaultPrefs := &model.NotificationPreference{
UserID:      userID,
Email:       true,
Push:        true,
InApp:       true,
SMS:         false,
Marketing:   true,
Transaction: true,
System:      true,
}
if err := s.notifRepo.CreateOrUpdatePreferences(defaultPrefs); err != nil {
return nil, err
}
return s.toPreferenceResponse(defaultPrefs), nil
}
return nil, err
}
return s.toPreferenceResponse(prefs), nil
}

// UpdatePreferences bildirim tercihlerini güncelle
func (s *NotificationService) UpdatePreferences(userID uint, req *model.UpdatePreferenceRequest) (*model.NotificationPreferenceResponse, error) {
prefs, err := s.notifRepo.GetPreferences(userID)
if err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
prefs = &model.NotificationPreference{
UserID: userID,
}
} else {
return nil, err
}
}

if req.Email != nil {
prefs.Email = *req.Email
}
if req.Push != nil {
prefs.Push = *req.Push
}
if req.InApp != nil {
prefs.InApp = *req.InApp
}
if req.SMS != nil {
prefs.SMS = *req.SMS
}
if req.Marketing != nil {
prefs.Marketing = *req.Marketing
}
if req.Transaction != nil {
prefs.Transaction = *req.Transaction
}
if req.System != nil {
prefs.System = *req.System
}

if err := s.notifRepo.CreateOrUpdatePreferences(prefs); err != nil {
return nil, err
}

return s.toPreferenceResponse(prefs), nil
}

// toResponse model'den response oluştur
func (s *NotificationService) toResponse(notification *model.Notification) *model.NotificationResponse {
return &model.NotificationResponse{
ID:        notification.ID,
Type:      notification.Type,
Title:     notification.Title,
Content:   notification.Content,
Data:      notification.Data,
IsRead:    notification.IsRead,
IsSent:    notification.IsSent,
CreatedAt: notification.CreatedAt,
ReadAt:    notification.ReadAt,
}
}

// toPreferenceResponse model'den response oluştur
func (s *NotificationService) toPreferenceResponse(prefs *model.NotificationPreference) *model.NotificationPreferenceResponse {
return &model.NotificationPreferenceResponse{
Email:       prefs.Email,
Push:        prefs.Push,
InApp:       prefs.InApp,
SMS:         prefs.SMS,
Marketing:   prefs.Marketing,
Transaction: prefs.Transaction,
System:      prefs.System,
}
}
