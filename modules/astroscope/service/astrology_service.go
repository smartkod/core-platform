package service

import (
"errors"
"fmt"
"math/rand"
"time"

"modules/astroscope/data"
"modules/astroscope/model"
"modules/astroscope/repository"

"gorm.io/gorm"
)

type AstrologyService struct {
astrologyRepo *repository.AstrologyRepository
}

func NewAstrologyService(db *gorm.DB) *AstrologyService {
return &AstrologyService{
astrologyRepo: repository.NewAstrologyRepository(db),
}
}

// GetHoroscope burÃ§ yorumu getirir
func (s *AstrologyService) GetHoroscope(req *model.HoroscopeRequest) (*model.HoroscopeResponse, error) {
// VarsayÄ±lan tip
if req.Type == "" {
req.Type = "daily"
}

// BurÃ§ kontrolÃ¼
sign := s.findSign(req.Sign)
if sign == nil {
return nil, errors.New("invalid zodiac sign")
}

today := time.Now().Format("2006-01-02")

// Ã–nce veritabanÄ±ndan kontrol et
horoscope, err := s.astrologyRepo.FindHoroscope(sign.ID, today, req.Type)
if err == nil {
return s.toHoroscopeResponse(horoscope, sign), nil
}

// Yoksa yeni oluÅŸtur
return s.generateHoroscope(sign, req.Type), nil
}

// GetCompatibility uyumluluk hesaplar
func (s *AstrologyService) GetCompatibility(req *model.CompatibilityRequest) (*model.CompatibilityResponse, error) {
sign1 := s.findSign(req.Sign1)
sign2 := s.findSign(req.Sign2)

if sign1 == nil || sign2 == nil {
return nil, errors.New("invalid zodiac sign")
}

// Rastgele uyumluluk skoru (gerÃ§ek astroloji hesaplamasÄ± iÃ§in daha karmaÅŸÄ±k)
rand.Seed(time.Now().UnixNano())
score := 50 + rand.Intn(50)
loveScore := 40 + rand.Intn(60)
careerScore := 40 + rand.Intn(60)
friendshipScore := 40 + rand.Intn(60)

elements := map[string]int{"Fire": 0, "Earth": 1, "Air": 2, "Water": 3}
elementScore := 0
if elements[sign1.Element] == elements[sign2.Element] {
elementScore = 20
}

totalScore := score + elementScore
if totalScore > 100 {
totalScore = 100
}

// Uyumluluk tanÄ±mÄ±
description := ""
switch {
case totalScore >= 80:
description = "MÃ¼kemmel uyum! Bu iki burÃ§ arasÄ±nda doÄŸal bir Ã§ekim var."
case totalScore >= 60:
description = "Ä°yi uyum. Birbirinizi tamamlayabilirsiniz."
case totalScore >= 40:
description = "Orta uyum. Biraz Ã§aba gerekiyor."
default:
description = "DÃ¼ÅŸÃ¼k uyum. FarklÄ± enerjilere sahipsiniz."
}

return &model.CompatibilityResponse{
Sign1:           sign1.Name,
Sign2:           sign2.Name,
Score:           totalScore,
LoveScore:       loveScore,
CareerScore:     careerScore,
FriendshipScore: friendshipScore,
Description:     description,
Strengths:       []string{"FarklÄ± bakÄ±ÅŸ aÃ§Ä±larÄ±", "Birbirinizi tamamlama"},
Challenges:      []string{"Ä°letiÅŸim farklÄ±lÄ±klarÄ±", "Ego Ã§atÄ±ÅŸmalarÄ±"},
}, nil
}

// findSign burÃ§ adÄ±na gÃ¶re bulur
func (s *AstrologyService) findSign(name string) *model.ZodiacSign {
for _, signData := range data.ZodiacSigns {
if signData["name"] == name {
return &model.ZodiacSign{
Name:         signData["name"],
Symbol:       signData["symbol"],
Element:      signData["element"],
Quality:      signData["quality"],
RulingPlanet: signData["ruling_planet"],
StartDate:    signData["start_date"],
EndDate:      signData["end_date"],
}
}
}
return nil
}

// generateHoroscope burÃ§ yorumu oluÅŸturur
func (s *AstrologyService) generateHoroscope(sign *model.ZodiacSign, horoscopeType string) *model.HoroscopeResponse {
today := time.Now().Format("2006-01-02")

// Ã–rnek yorumlar
messages := []string{
"BugÃ¼n enerjiniz yÃ¼ksek olacak.",
"Sevdiklerinizle gÃ¼zel bir gÃ¼n geÃ§irebilirsiniz.",
"Ä°ÅŸ hayatÄ±nÄ±zda olumlu geliÅŸmeler yaÅŸanabilir.",
"Finansal konularda dikkatli olmalÄ±sÄ±nÄ±z.",
"BugÃ¼n yeni baÅŸlangÄ±Ã§lar iÃ§in uygun bir gÃ¼n.",
}

loveMessages := []string{
"AÅŸk hayatÄ±nÄ±zda sÃ¼rprizler olabilir.",
"Partnerinizle aranÄ±zda gÃ¼zel bir iletiÅŸim olacak.",
"Bekar iseniz yeni biriyle tanÄ±ÅŸabilirsiniz.",
}

careerMessages := []string{
"Kariyerinizde yeni fÄ±rsatlar kapÄ±nÄ±zÄ± Ã§alabilir.",
"Ä°ÅŸ yerinde takdir edileceksiniz.",
"Yeni projeler iÃ§in uygun bir zaman.",
}

rand.Seed(time.Now().UnixNano())

content := fmt.Sprintf("%s %s", 
messages[rand.Intn(len(messages))],
messages[rand.Intn(len(messages))],
)

return &model.HoroscopeResponse{
Sign:        sign.Name,
Date:        today,
Type:        horoscopeType,
Title:       fmt.Sprintf("%s GÃ¼nlÃ¼k BurÃ§ Yorumu", sign.Name),
Content:     content,
Love:        loveMessages[rand.Intn(len(loveMessages))],
Career:      careerMessages[rand.Intn(len(careerMessages))],
Health:      "Kendinize iyi bakÄ±n, bugÃ¼n biraz dinlenin.",
Finance:     "HarcamalarÄ±nÄ±zÄ± kontrol altÄ±nda tutun.",
LuckyNumber: rand.Intn(9) + 1,
LuckyColor:  []string{"KÄ±rmÄ±zÄ±", "Mavi", "YeÅŸil", "SarÄ±", "Mor"}[rand.Intn(5)],
LuckyStone:  []string{"Ametist", "Kuvars", "ZÃ¼mrÃ¼t", "Yakut", "Safir"}[rand.Intn(5)],
}
}

// toHoroscopeResponse model'den response oluÅŸturur
func (s *AstrologyService) toHoroscopeResponse(horoscope *model.Horoscope, sign *model.ZodiacSign) *model.HoroscopeResponse {
return &model.HoroscopeResponse{
Sign:        sign.Name,
Date:        horoscope.Date,
Type:        horoscope.Type,
Title:       horoscope.Title,
Content:     horoscope.Content,
Love:        horoscope.Love,
Career:      horoscope.Career,
Health:      horoscope.Health,
Finance:     horoscope.Finance,
LuckyNumber: horoscope.LuckyNumber,
LuckyColor:  horoscope.LuckyColor,
LuckyStone:  horoscope.LuckyStone,
}
}
