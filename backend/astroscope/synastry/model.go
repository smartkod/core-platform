package synastry

// SynastryRequest uyumluluk isteÄŸi
type SynastryRequest struct {
UserID1 uint `json:"user_id1" validate:"required"`
UserID2 uint `json:"user_id2" validate:"required"`
}

// SynastryResponse uyumluluk yanÄ±tÄ±
type SynastryResponse struct {
Score          int                 `json:"score"` // 0-100
LoveScore      int                 `json:"love_score"`
CareerScore    int                 `json:"career_score"`
FriendshipScore int                `json:"friendship_score"`
Description    string              `json:"description"`
Strengths      []string            `json:"strengths"`
Challenges     []string            `json:"challenges"`
Aspects        []SynastryAspect    `json:"aspects"`
}

// SynastryAspect uyumluluk aÃ§Ä±sÄ±
type SynastryAspect struct {
Planet1     string  `json:"planet1"`
Planet2     string  `json:"planet2"`
Type        string  `json:"type"`
Orb         float64 `json:"orb"`
Interpretation string `json:"interpretation"`
}
