package natal

import (
"time"

"gorm.io/gorm"
)

// BirthChart doÄŸum haritasÄ±
type BirthChart struct {
ID           uint           `json:"id" gorm:"primaryKey"`
UserID       uint           `json:"user_id" gorm:"uniqueIndex;not null"`
Name         string         `json:"name"`
DateOfBirth  time.Time      `json:"date_of_birth" gorm:"not null"`
TimeOfBirth  string         `json:"time_of_birth"` // HH:MM
PlaceOfBirth string         `json:"place_of_birth"`
Latitude     float64        `json:"latitude"`
Longitude    float64        `json:"longitude"`
Timezone     string         `json:"timezone"`
SunSign      string         `json:"sun_sign"`
MoonSign     string         `json:"moon_sign"`
RisingSign   string         `json:"rising_sign"`
Planets      string         `json:"planets" gorm:"type:jsonb"` // JSON
Houses       string         `json:"houses" gorm:"type:jsonb"`  // JSON
Aspects      string         `json:"aspects" gorm:"type:jsonb"` // JSON
CreatedAt    time.Time      `json:"created_at"`
UpdatedAt    time.Time      `json:"updated_at"`
DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

func (BirthChart) TableName() string {
return "birth_charts"
}

// NatalRequest doÄŸum haritasÄ± isteÄŸi
type NatalRequest struct {
Name         string  `json:"name"`
DateOfBirth  string  `json:"date_of_birth" validate:"required"` // YYYY-MM-DD
TimeOfBirth  string  `json:"time_of_birth"`                     // HH:MM
PlaceOfBirth string  `json:"place_of_birth"`
Latitude     float64 `json:"latitude"`
Longitude    float64 `json:"longitude"`
Timezone     string  `json:"timezone"`
}

// NatalResponse doÄŸum haritasÄ± yanÄ±tÄ±
type NatalResponse struct {
SunSign    string                 `json:"sun_sign"`
MoonSign   string                 `json:"moon_sign"`
RisingSign string                 `json:"rising_sign"`
Planets    map[string]PlanetData  `json:"planets"`
Houses     map[string]HouseData   `json:"houses"`
Aspects    []AspectData           `json:"aspects"`
Summary    string                 `json:"summary"`
}

// PlanetData gezegen verisi
type PlanetData struct {
Name      string  `json:"name"`
Sign      string  `json:"sign"`
Degree    float64 `json:"degree"`
House     int     `json:"house"`
Element   string  `json:"element"` // Fire, Earth, Air, Water
Quality   string  `json:"quality"` // Cardinal, Fixed, Mutable
}

// HouseData ev verisi
type HouseData struct {
Number    int     `json:"number"`
Sign      string  `json:"sign"`
Degree    float64 `json:"degree"`
Planets   []string `json:"planets"`
}

// AspectData aÃ§Ä± verisi
type AspectData struct {
Planet1   string  `json:"planet1"`
Planet2   string  `json:"planet2"`
Type      string  `json:"type"` // Conjunction, Trine, Square, etc.
Orb       float64 `json:"orb"`
Degree    float64 `json:"degree"`
}

// AIInterpretation AI yorumu
type AIInterpretation struct {
ID          uint      `json:"id" gorm:"primaryKey"`
BirthChartID uint    `json:"birth_chart_id" gorm:"not null"`
Type        string   `json:"type" gorm:"not null"` // sun, moon, rising, planet, house
Content     string   `json:"content" gorm:"type:text;not null"`
Sources     string   `json:"sources" gorm:"type:jsonb"` // Klasik kaynaklar
Confidence  int      `json:"confidence" gorm:"default:0"`
CreatedAt   time.Time `json:"created_at"`
}
