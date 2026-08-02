package repository

import (
"modules/premium/model"

"gorm.io/gorm"
)

type PremiumRepository struct {
db *gorm.DB
}

func NewPremiumRepository(db *gorm.DB) *PremiumRepository {
return &PremiumRepository{
db: db,
}
}

// GetPlanByID plan ID'ye gÃ¶re getir
func (r *PremiumRepository) GetPlanByID(id uint) (*model.PremiumPlan, error) {
var plan model.PremiumPlan
err := r.db.First(&plan, id).Error
if err != nil {
return nil, err
}
return &plan, nil
}

// GetActivePlans aktif planlarÄ± getir
func (r *PremiumRepository) GetActivePlans() ([]model.PremiumPlan, error) {
var plans []model.PremiumPlan
err := r.db.Where("is_active = ?", true).Find(&plans).Error
return plans, err
}

// GetUserPremium kullanÄ±cÄ±nÄ±n premium kaydÄ±nÄ± getir
func (r *PremiumRepository) GetUserPremium(userID uint) (*model.UserPremium, error) {
var premium model.UserPremium
err := r.db.Where("user_id = ? AND is_active = ?", userID, true).First(&premium).Error
if err != nil {
return nil, err
}
return &premium, nil
}

// CreateUserPremium kullanÄ±cÄ± premium kaydÄ± oluÅŸtur
func (r *PremiumRepository) CreateUserPremium(premium *model.UserPremium) error {
return r.db.Create(premium).Error
}

// UpdateUserPremium kullanÄ±cÄ± premium kaydÄ±nÄ± gÃ¼ncelle
func (r *PremiumRepository) UpdateUserPremium(premium *model.UserPremium) error {
return r.db.Save(premium).Error
}

// CancelUserPremium kullanÄ±cÄ± premium kaydÄ±nÄ± iptal et
func (r *PremiumRepository) CancelUserPremium(userID uint) error {
return r.db.Model(&model.UserPremium{}).
Where("user_id = ?", userID).
Updates(map[string]interface{}{
"is_active":   false,
"canceled_at": gorm.Expr("NOW()"),
}).Error
}
