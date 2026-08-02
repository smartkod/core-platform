package transit

// TransitRequest transit isteÄŸi
type TransitRequest struct {
Date string `json:"date" validate:"required"` // YYYY-MM-DD
}

// TransitResponse transit yanÄ±tÄ±
type TransitResponse struct {
Date        string                 `json:"date"`
Transits    []TransitData          `json:"transits"`
Summary     string                 `json:"summary"`
Highlights  []string               `json:"highlights"`
}

// TransitData transit verisi
type TransitData struct {
Planet      string  `json:"planet"`
Sign        string  `json:"sign"`
House       int     `json:"house"`
Aspect      string  `json:"aspect"`
Orb         float64 `json:"orb"`
Influence   string  `json:"influence"` // positive, challenging, neutral
}
