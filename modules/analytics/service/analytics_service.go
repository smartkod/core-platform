package service

import (
"time"

"modules/analytics/model"
"modules/analytics/repository"

"gorm.io/gorm"
)

type AnalyticsService struct {
analyticsRepo *repository.AnalyticsRepository
}

func NewAnalyticsService(db *gorm.DB) *AnalyticsService {
return &AnalyticsService{
analyticsRepo: repository.NewAnalyticsRepository(db),
}
}

// GetDashboard dashboard verilerini getirir
func (s *AnalyticsService) GetDashboard() (*model.AnalyticsDashboard, error) {
// Mock veriler (gerÃ§ek veritabanÄ± sorgularÄ± yerine)
dashboard := &model.AnalyticsDashboard{
Overview: model.OverviewStats{
TotalUsers:      1250,
ActiveToday:     150,
ActiveThisWeek:  850,
TotalRevenue:    32500.50,
PremiumUsers:    45,
TotalPosts:      320,
TotalComments:   1250,
GrowthRate:      12.5,
},
Users: model.UserStats{
DailyActive: []model.DailyActive{
{Date: "2026-07-25", Count: 120},
{Date: "2026-07-26", Count: 135},
{Date: "2026-07-27", Count: 142},
{Date: "2026-07-28", Count: 138},
{Date: "2026-07-29", Count: 145},
{Date: "2026-07-30", Count: 150},
{Date: "2026-07-31", Count: 160},
},
NewUsers: []model.NewUser{
{Date: "2026-07-25", Count: 8},
{Date: "2026-07-26", Count: 12},
{Date: "2026-07-27", Count: 10},
{Date: "2026-07-28", Count: 15},
{Date: "2026-07-29", Count: 11},
{Date: "2026-07-30", Count: 14},
{Date: "2026-07-31", Count: 18},
},
RetentionRate:  68.5,
AvgSessionTime: 12.3,
},
Community: model.CommunityStats{
TotalPosts:      320,
TotalComments:   1250,
TotalLikes:      4500,
TotalFollows:    380,
EngagementRate:  15.2,
TopCategories: []model.CategoryStat{
{Category: "experience", Count: 180},
{Category: "question", Count: 95},
{Category: "insight", Count: 45},
},
},
Revenue: model.RevenueStats{
TotalRevenue: 32500.50,
RevenueBySource: model.RevenueBySource{
Premium: 25000.00,
Ads:     5000.00,
Tokens:  2500.50,
},
MonthlyRevenue: []model.MonthlyRevenue{
{Month: "Ocak", Total: 2000.00},
{Month: "Subat", Total: 2500.00},
{Month: "Mart", Total: 3000.00},
{Month: "Nisan", Total: 3500.00},
{Month: "Mayis", Total: 4000.00},
{Month: "Haziran", Total: 4500.00},
{Month: "Temmuz", Total: 5000.00},
},
},
AIUsage: model.AIUsageStats{
TotalRequests:   8500,
AvgResponseTime: 1.2,
MostUsedModel:   "gpt-3.5-turbo",
TotalTokensUsed: 1250000,
},
PopularContent: []model.PopularContent{
{ID: 1, Title: "Saturn 4. evde taÅŸÄ±nma deneyimim", Type: "post", Views: 450, Likes: 120, Comments: 35},
{ID: 2, Title: "JÃ¼piter 10. evde kariyer deÄŸiÅŸikliÄŸi", Type: "post", Views: 380, Likes: 95, Comments: 28},
{ID: 3, Title: "VenÃ¼s 7. evde iliÅŸki dinamikleri", Type: "post", Views: 320, Likes: 85, Comments: 22},
},
RecentActivities: []model.RecentActivity{
{UserID: 1, UserName: "AstroMaster", Action: "yeni paylaÅŸÄ±m", Target: "Saturn 4. evde...", CreatedAt: "2026-07-31T10:30:00Z"},
{UserID: 2, UserName: "StarGazer", Action: "yorum yaptÄ±", Target: "JÃ¼piter 10. evde...", CreatedAt: "2026-07-31T10:15:00Z"},
{UserID: 3, UserName: "CosmicWanderer", Action: "beÄŸendi", Target: "Saturn 4. evde...", CreatedAt: "2026-07-31T10:00:00Z"},
},
}

return dashboard, nil
}

// TrackActivity kullanÄ±cÄ± aktivitesini kaydeder
func (s *AnalyticsService) TrackActivity(userID uint, activityType, action, metadata string) error {
activity := &model.UserActivity{
UserID:   userID,
Type:     activityType,
Action:   action,
Metadata: metadata,
}
return s.analyticsRepo.CreateActivity(activity)
}
