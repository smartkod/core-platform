package repository

import (
"modules/analytics/model"

"gorm.io/gorm"
)

type AnalyticsRepository struct {
db *gorm.DB
}

func NewAnalyticsRepository(db *gorm.DB) *AnalyticsRepository {
return &AnalyticsRepository{
db: db,
}
}

func (r *AnalyticsRepository) CreateActivity(activity *model.UserActivity) error {
return r.db.Create(activity).Error
}

func (r *AnalyticsRepository) GetDailyStats(date string) (*model.DailyStats, error) {
var stats model.DailyStats
err := r.db.Where("date = ?", date).First(&stats).Error
if err != nil {
return nil, err
}
return &stats, nil
}
