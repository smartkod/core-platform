package model

import (
"time"

"gorm.io/gorm"
)

// LifeEvent kullanÄ±cÄ± yaÅŸam olayÄ± (GeliÅŸtirilmiÅŸ)
type LifeEvent struct {
ID          uint            + "json:\"id\" gorm:\"primaryKey\"" + 
UserID      uint            + "json:\"user_id\" gorm:\"not null;index\"" + 
Title       string          + "json:\"title\" gorm:\"not null\"" + 
Description string          + "json:\"description\"" + 
Category    string          + "json:\"category\" gorm:\"not null\"" + 
SubCategory string          + "json:\"sub_category\"" + 
EventDate   time.Time       + "json:\"event_date\" gorm:\"not null\"" + 

// GÃ¼ven ve DoÄŸrulama
Confidence    int        + "json:\"confidence\" gorm:\"default:50\"" +  // 0-100
IsVerified    bool       + "json:\"is_verified\" gorm:\"default:false\"" + 
VerifiedCount int        + "json:\"verified_count\" gorm:\"default:0\"" + 
UserFeedback  string     + "json:\"user_feedback\"" +  // accurate, inaccurate, partial

// Zaman FaktÃ¶rÃ¼
PreEventDays  int        + "json:\"pre_event_days\" gorm:\"default:30\"" + 
PostEventDays int        + "json:\"post_event_days\" gorm:\"default:30\"" + 

// Astrolojik BaÄŸlam
AstroContext    string  + "json:\"astro_context\" gorm:\"type:jsonb\"" + 
ClassicalSources string  + "json:\"classical_sources\" gorm:\"type:jsonb\"" + 
AIAnalysis      string  + "json:\"ai_analysis\" gorm:\"type:jsonb\"" + 
CommunityContext string  + "json:\"community_context\" gorm:\"type:jsonb\"" + 

// Gizlilik
IsPublic    bool            + "json:\"is_public\" gorm:\"default:false\"" + 
CreatedAt   time.Time       + "json:\"created_at\"" + 
UpdatedAt   time.Time       + "json:\"updated_at\"" + 
DeletedAt   gorm.DeletedAt  + "json:\"-\" gorm:\"index\"" + 
}

func (LifeEvent) TableName() string {
return "life_events"
}

// EventCategory kategoriler
var EventCategories = map[string][]string{
"career":     {"new_job", "promotion", "resignation", "business_start", "career_change", "retirement"},
"residence":  {"house_move", "city_change", "country_change", "renovation", "new_house"},
"relationship": {"marriage", "engagement", "divorce", "new_relationship", "separation", "breakup"},
"health":     {"illness", "recovery", "surgery", "pregnancy", "birth", "injury"},
"travel":     {"long_travel", "abroad_travel", "education_abroad", "business_trip"},
"family":     {"marriage", "birth", "death", "family_event", "reunion"},
"education":  {"graduation", "new_course", "certification", "phd", "degree"},
}

// ClassicalSource klasik astroloji kaynaÄŸÄ±
type ClassicalSource struct {
Name      string  + "json:\"name\"" + 
Book      string  + "json:\"book\"" + 
Chapter   string  + "json:\"chapter\"" + 
Quote     string  + "json:\"quote\"" + 
Relevance string  + "json:\"relevance\"" +  // high, medium, low
}

// EventAnalytics anonim analiz sonucu
type EventAnalytics struct {
Category       string             + "json:\"category\"" + 
EventType      string             + "json:\"event_type\"" + 
TotalUsers     int                + "json:\"total_users\"" + 
TotalEvents    int                + "json:\"total_events\"" + 
CommonHouses   map[string]int     + "json:\"common_houses\"" + 
CommonPlanets  map[string]int     + "json:\"common_planets\"" + 
CommonTransits map[string]int     + "json:\"common_transits\"" + 
AverageAge     int                + "json:\"average_age\"" + 
Patterns       []string           + "json:\"patterns\"" + 
}
