package service

import (
"errors"
"time"

"modules/premium/model"
"modules/premium/repository"

"gorm.io/gorm"
)

type PremiumService struct {
premiumRepo *repository.PremiumRepository
}

func NewPremiumService(db *gorm.DB) *PremiumService {
return &PremiumService{
premiumRepo: repository.NewPremiumRepository(db),
}
}

// GetPlans aktif premium planlarÄ±nÄ± getir
func (s *PremiumService) GetPlans() ([]model.PremiumPlanResponse, error) {
plans, err := s.premiumRepo.GetActivePlans()
if err != nil {
return nil, err
}

responses := make([]model.PremiumPlanResponse, len(plans))
for i, plan := range plans {
responses[i] = s.toPlanResponse(&plan)
}
return responses, nil
}

// GetUserPremium kullanÄ±cÄ±nÄ±n premium durumunu getir
func (s *PremiumService) GetUserPremium(userID uint) (*model.PremiumResponse, error) {
premium, err := s.premiumRepo.GetUserPremium(userID)
if err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
return &model.PremiumResponse{
IsActive: false,
}, nil
}
return nil, err
}

// Premium sÃ¼resi dolmuÅŸ mu kontrol et
if premium.EndDate != nil && premium.EndDate.Before(time.Now()) {
premium.IsActive = false
s.premiumRepo.UpdateUserPremium(premium)
return &model.PremiumResponse{
IsActive: false,
}, nil
}

plan, err := s.premiumRepo.GetPlanByID(premium.PlanID)
if err != nil {
return nil, err
}

daysRemaining := 0
if premium.EndDate != nil {
daysRemaining = int(premium.EndDate.Sub(time.Now()).Hours() / 24)
if daysRemaining < 0 {
daysRemaining = 0
}
}

return &model.PremiumResponse{
IsActive:      premium.IsActive,
PlanName:      plan.Name,
PlanID:        premium.PlanID,
StartDate:     &premium.StartDate,
EndDate:       premium.EndDate,
DaysRemaining: daysRemaining,
IsLifetime:    premium.EndDate == nil,
AutoRenew:     premium.AutoRenew,
}, nil
}

// PurchasePremium premium satÄ±n al
func (s *PremiumService) PurchasePremium(userID uint, req *model.PurchaseRequest) (*model.PremiumResponse, error) {
// PlanÄ± kontrol et
plan, err := s.premiumRepo.GetPlanByID(req.PlanID)
if err != nil {
return nil, errors.New("plan not found")
}

if !plan.IsActive {
return nil, errors.New("plan is not active")
}

// Mevcut premium kaydÄ±nÄ± kontrol et
existing, _ := s.premiumRepo.GetUserPremium(userID)

startDate := time.Now()
var endDate *time.Time

// Lifetime plan (duration = 0)
if plan.Duration == 0 {
endDate = nil
} else {
end := startDate.AddDate(0, 0, plan.Duration)
endDate = &end
}

// Ã–deme iÅŸlemi yapÄ±lacak (mock)
// TODO: Payment entegrasyonu

// Yeni premium kaydÄ± oluÅŸtur veya gÃ¼ncelle
if existing != nil {
// Mevcut premiumu gÃ¼ncelle
existing.PlanID = plan.ID
existing.StartDate = startDate
existing.EndDate = endDate
existing.IsActive = true
existing.AutoRenew = req.AutoRenew
existing.CanceledAt = nil

if err := s.premiumRepo.UpdateUserPremium(existing); err != nil {
return nil, err
}
} else {
// Yeni premium kaydÄ± oluÅŸtur
premium := &model.UserPremium{
UserID:    userID,
PlanID:    plan.ID,
StartDate: startDate,
EndDate:   endDate,
IsActive:  true,
AutoRenew: req.AutoRenew,
}

if err := s.premiumRepo.CreateUserPremium(premium); err != nil {
return nil, err
}
}

return s.GetUserPremium(userID)
}

// CancelPremium premium iptal et
func (s *PremiumService) CancelPremium(userID uint) error {
premium, err := s.premiumRepo.GetUserPremium(userID)
if err != nil {
return errors.New("no active premium subscription found")
}

return s.premiumRepo.CancelUserPremium(userID)
}

// CheckPremium kullanÄ±cÄ±nÄ±n premium aktif mi kontrol et
func (s *PremiumService) CheckPremium(userID uint) bool {
response, err := s.GetUserPremium(userID)
if err != nil {
return false
}
return response.IsActive
}

// toPlanResponse model'den response oluÅŸtur
func (s *PremiumService) toPlanResponse(plan *model.PremiumPlan) model.PremiumPlanResponse {
var features []string
// JSON features'Ä± parse et
// TODO: JSON parse

return model.PremiumPlanResponse{
ID:          plan.ID,
Name:        plan.Name,
Description: plan.Description,
Price:       plan.Price,
Currency:    plan.Currency,
Duration:    plan.Duration,
Features:    features,
}
}
