package service

import (
"crypto/rand"
"encoding/hex"
"encoding/json"
"fmt"
"time"

"modules/ai_copilot/model"
"modules/ai_copilot/repository"

"gorm.io/gorm"
)

type CopilotService struct {
copilotRepo *repository.CopilotRepository
}

func NewCopilotService(db *gorm.DB) *CopilotService {
return &CopilotService{
copilotRepo: repository.NewCopilotRepository(db),
}
}

// GetOrCreateSession kullanıcının oturumunu getirir veya oluşturur
func (s *CopilotService) GetOrCreateSession(userID uint) (*model.CopilotSession, error) {
session, err := s.copilotRepo.FindByUserID(userID)
if err != nil {
if err == gorm.ErrRecordNotFound {
// Yeni oturum oluştur
sessionID := s.generateSessionID()
session = &model.CopilotSession{
UserID:    userID,
SessionID: sessionID,
Context:   s.getDefaultContext(userID),
}
if err := s.copilotRepo.CreateSession(session); err != nil {
return nil, err
}
return session, nil
}
return nil, err
}
return session, nil
}

// Ask soru sor ve yanıt al
func (s *CopilotService) Ask(userID uint, req *model.CopilotRequest) (*model.CopilotResponse, error) {
// Oturumu al
session, err := s.GetOrCreateSession(userID)
if err != nil {
return nil, err
}

// Kullanıcı bağlamını oluştur
context := s.buildContext(userID, session)

// Soruyu işle
answer, sources, confidence := s.processQuery(req.Query, context)

// Yanıtı oluştur
response := &model.CopilotResponse{
ID:         fmt.Sprintf("copilot_%d", time.Now().UnixNano()),
Answer:     answer,
Context:    context,
Sources:    sources,
Confidence: confidence,
CreatedAt:  time.Now(),
}

// Mesajları kaydet
userMsg := &model.CopilotMessage{
SessionID: session.ID,
Role:      "user",
Content:   req.Query,
Tokens:    len(req.Query) / 4,
}
s.copilotRepo.CreateMessage(userMsg)

assistantMsg := &model.CopilotMessage{
SessionID: session.ID,
Role:      "assistant",
Content:   answer,
Context:   s.marshalContext(context),
Tokens:    len(answer) / 4,
}
s.copilotRepo.CreateMessage(assistantMsg)

return response, nil
}

// GetHistory sohbet geçmişini getir
func (s *CopilotService) GetHistory(userID uint, limit int) ([]model.CopilotMessage, error) {
session, err := s.GetOrCreateSession(userID)
if err != nil {
return nil, err
}
return s.copilotRepo.GetMessages(session.ID, limit)
}

// buildContext kullanıcı bağlamını oluştur
func (s *CopilotService) buildContext(userID uint, session *model.CopilotSession) map[string]interface{} {
context := map[string]interface{}{
"user_id": userID,
"learning_level": "intermediate",
"preferences": map[string]string{
"language": "tr",
"style":    "classical",
},
}

// Varsayılan doğum haritası (mock)
context["birth_chart"] = map[string]interface{}{
"sun_sign":    "Leo",
"moon_sign":   "Taurus",
"rising_sign": "Sagittarius",
}

// Varsayılan transitler (mock)
context["current_transits"] = map[string]interface{}{
"sun":    "Leo",
"moon":   "Scorpio",
"mercury": "Virgo",
"venus":  "Cancer",
"mars":   "Gemini",
}

// Son olaylar (mock)
context["recent_events"] = []string{
"Yeni eve taşınma (Temmuz 2026)",
"Yeni işe başlama (Temmuz 2026)",
}

return context
}

// processQuery soruyu işle
func (s *CopilotService) processQuery(query string, context map[string]interface{}) (string, []string, int) {
// Örnek yanıtlar
answers := []string{
"Bu dönemde, 4. ev ve IC temaları aktif. Taşınma, ev değişikliği veya ailevi konular öne çıkabilir. Satürn 4. ev transitinde, kalıcı değişiklikler yapma zamanı.",
"Kariyerinizde Jüpiter 10. ev transitinde genişleme ve fırsatlar gözleniyor. Bu dönem, terfi, yeni iş teklifleri veya iş değişikliği için uygun.",
"İlişkilerde Venüs 7. evde, partnerinizle aranızda uyum ve derinleşen bir bağ var. Bekar iseniz, yeni bir ilişki başlangıcı olabilir.",
"Finansal konularda, Satürn 2. ev transitinde dikkatli olun. Harcamaları kontrol altında tutmak ve yatırımları gözden geçirmek iyi olur.",
"Sağlığınıza dikkat edin. Mars 6. evde enerjik hissettirse de, aşırı yüklenmekten kaçının. Düzenli egzersiz ve dengeli beslenme önemli.",
}

sources := []string{
"Ptolemy - Tetrabiblos II.3",
"William Lilly - Christian Astrology",
"Guido Bonatti - Book of Astronomy",
"Abu Ma'shar - Great Introduction",
}

// Rastgele yanıt seç
answer := answers[time.Now().UnixNano()%int64(len(answers))]
confidence := 70 + int(time.Now().UnixNano()%20)

return answer, sources, confidence
}

// marshalContext context'i JSON'a dönüştür
func (s *CopilotService) marshalContext(context map[string]interface{}) string {
data, _ := json.Marshal(context)
return string(data)
}

// getDefaultContext varsayılan bağlamı getir
func (s *CopilotService) getDefaultContext(userID uint) string {
context := map[string]interface{}{
"user_id":         userID,
"learning_level":  "beginner",
"preferences":     map[string]string{"language": "tr"},
"birth_chart":     map[string]string{"sun_sign": "Leo"},
"current_transits": map[string]string{"sun": "Leo"},
}
data, _ := json.Marshal(context)
return string(data)
}

// generateSessionID benzersiz session ID oluştur
func (s *CopilotService) generateSessionID() string {
bytes := make([]byte, 16)
rand.Read(bytes)
return hex.EncodeToString(bytes)
}
