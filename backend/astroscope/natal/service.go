package natal

import (
"errors"
"fmt"
"math/rand"
"time"

"gorm.io/gorm"
)

type NatalService struct {
db *gorm.DB
}

func NewNatalService(db *gorm.DB) *NatalService {
return &NatalService{db: db}
}

// CalculateNatal doÄŸum haritasÄ± hesaplar
func (s *NatalService) CalculateNatal(userID uint, req *NatalRequest) (*NatalResponse, error) {
// 1. DoÄŸum tarihini parse et
birthDate, err := time.Parse("2006-01-02", req.DateOfBirth)
if err != nil {
return nil, errors.New("geÃ§ersiz doÄŸum tarihi")
}

// 2. BurÃ§ hesapla
sunSign := s.calculateSunSign(birthDate)
moonSign := s.calculateMoonSign(birthDate)
risingSign := s.calculateRisingSign(birthDate, req.TimeOfBirth)

// 3. Gezegen verileri (mock)
planets := s.getPlanetsData(sunSign, moonSign, risingSign)

// 4. Ev verileri (mock)
houses := s.getHousesData(risingSign)

// 5. AÃ§Ä±lar (mock)
aspects := s.getAspectsData(planets)

// 6. VeritabanÄ±na kaydet
birthChart := &BirthChart{
UserID:       userID,
Name:         req.Name,
DateOfBirth:  birthDate,
TimeOfBirth:  req.TimeOfBirth,
PlaceOfBirth: req.PlaceOfBirth,
Latitude:     req.Latitude,
Longitude:    req.Longitude,
Timezone:     req.Timezone,
SunSign:      sunSign,
MoonSign:     moonSign,
RisingSign:   risingSign,
}

if err := s.db.Create(birthChart).Error; err != nil {
return nil, err
}

// 7. AI yorumu oluÅŸtur
summary := s.generateSummary(sunSign, moonSign, risingSign, planets)

return &NatalResponse{
SunSign:    sunSign,
MoonSign:   moonSign,
RisingSign: risingSign,
Planets:    planets,
Houses:     houses,
Aspects:    aspects,
Summary:    summary,
}, nil
}

// GetByUserID kullanÄ±cÄ±nÄ±n doÄŸum haritasÄ±nÄ± getirir
func (s *NatalService) GetByUserID(userID uint) (*BirthChart, error) {
var chart BirthChart
err := s.db.Where("user_id = ?", userID).First(&chart).Error
if err != nil {
return nil, err
}
return &chart, nil
}

// calculateSunSign gÃ¼neÅŸ burcunu hesaplar
func (s *NatalService) calculateSunSign(date time.Time) string {
signs := []string{
"KoÃ§", "BoÄŸa", "Ä°kizler", "YengeÃ§", "Aslan", "BaÅŸak",
"Terazi", "Akrep", "Yay", "OÄŸlak", "Kova", "BalÄ±k",
}

month := int(date.Month())
day := date.Day()

// Basit burÃ§ hesaplama
if (month == 3 && day >= 21) || (month == 4 && day <= 19) {
return "KoÃ§"
}
if (month == 4 && day >= 20) || (month == 5 && day <= 20) {
return "BoÄŸa"
}
if (month == 5 && day >= 21) || (month == 6 && day <= 20) {
return "Ä°kizler"
}
if (month == 6 && day >= 21) || (month == 7 && day <= 22) {
return "YengeÃ§"
}
if (month == 7 && day >= 23) || (month == 8 && day <= 22) {
return "Aslan"
}
if (month == 8 && day >= 23) || (month == 9 && day <= 22) {
return "BaÅŸak"
}
if (month == 9 && day >= 23) || (month == 10 && day <= 22) {
return "Terazi"
}
if (month == 10 && day >= 23) || (month == 11 && day <= 21) {
return "Akrep"
}
if (month == 11 && day >= 22) || (month == 12 && day <= 21) {
return "Yay"
}
if (month == 12 && day >= 22) || (month == 1 && day <= 19) {
return "OÄŸlak"
}
if (month == 1 && day >= 20) || (month == 2 && day <= 18) {
return "Kova"
}
if (month == 2 && day >= 19) || (month == 3 && day <= 20) {
return "BalÄ±k"
}

return "KoÃ§"
}

// calculateMoonSign ay burcunu hesaplar (mock)
func (s *NatalService) calculateMoonSign(date time.Time) string {
signs := []string{"KoÃ§", "BoÄŸa", "Ä°kizler", "YengeÃ§", "Aslan", "BaÅŸak",
"Terazi", "Akrep", "Yay", "OÄŸlak", "Kova", "BalÄ±k"}
return signs[date.Day()%12]
}

// calculateRisingSign yÃ¼kselen burcu hesaplar (mock)
func (s *NatalService) calculateRisingSign(date time.Time, timeOfBirth string) string {
signs := []string{"KoÃ§", "BoÄŸa", "Ä°kizler", "YengeÃ§", "Aslan", "BaÅŸak",
"Terazi", "Akrep", "Yay", "OÄŸlak", "Kova", "BalÄ±k"}
return signs[(date.Day()+len(timeOfBirth))%12]
}

// getPlanetsData gezegen verilerini getirir (mock)
func (s *NatalService) getPlanetsData(sun, moon, rising string) map[string]PlanetData {
return map[string]PlanetData{
"GÃ¼neÅŸ": {Name: "GÃ¼neÅŸ", Sign: sun, Degree: 15.5, House: 1, Element: s.getElement(sun), Quality: s.getQuality(sun)},
"Ay":     {Name: "Ay", Sign: moon, Degree: 22.3, House: 4, Element: s.getElement(moon), Quality: s.getQuality(moon)},
"MerkÃ¼r": {Name: "MerkÃ¼r", Sign: sun, Degree: 8.7, House: 2, Element: "Hava", Quality: "DeÄŸiÅŸken"},
"VenÃ¼s":  {Name: "VenÃ¼s", Sign: "BoÄŸa", Degree: 12.1, House: 3, Element: "Toprak", Quality: "Sabit"},
"Mars":   {Name: "Mars", Sign: "Aslan", Degree: 25.9, House: 5, Element: "AteÅŸ", Quality: "Sabit"},
}
}

// getHousesData ev verilerini getirir (mock)
func (s *NatalService) getHousesData(rising string) map[string]HouseData {
return map[string]HouseData{
"1":  {Number: 1, Sign: rising, Degree: 0.0, Planets: []string{"GÃ¼neÅŸ"}},
"2":  {Number: 2, Sign: "BoÄŸa", Degree: 30.0, Planets: []string{"MerkÃ¼r"}},
"3":  {Number: 3, Sign: "Ä°kizler", Degree: 60.0, Planets: []string{"VenÃ¼s"}},
"4":  {Number: 4, Sign: "YengeÃ§", Degree: 90.0, Planets: []string{"Ay"}},
"5":  {Number: 5, Sign: "Aslan", Degree: 120.0, Planets: []string{"Mars"}},
"6":  {Number: 6, Sign: "BaÅŸak", Degree: 150.0, Planets: []string{}},
"7":  {Number: 7, Sign: "Terazi", Degree: 180.0, Planets: []string{}},
"8":  {Number: 8, Sign: "Akrep", Degree: 210.0, Planets: []string{}},
"9":  {Number: 9, Sign: "Yay", Degree: 240.0, Planets: []string{}},
"10": {Number: 10, Sign: "OÄŸlak", Degree: 270.0, Planets: []string{}},
"11": {Number: 11, Sign: "Kova", Degree: 300.0, Planets: []string{}},
"12": {Number: 12, Sign: "BalÄ±k", Degree: 330.0, Planets: []string{}},
}
}

// getAspectsData aÃ§Ä± verilerini getirir (mock)
func (s *NatalService) getAspectsData(planets map[string]PlanetData) []AspectData {
return []AspectData{
{Planet1: "GÃ¼neÅŸ", Planet2: "Ay", Type: "KavuÅŸum", Orb: 2.0, Degree: 0.0},
{Planet1: "GÃ¼neÅŸ", Planet2: "MerkÃ¼r", Type: "Trine", Orb: 3.0, Degree: 120.0},
{Planet1: "VenÃ¼s", Planet2: "Mars", Type: "Kare", Orb: 2.5, Degree: 90.0},
}
}

// getElement element dÃ¶ndÃ¼rÃ¼r
func (s *NatalService) getElement(sign string) string {
elements := map[string]string{
"KoÃ§": "AteÅŸ", "Aslan": "AteÅŸ", "Yay": "AteÅŸ",
"BoÄŸa": "Toprak", "BaÅŸak": "Toprak", "OÄŸlak": "Toprak",
"Ä°kizler": "Hava", "Terazi": "Hava", "Kova": "Hava",
"YengeÃ§": "Su", "Akrep": "Su", "BalÄ±k": "Su",
}
return elements[sign]
}

// getQuality quality dÃ¶ndÃ¼rÃ¼r
func (s *NatalService) getQuality(sign string) string {
qualities := map[string]string{
"KoÃ§": "Ã–ncÃ¼", "YengeÃ§": "Ã–ncÃ¼", "Terazi": "Ã–ncÃ¼", "OÄŸlak": "Ã–ncÃ¼",
"BoÄŸa": "Sabit", "Aslan": "Sabit", "Akrep": "Sabit", "Kova": "Sabit",
"Ä°kizler": "DeÄŸiÅŸken", "BaÅŸak": "DeÄŸiÅŸken", "Yay": "DeÄŸiÅŸken", "BalÄ±k": "DeÄŸiÅŸken",
}
return qualities[sign]
}

// generateSummary AI yorumu oluÅŸturur
func (s *NatalService) generateSummary(sun, moon, rising string, planets map[string]PlanetData) string {
summaries := []string{
fmt.Sprintf("%s burcu, doÄŸal liderlik ve yaratÄ±cÄ±lÄ±k Ã¶zelliklerini gÃ¶sterir.", sun),
fmt.Sprintf("%s burcu, duygusal derinlik ve sezgisel yeteneklerle iliÅŸkilidir.", moon),
fmt.Sprintf("%s yÃ¼kselen burcu, dÄ±ÅŸ dÃ¼nyaya nasÄ±l gÃ¶rÃ¼ndÃ¼ÄŸÃ¼nÃ¼zÃ¼ etkiler.", rising),
}
return summaries[rand.Intn(len(summaries))]
}
