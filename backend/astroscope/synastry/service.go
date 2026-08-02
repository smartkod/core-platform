package synastry

import (
"math/rand"
)

type SynastryService struct{}

func NewSynastryService() *SynastryService {
return &SynastryService{}
}

// Calculate uyumluluk hesaplar
func (s *SynastryService) Calculate(req *SynastryRequest) (*SynastryResponse, error) {
score := 40 + rand.Intn(60)
loveScore := 30 + rand.Intn(70)
careerScore := 30 + rand.Intn(70)
friendshipScore := 30 + rand.Intn(70)

strengths := []string{
"Ä°letiÅŸimde uyum",
"Benzer deÄŸerler",
"Duygusal baÄŸlantÄ±",
}
challenges := []string{
"FarklÄ± yaklaÅŸÄ±mlar",
"Zaman yÃ¶netimi",
"Karar alma farklÄ±lÄ±klarÄ±",
}

return &SynastryResponse{
Score:           score,
LoveScore:       loveScore,
CareerScore:     careerScore,
FriendshipScore: friendshipScore,
Description:     "Ä°ki burÃ§ arasÄ±nda uyumlu bir baÄŸlantÄ± var.",
Strengths:       strengths,
Challenges:      challenges,
Aspects:         s.getAspects(),
}, nil
}

func (s *SynastryService) getAspects() []SynastryAspect {
return []SynastryAspect{
{Planet1: "GÃ¼neÅŸ", Planet2: "Ay", Type: "KavuÅŸum", Orb: 2.0, Interpretation: "Duygusal uyum"},
{Planet1: "VenÃ¼s", Planet2: "Mars", Type: "Trine", Orb: 3.0, Interpretation: "Tutkulu baÄŸlantÄ±"},
}
}
