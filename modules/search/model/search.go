package model

import (
"time"

"gorm.io/gorm"
)

// SearchHistory arama geçmişi
type SearchHistory struct {
ID          uint           json:"id" gorm:"primaryKey"
UserID      uint           json:"user_id" gorm:"not null;index"
Query       string         json:"query" gorm:"not null"
Category    string         json:"category" gorm:"default:all" // all, users, content, files
ResultCount int            json:"result_count" gorm:"default:0"
CreatedAt   time.Time      json:"created_at"
UpdatedAt   time.Time      json:"updated_at"
DeletedAt   gorm.DeletedAt json:"-" gorm:"index"
}

func (SearchHistory) TableName() string {
return "search_history"
}

// SearchRequest arama isteği
type SearchRequest struct {
Query      string json:"query" validate:"required"
Category   string json:"category"   // all, users, content, files
Limit      int    json:"limit"
Offset     int    json:"offset"
SortBy     string json:"sort_by"    // relevance, date, popularity
SortOrder  string json:"sort_order" // asc, desc
}

// SearchResult arama sonucu
type SearchResult struct {
ID          string      json:"id"
Type        string      json:"type" // user, content, file
Title       string      json:"title"
Description string      json:"description"
URL         string      json:"url"
Score       float64     json:"score"
Data        interface{} json:"data,omitempty"
}

// SearchResponse arama yanıtı
type SearchResponse struct {
Results    []SearchResult json:"results"
Total      int64          json:"total"
Page       int            json:"page"
Limit      int            json:"limit"
TotalPages int            json:"total_pages"
}

// SearchSuggestion öneri
type SearchSuggestion struct {
Query string json:"query"
Count int    json:"count"
}
