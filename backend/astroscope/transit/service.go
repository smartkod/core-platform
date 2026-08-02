package transit

import (
"math/rand"
"time"
)

type TransitService struct{}

func NewTransitService() *TransitService {
return &TransitService{}
}

// GetDailyTransits gÃ¼nlÃ¼k transitleri getirir
func (s *TransitService) GetDailyTransits(userID uint, req *TransitRequest) (*TransitResponse, error) {
// Mock transit verileri
planets := []string{"GÃ¼neÅŸ", "Ay", "MerkÃ¼r", "VenÃ¼s", "Mars", "JÃ¼piter", "SatÃ¼rn"}
signs := []string{"KoÃ§", "BoÄŸa", "Ä°kizler", "YengeÃ§", "Aslan", "BaÅŸak", "Terazi", "Akrep", "Yay", "OÄŸlak", "Kova", "BalÄ±k"}
aspects := []string{"KavuÅŸum", "Trine", "Kare", "AltmÄ±ÅŸlÄ±k", "KarÅŸÄ±t"}
influences := []string{"positive", "challenging", "neutral"}

transits := make([]TransitData, 0)
for i := 0; i < 5; i++ {
transits = append(transits, TransitData{
Planet:    planets[rand.Intn(len(planets))],
Sign:      signs[rand.Intn(len(signs))],
House:     rand.Intn(12) + 1,
Aspect:    aspects[rand.Intn(len(aspects))],
Orb:       float64(rand.Intn(5) + 1),
Influence: influences[rand.Intn(len(influences))],
})
}

highlights := []string{
"BugÃ¼n Ay YengeÃ§ burcunda, duygusal dengeniz Ã¶nemli",
"GÃ¼neÅŸ ve JÃ¼piter arasÄ±ndaki olumlu aÃ§Ä±, yeni fÄ±rsatlar getiriyor",
"Mars aktif, enerjiniz yÃ¼ksek olacak",
}

return &TransitResponse{
Date:       req.Date,
Transits:   transits,
Summary:    "BugÃ¼n hareketli bir gÃ¼n sizi bekliyor. Ä°liÅŸkiler ve kariyer konularÄ±nda Ã¶nemli geliÅŸmeler olabilir.",
Highlights: highlights,
}, nil
}
