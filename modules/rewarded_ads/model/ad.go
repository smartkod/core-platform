package model

import (
"time"

"gorm.io/gorm"
)

// AdReward reklam ödül kaydı
type AdReward struct {
ID          uint           json:"id" gorm:"primaryKey"
UserID      uint           json:"user_id" gorm:"not null;index"
AdProvider  string         json:"ad_provider" gorm:"not null"
AdID        string         json:"ad_id" gorm:"not null;index"
RewardType  string         json:"reward_type" gorm:"not null" // token, premium_day
RewardValue int            json:"reward_value" gorm:"not null"
IsVerified  bool           json:"is_verified" gorm:"default:false"
IPAddress   string         json:"ip_address"
CreatedAt   time.Time      json:"created_at"
DeletedAt   gorm.DeletedAt json:"-" gorm:"index"
}

func (AdReward) TableName() string {
return "ad_rewards"
}

// DailyAdLimit günlük reklam limiti
type DailyAdLimit struct {
ID           uint           json:"id" gorm:"primaryKey"
UserID       uint           json:"user_id" gorm:"uniqueIndex;not null"
Date         string         json:"date" gorm:"not null;index"
TotalWatched int            json:"total_watched" gorm:"default:0"
TotalReward  int            json:"total_reward" gorm:"default:0"
CreatedAt    time.Time      json:"created_at"
UpdatedAt    time.Time      json:"updated_at"
DeletedAt    gorm.DeletedAt json:"-" gorm:"index"
}

func (DailyAdLimit) TableName() string {
return "daily_ad_limits"
}

// AdWatchRequest reklam izleme isteği
type AdWatchRequest struct {
AdProvider string json:"ad_provider" validate:"required"
AdID       string json:"ad_id" validate:"required"
RewardType string json:"reward_type" validate:"required" // token, premium_day
}

// AdRewardResponse reklam ödül yanıtı
type AdRewardResponse struct {
Success     bool   json:"success"
Message     string json:"message"
RewardValue int    json:"reward_value"
TotalToday  int    json:"total_today"
Remaining   int    json:"remaining"
}

// AdStatusResponse reklam durumu yanıtı
type AdStatusResponse struct {
CanWatch        bool json:"can_watch"
WatchedToday    int  json:"watched_today"
RemainingToday  int  json:"remaining_today"
TokenPerAd      int  json:"token_per_ad"
DailyLimit      int  json:"daily_limit"
DailyTokenLimit int  json:"daily_token_limit"
}
