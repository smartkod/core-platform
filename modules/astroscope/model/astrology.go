package model

import (
"time"

"gorm.io/gorm"
)

// ZodiacSign burç
type ZodiacSign struct {
ID          uint           json:"id" gorm:"primaryKey"
Name        string         json:"name" gorm:"unique;not null" // Aries, Taurus, Gemini, ...
Symbol      string         json:"symbol" gorm:"not null"      // ♈, ♉, ♊, ...
Element     string         json:"element"                     // Fire, Earth, Air, Water
Quality     string         json:"quality"                     // Cardinal, Fixed, Mutable
RulingPlanet string        json:"ruling_planet"
StartDate   string         json:"start_date"                  // MM-DD
EndDate     string         json:"end_date"                    // MM-DD
Description string         json:"description"
CreatedAt   time.Time      json:"created_at"
UpdatedAt   time.Time      json:"updated_at"
DeletedAt   gorm.DeletedAt json:"-" gorm:"index"
}

func (ZodiacSign) TableName() string {
return "zodiac_signs"
}

// Horoscope burç yorumu
type Horoscope struct {
ID          uint           json:"id" gorm:"primaryKey"
SignID      uint           json:"sign_id" gorm:"not null"
Date        string         json:"date" gorm:"not null" // YYYY-MM-DD
Type        string         json:"type" gorm:"not null" // daily, weekly, monthly
Title       string         json:"title"
Content     string         json:"content" gorm:"type:text;not null"
Love        string         json:"love" gorm:"type:text"
Career      string         json:"career" gorm:"type:text"
Health      string         json:"health" gorm:"type:text"
Finance     string         json:"finance" gorm:"type:text"
LuckyNumber int            json:"lucky_number"
LuckyColor  string         json:"lucky_color"
LuckyStone  string         json:"lucky_stone"
CreatedAt   time.Time      json:"created_at"
UpdatedAt   time.Time      json:"updated_at"
DeletedAt   gorm.DeletedAt json:"-" gorm:"index"
}

func (Horoscope) TableName() string {
return "horoscopes"
}

// BirthChart doğum haritası
type BirthChart struct {
ID          uint           json:"id" gorm:"primaryKey"
UserID      uint           json:"user_id" gorm:"uniqueIndex;not null"
Name        string         json:"name"
DateOfBirth time.Time      json:"date_of_birth" gorm:"not null"
TimeOfBirth string         json:"time_of_birth" // HH:MM
PlaceOfBirth string        json:"place_of_birth"
Latitude    float64        json:"latitude"
Longitude   float64        json:"longitude"
SunSign     string         json:"sun_sign"
MoonSign    string         json:"moon_sign"
RisingSign  string         json:"rising_sign"
ChartData   string         json:"chart_data" gorm:"type:jsonb"
CreatedAt   time.Time      json:"created_at"
UpdatedAt   time.Time      json:"updated_at"
DeletedAt   gorm.DeletedAt json:"-" gorm:"index"
}

func (BirthChart) TableName() string {
return "birth_charts"
}

// CompatibilityRequest uyumluluk isteği
type CompatibilityRequest struct {
Sign1 string json:"sign1" validate:"required"
Sign2 string json:"sign2" validate:"required"
}

// CompatibilityResponse uyumluluk yanıtı
type CompatibilityResponse struct {
Sign1          string  json:"sign1"
Sign2          string  json:"sign2"
Score          int     json:"score" // 0-100
LoveScore      int     json:"love_score"
CareerScore    int     json:"career_score"
FriendshipScore int    json:"friendship_score"
Description    string  json:"description"
Strengths      []string json:"strengths"
Challenges     []string json:"challenges"
}

// HoroscopeRequest burç yorumu isteği
type HoroscopeRequest struct {
Sign string json:"sign" validate:"required"
Type string json:"type" // daily, weekly, monthly
}

// HoroscopeResponse burç yorumu yanıtı
type HoroscopeResponse struct {
Sign        string json:"sign"
Date        string json:"date"
Type        string json:"type"
Title       string json:"title"
Content     string json:"content"
Love        string json:"love"
Career      string json:"career"
Health      string json:"health"
Finance     string json:"finance"
LuckyNumber int    json:"lucky_number"
LuckyColor  string json:"lucky_color"
LuckyStone  string json:"lucky_stone"
}
