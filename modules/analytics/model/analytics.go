package model

import (
"time"

"gorm.io/gorm"
)

// UserActivity kullanÄ±cÄ± aktivite kaydÄ±
type UserActivity struct {
ID          uint           json:"id" gorm:"primaryKey"
UserID      uint           json:"user_id" gorm:"not null;index"
Type        string         json:"type" gorm:"not null" // login, view, post, comment, like, follow, purchase
Action      string         json:"action"
Metadata    string         json:"metadata" gorm:"type:jsonb"
IPAddress   string         json:"ip_address"
UserAgent   string         json:"user_agent"
CreatedAt   time.Time      json:"created_at"
DeletedAt   gorm.DeletedAt json:"-" gorm:"index"
}

func (UserActivity) TableName() string {
return "user_activities"
}

// DailyStats gÃ¼nlÃ¼k istatistikler
type DailyStats struct {
ID              uint           json:"id" gorm:"primaryKey"
Date            string         json:"date" gorm:"uniqueIndex;not null" // YYYY-MM-DD
TotalUsers      int            json:"total_users"
ActiveUsers     int            json:"active_users"
NewUsers        int            json:"new_users"
TotalPosts      int            json:"total_posts"
TotalComments   int            json:"total_comments"
TotalLikes      int            json:"total_likes"
TotalFollows    int            json:"total_follows"
TotalPremium    int            json:"total_premium"
TotalRevenue    float64        json:"total_revenue"
TotalAdsWatched int            json:"total_ads_watched"
TotalTokensEarned int          json:"total_tokens_earned"
TotalTokensSpent int          json:"total_tokens_spent"
CreatedAt       time.Time      json:"created_at"
UpdatedAt       time.Time      json:"updated_at"
DeletedAt       gorm.DeletedAt json:"-" gorm:"index"
}

func (DailyStats) TableName() string {
return "daily_stats"
}

// AnalyticsDashboard analitik dashboard yanÄ±tÄ±
type AnalyticsDashboard struct {
Overview        OverviewStats       json:"overview"
Users           UserStats           json:"users"
Community       CommunityStats      json:"community"
Revenue         RevenueStats        json:"revenue"
AIUsage         AIUsageStats        json:"ai_usage"
PopularContent  []PopularContent    json:"popular_content"
RecentActivities []RecentActivity   json:"recent_activities"
}

// OverviewStats genel istatistikler
type OverviewStats struct {
TotalUsers      int     json:"total_users"
ActiveToday     int     json:"active_today"
ActiveThisWeek  int     json:"active_this_week"
TotalRevenue    float64 json:"total_revenue"
PremiumUsers    int     json:"premium_users"
TotalPosts      int     json:"total_posts"
TotalComments   int     json:"total_comments"
GrowthRate      float64 json:"growth_rate"
}

// UserStats kullanÄ±cÄ± istatistikleri
type UserStats struct {
DailyActive    []DailyActive json:"daily_active"
NewUsers       []NewUser     json:"new_users"
RetentionRate  float64       json:"retention_rate"
AvgSessionTime float64       json:"avg_session_time"
}

// CommunityStats topluluk istatistikleri
type CommunityStats struct {
TotalPosts      int     json:"total_posts"
TotalComments   int     json:"total_comments"
TotalLikes      int     json:"total_likes"
TotalFollows    int     json:"total_follows"
EngagementRate  float64 json:"engagement_rate"
TopCategories   []CategoryStat json:"top_categories"
}

// RevenueStats gelir istatistikleri
type RevenueStats struct {
TotalRevenue    float64         json:"total_revenue"
RevenueBySource RevenueBySource json:"revenue_by_source"
MonthlyRevenue  []MonthlyRevenue json:"monthly_revenue"
}

// RevenueBySource gelir kaynaÄŸÄ± daÄŸÄ±lÄ±mÄ±
type RevenueBySource struct {
Premium float64 json:"premium"
Ads     float64 json:"ads"
Tokens  float64 json:"tokens"
}

// AIUsageStats AI kullanÄ±m istatistikleri
type AIUsageStats struct {
TotalRequests   int     json:"total_requests"
AvgResponseTime float64 json:"avg_response_time"
MostUsedModel   string  json:"most_used_model"
TotalTokensUsed int     json:"total_tokens_used"
}

// PopularContent popÃ¼ler iÃ§erik
type PopularContent struct {
ID        uint   json:"id"
Title     string json:"title"
Type      string json:"type"
Views     int    json:"views"
Likes     int    json:"likes"
Comments  int    json:"comments"
}

// RecentActivity son aktiviteler
type RecentActivity struct {
UserID    uint   json:"user_id"
UserName  string json:"user_name"
Action    string json:"action"
Target    string json:"target"
CreatedAt string json:"created_at"
}

// DailyActive gÃ¼nlÃ¼k aktif kullanÄ±cÄ±
type DailyActive struct {
Date  string json:"date"
Count int    json:"count"
}

// NewUser yeni kullanÄ±cÄ±
type NewUser struct {
Date  string json:"date"
Count int    json:"count"
}

// CategoryStat kategori istatistiÄŸi
type CategoryStat struct {
Category string json:"category"
Count    int    json:"count"
}

// MonthlyRevenue aylÄ±k gelir
type MonthlyRevenue struct {
Month string  json:"month"
Total float64 json:"total"
}
