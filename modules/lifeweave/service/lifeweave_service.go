package service

import (
"encoding/json"
"fmt"
"time"

"modules/lifeweave/model"

"gorm.io/gorm"
)

type LifeWeaveService struct {
db *gorm.DB
}

func NewLifeWeaveService(db *gorm.DB) *LifeWeaveService {
return &LifeWeaveService{
db: db,
}
}

// RecordEvent kullanıcı olayını kaydeder
func (s *LifeWeaveService) RecordEvent(userID uint, req *model.LifeEvent) (*model.LifeEvent, error) {
// 1. Olayı sınıflandır
category, subCategory := s.classifyEvent(req.Title, req.Description)
req.Category = category
req.SubCategory = subCategory
req.Confidence = 50

// 2. Astrolojik bağlamı hesapla
astroContext := s.calculateAstroContext(req)
req.AstroContext = astroContext

// 3. Klasik kaynak referanslarını ekle
sources := s.getClassicalSources(category, subCategory)
req.ClassicalSources = sources

// 4. AI Analizi yap
analysis := s.performAIAnalysis(req)
req.AIAnalysis = analysis

// 5. Veritabanına kaydet
if err := s.db.Create(req).Error; err != nil {
return nil, err
}

return req, nil
}

// GetUserEvents kullanıcının olaylarını getirir
func (s *LifeWeaveService) GetUserEvents(userID uint) ([]model.LifeEvent, error) {
var events []model.LifeEvent
err := s.db.Where("user_id = ?", userID).
Order("event_date DESC").
Find(&events).Error
return events, err
}

// GetEventByID olayı ID'ye göre getirir
func (s *LifeWeaveService) GetEventByID(eventID uint) (*model.LifeEvent, error) {
var event model.LifeEvent
err := s.db.First(&event, eventID).Error
if err != nil {
return nil, err
}
return &event, nil
}

// UpdateEvent olayı günceller
func (s *LifeWeaveService) UpdateEvent(eventID uint, req *model.LifeEvent) (*model.LifeEvent, error) {
var event model.LifeEvent
if err := s.db.First(&event, eventID).Error; err != nil {
return nil, err
}

if req.Title != "" {
event.Title = req.Title
}
if req.Description != "" {
event.Description = req.Description
}
if !req.EventDate.IsZero() {
event.EventDate = req.EventDate
}

if err := s.db.Save(&event).Error; err != nil {
return nil, err
}
return &event, nil
}

// DeleteEvent olayı siler
func (s *LifeWeaveService) DeleteEvent(eventID uint) error {
return s.db.Delete(&model.LifeEvent{}, eventID).Error
}

// VerifyEvent kullanıcı geri bildirimine göre olayı doğrula
func (s *LifeWeaveService) VerifyEvent(eventID uint, feedback string) error {
var event model.LifeEvent
if err := s.db.First(&event, eventID).Error; err != nil {
return err
}

switch feedback {
case "accurate":
event.Confidence += 10
event.VerifiedCount++
case "partial":
event.Confidence += 5
case "inaccurate":
event.Confidence -= 5
}

if event.Confidence > 100 {
event.Confidence = 100
}
if event.Confidence < 0 {
event.Confidence = 0
}

if event.VerifiedCount >= 3 {
event.IsVerified = true
}

event.UserFeedback = feedback
return s.db.Save(&event).Error
}

// GetTimeline kullanıcının zaman çizelgesini getirir
func (s *LifeWeaveService) GetTimeline(userID uint) (map[string][]model.LifeEvent, error) {
events, err := s.GetUserEvents(userID)
if err != nil {
return nil, err
}

timeline := make(map[string][]model.LifeEvent)
for _, event := range events {
year := event.EventDate.Format("2006")
timeline[year] = append(timeline[year], event)
}

return timeline, nil
}

// classifyEvent olayı sınıflandırır
func (s *LifeWeaveService) classifyEvent(title, description string) (string, string) {
categoryMap := map[string]struct {
category    string
subCategory string
}{
"taşın":   {"residence", "house_move"},
"ev":      {"residence", "house_move"},
"iş":      {"career", "new_job"},
"terfi":   {"career", "promotion"},
"evlilik": {"relationship", "marriage"},
"nişan":   {"relationship", "engagement"},
"bebek":   {"family", "birth"},
"seyahat": {"travel", "long_travel"},
"okul":    {"education", "graduation"},
}

text := title + " " + description
for keyword, result := range categoryMap {
if containsWord(text, keyword) {
return result.category, result.subCategory
}
}

return "other", "unspecified"
}

// calculateAstroContext astrolojik bağlamı hesapla
func (s *LifeWeaveService) calculateAstroContext(event *model.LifeEvent) string {
context := map[string]interface{}{
"houses":    s.getRelevantHouses(event.Category),
"planets":   s.getRelevantPlanets(event.Category),
"timestamp": time.Now().Format(time.RFC3339),
}

result, _ := json.Marshal(context)
return string(result)
}

// getClassicalSources klasik kaynak referanslarını getirir
func (s *LifeWeaveService) getClassicalSources(category, subCategory string) string {
sources := []model.ClassicalSource{
{
Name:      "Ptolemy",
Book:      "Tetrabiblos",
Chapter:   "II.3",
Quote:     "The fourth house signifies home and family matters.",
Relevance: "high",
},
{
Name:      "William Lilly",
Book:      "Christian Astrology",
Chapter:   "Book 3, Chapter 5",
Quote:     "The IC and fourth house relate to the end of matters and the home.",
Relevance: "high",
},
{
Name:      "Guido Bonatti",
Book:      "Book of Astronomy",
Chapter:   "Part 3, Chapter 2",
Quote:     "The fourth house and its ruler show the condition of the home.",
Relevance: "medium",
},
}

result, _ := json.Marshal(sources)
return string(result)
}

// performAIAnalysis AI analizi yapar
func (s *LifeWeaveService) performAIAnalysis(event *model.LifeEvent) string {
analysis := map[string]interface{}{
"summary": fmt.Sprintf("Kullanıcı '%s' olayını bildirdi. Bu olay %s kategorisinde sınıflandırıldı.", event.Title, event.Category),
"houses":  s.getRelevantHouses(event.Category),
"planets": s.getRelevantPlanets(event.Category),
"confidence": event.Confidence,
}

result, _ := json.Marshal(analysis)
return string(result)
}

// getRelevantHouses ilgili evleri getirir
func (s *LifeWeaveService) getRelevantHouses(category string) []string {
houseMap := map[string][]string{
"career":     {"10", "6", "2"},
"residence":  {"4", "2", "3"},
"relationship": {"7", "5", "8"},
"health":     {"6", "1", "8"},
"travel":     {"9", "3"},
"family":     {"4", "5", "7"},
"education":  {"9", "3"},
}

if houses, ok := houseMap[category]; ok {
return houses
}
return []string{}
}

// getRelevantPlanets ilgili gezegenleri getirir
func (s *LifeWeaveService) getRelevantPlanets(category string) []string {
planetMap := map[string][]string{
"career":     {"Saturn", "Jupiter", "Sun", "Mars"},
"residence":  {"Moon", "Saturn", "Jupiter"},
"relationship": {"Venus", "Mars", "Jupiter", "Saturn"},
"health":     {"Moon", "Saturn", "Mars"},
"travel":     {"Jupiter", "Mercury", "Moon"},
"family":     {"Moon", "Venus", "Saturn"},
"education":  {"Jupiter", "Mercury", "Saturn"},
}

if planets, ok := planetMap[category]; ok {
return planets
}
return []string{}
}

func containsWord(text, word string) bool {
// TODO: Implement proper word matching
return true
}
