package model

import (
"time"

"gorm.io/gorm"
)

// CommunityPost topluluk paylaÅŸÄ±mÄ±
type CommunityPost struct {
ID          uint           json:"id" gorm:"primaryKey"
UserID      uint           json:"user_id" gorm:"not null;index"
Title       string         json:"title" gorm:"not null"
Content     string         json:"content" gorm:"type:text;not null"
Category    string         json:"category" gorm:"not null" // experience, question, insight
Tags        string         json:"tags" gorm:"type:jsonb"   // JSON array
IsAnonymous bool           json:"is_anonymous" gorm:"default:false"
LikeCount   int            json:"like_count" gorm:"default:0"
CommentCount int           json:"comment_count" gorm:"default:0"
ViewCount   int            json:"view_count" gorm:"default:0"
HelpfulCount int           json:"helpful_count" gorm:"default:0"
CreatedAt   time.Time      json:"created_at"
UpdatedAt   time.Time      json:"updated_at"
DeletedAt   gorm.DeletedAt json:"-" gorm:"index"
}

func (CommunityPost) TableName() string {
return "community_posts"
}

// CommunityComment topluluk yorumu
type CommunityComment struct {
ID          uint           json:"id" gorm:"primaryKey"
PostID      uint           json:"post_id" gorm:"not null;index"
UserID      uint           json:"user_id" gorm:"not null;index"
Content     string         json:"content" gorm:"type:text;not null"
IsAnonymous bool           json:"is_anonymous" gorm:"default:false"
HelpfulCount int           json:"helpful_count" gorm:"default:0"
CreatedAt   time.Time      json:"created_at"
UpdatedAt   time.Time      json:"updated_at"
DeletedAt   gorm.DeletedAt json:"-" gorm:"index"
}

func (CommunityComment) TableName() string {
return "community_comments"
}

// EventPattern anonim olay deseni
type EventPattern struct {
ID          uint           json:"id" gorm:"primaryKey"
Category    string         json:"category" gorm:"not null"
SubCategory string         json:"sub_category" gorm:"not null"
TotalEvents int            json:"total_events" gorm:"default:0"
TotalUsers  int            json:"total_users" gorm:"default:0"
CommonHouses string        json:"common_houses" gorm:"type:jsonb"
CommonPlanets string       json:"common_planets" gorm:"type:jsonb"
CommonTransits string      json:"common_transits" gorm:"type:jsonb"
Insight     string         json:"insight" gorm:"type:text"
Confidence  int            json:"confidence" gorm:"default:0"
CreatedAt   time.Time      json:"created_at"
UpdatedAt   time.Time      json:"updated_at"
DeletedAt   gorm.DeletedAt json:"-" gorm:"index"
}

func (EventPattern) TableName() string {
return "event_patterns"
}

// CommunityPostRequest topluluk paylaÅŸÄ±mÄ± isteÄŸi
type CommunityPostRequest struct {
Title       string   json:"title" validate:"required"
Content     string   json:"content" validate:"required"
Category    string   json:"category" validate:"required" // experience, question, insight
Tags        []string json:"tags"
IsAnonymous bool     json:"is_anonymous"
}

// CommunityPostResponse topluluk paylaÅŸÄ±mÄ± yanÄ±tÄ±
type CommunityPostResponse struct {
ID           uint      json:"id"
Title        string    json:"title"
Content      string    json:"content"
Category     string    json:"category"
Tags         []string  json:"tags"
IsAnonymous  bool      json:"is_anonymous"
LikeCount    int       json:"like_count"
CommentCount int       json:"comment_count"
HelpfulCount int       json:"helpful_count"
CreatedAt    time.Time json:"created_at"
UpdatedAt    time.Time json:"updated_at"
}

// PatternDiscoveryResponse desen keÅŸfi yanÄ±tÄ±
type PatternDiscoveryResponse struct {
Category    string                 json:"category"
SubCategory string                 json:"sub_category"
TotalEvents int                    json:"total_events"
TotalUsers  int                    json:"total_users"
CommonHouses []string              json:"common_houses"
CommonPlanets []string             json:"common_planets"
CommonTransits []string            json:"common_transits"
Insight     string                 json:"insight"
Confidence  int                    json:"confidence"
Statistics  map[string]interface{} json:"statistics"
}
