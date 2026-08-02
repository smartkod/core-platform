package repository

import (
"modules/rewarded_ads/model"

"gorm.io/gorm"
)

type AdRepository struct {
db *gorm.DB
}

func NewAdRepository(db *gorm.DB) *AdRepository {
return &AdRepository{
db: db,
}
}

func (r *AdRepository) GetDailyLimit(userID uint, date string) (*model.DailyAdLimit, error) {
var limit model.DailyAdLimit
err := r.db.Where("user_id = ? AND date = ?", userID, date).First(&limit).Error
if err != nil {
return nil, err
}
return &limit, nil
}

func (r *AdRepository) UpdateDailyLimit(tx *gorm.DB, limit *model.DailyAdLimit) error {
return tx.Save(limit).Error
}

func (r *AdRepository) CreateReward(tx *gorm.DB, reward *model.AdReward) error {
return tx.Create(reward).Error
}

func (r *AdRepository) AdExists(userID uint, adID, provider string) (bool, error) {
var count int64
err := r.db.Model(&model.AdReward{}).
Where("user_id = ? AND ad_id = ? AND ad_provider = ?", userID, adID, provider).
Count(&count).Error
return count > 0, err
}
