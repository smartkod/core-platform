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

// GetHoroscope burç yorumu getirir
func (s *AstrologyService) GetHoroscope(req *model.HoroscopeRequest) (*model.HoroscopeResponse, error) {
// Varsayılan tip
if req.Type == "" {
req.Type = "daily"
}

// Burç kontrolü
sign := s.findSign(req.Sign)
if sign == nil {
return nil, errors.New("invalid zodiac sign")
}

today := time.Now().Format("2006-01-02")

// Önce veritabanından kontrol et
horoscope, err := s.astrologyRepo.FindHoroscope(sign.ID, today, req.Type)
if err == nil {
return s.toHoroscopeResponse(horoscope, sign), nil
}

// Yoksa yeni oluştur
return s.generateHoroscope(sign, req.Type), nil
}

// GetCompatibility uyumluluk hesaplar
func (s *AstrologyService) GetCompatibility(req *model.CompatibilityRequest) (*model.CompatibilityResponse, error) {
sign1 := s.findSign(req.Sign1)
sign2 := s.findSign(req.Sign2)

if sign1 == nil || sign2 == nil {
return nil, errors.New("invalid zodiac sign")
}

// Rastgele uyumluluk skoru (gerçek astroloji hesaplaması için daha karmaşık)
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

// Uyumluluk tanımı
description := ""
switch {
case totalScore >= 80:
description = "Mükemmel uyum! Bu iki burç arasında doğal bir çekim var."
case totalScore >= 60:
description = "İyi uyum. Birbirinizi tamamlayabilirsiniz."
case totalScore >= 40:
description = "Orta uyum. Biraz çaba gerekiyor."
default:
description = "Düşük uyum. Farklı enerjilere sahipsiniz."
}

return &model.CompatibilityResponse{
Sign1:           sign1.Name,
Sign2:           sign2.Name,
Score:           totalScore,
LoveScore:       loveScore,
CareerScore:     careerScore,
FriendshipScore: friendshipScore,
Description:     description,
Strengths:       []string{"Farklı bakış açıları", "Birbirinizi tamamlama"},
Challenges:      []string{"İletişim farklılıkları", "Ego çatışmaları"},
}, nil
}

// findSign burç adına göre bulur
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

// generateHoroscope burç yorumu oluşturur
func (s *AstrologyService) generateHoroscope(sign *model.ZodiacSign, horoscopeType string) *model.HoroscopeResponse {
today := time.Now().Format("2006-01-02")

// Örnek yorumlar
messages := []string{
"Bugün enerjiniz yüksek olacak.",
"Sevdiklerinizle güzel bir gün geçirebilirsiniz.",
"İş hayatınızda olumlu gelişmeler yaşanabilir.",
"Finansal konularda dikkatli olmalısınız.",
"Bugün yeni başlangıçlar için uygun bir gün.",
}

loveMessages := []string{
"Aşk hayatınızda sürprizler olabilir.",
"Partnerinizle aranızda güzel bir iletişim olacak.",
"Bekar iseniz yeni biriyle tanışabilirsiniz.",
}

careerMessages := []string{
"Kariyerinizde yeni fırsatlar kapınızı çalabilir.",
"İş yerinde takdir edileceksiniz.",
"Yeni projeler için uygun bir zaman.",
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
Title:       fmt.Sprintf("%s Günlük Burç Yorumu", sign.Name),
Content:     content,
Love:        loveMessages[rand.Intn(len(loveMessages))],
Career:      careerMessages[rand.Intn(len(careerMessages))],
Health:      "Kendinize iyi bakın, bugün biraz dinlenin.",
Finance:     "Harcamalarınızı kontrol altında tutun.",
LuckyNumber: rand.Intn(9) + 1,
LuckyColor:  []string{"Kırmızı", "Mavi", "Yeşil", "Sarı", "Mor"}[rand.Intn(5)],
LuckyStone:  []string{"Ametist", "Kuvars", "Zümrüt", "Yakut", "Safir"}[rand.Intn(5)],
}
}

// toHoroscopeResponse model'den response oluşturur
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
