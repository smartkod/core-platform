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

// GetOrCreateSession kullanÄ±cÄ±nÄ±n oturumunu getirir veya oluÅŸturur
func (s *CopilotService) GetOrCreateSession(userID uint) (*model.CopilotSession, error) {
session, err := s.copilotRepo.FindByUserID(userID)
if err != nil {
if err == gorm.ErrRecordNotFound {
// Yeni oturum oluÅŸtur
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

// Ask soru sor ve yanÄ±t al
func (s *CopilotService) Ask(userID uint, req *model.CopilotRequest) (*model.CopilotResponse, error) {
// Oturumu al
session, err := s.GetOrCreateSession(userID)
if err != nil {
return nil, err
}

// KullanÄ±cÄ± baÄŸlamÄ±nÄ± oluÅŸtur
context := s.buildContext(userID, session)

// Soruyu iÅŸle
answer, sources, confidence := s.processQuery(req.Query, context)

// YanÄ±tÄ± oluÅŸtur
response := &model.CopilotResponse{
ID:         fmt.Sprintf("copilot_%d", time.Now().UnixNano()),
Answer:     answer,
Context:    context,
Sources:    sources,
Confidence: confidence,
CreatedAt:  time.Now(),
}

// MesajlarÄ± kaydet
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

// GetHistory sohbet geÃ§miÅŸini getir
func (s *CopilotService) GetHistory(userID uint, limit int) ([]model.CopilotMessage, error) {
session, err := s.GetOrCreateSession(userID)
if err != nil {
return nil, err
}
return s.copilotRepo.GetMessages(session.ID, limit)
}

// buildContext kullanÄ±cÄ± baÄŸlamÄ±nÄ± oluÅŸtur
func (s *CopilotService) buildContext(userID uint, session *model.CopilotSession) map[string]interface{} {
context := map[string]interface{}{
"user_id": userID,
"learning_level": "intermediate",
"preferences": map[string]string{
"language": "tr",
"style":    "classical",
},
}

// VarsayÄ±lan doÄŸum haritasÄ± (mock)
context["birth_chart"] = map[string]interface{}{
"sun_sign":    "Leo",
"moon_sign":   "Taurus",
"rising_sign": "Sagittarius",
}

// VarsayÄ±lan transitler (mock)
context["current_transits"] = map[string]interface{}{
"sun":    "Leo",
"moon":   "Scorpio",
"mercury": "Virgo",
"venus":  "Cancer",
"mars":   "Gemini",
}

// Son olaylar (mock)
context["recent_events"] = []string{
"Yeni eve taÅŸÄ±nma (Temmuz 2026)",
"Yeni iÅŸe baÅŸlama (Temmuz 2026)",
}

return context
}

// processQuery soruyu iÅŸle
func (s *CopilotService) processQuery(query string, context map[string]interface{}) (string, []string, int) {
// Ã–rnek yanÄ±tlar
answers := []string{
"Bu dÃ¶nemde, 4. ev ve IC temalarÄ± aktif. TaÅŸÄ±nma, ev deÄŸiÅŸikliÄŸi veya ailevi konular Ã¶ne Ã§Ä±kabilir. SatÃ¼rn 4. ev transitinde, kalÄ±cÄ± deÄŸiÅŸiklikler yapma zamanÄ±.",
"Kariyerinizde JÃ¼piter 10. ev transitinde geniÅŸleme ve fÄ±rsatlar gÃ¶zleniyor. Bu dÃ¶nem, terfi, yeni iÅŸ teklifleri veya iÅŸ deÄŸiÅŸikliÄŸi iÃ§in uygun.",
"Ä°liÅŸkilerde VenÃ¼s 7. evde, partnerinizle aranÄ±zda uyum ve derinleÅŸen bir baÄŸ var. Bekar iseniz, yeni bir iliÅŸki baÅŸlangÄ±cÄ± olabilir.",
"Finansal konularda, SatÃ¼rn 2. ev transitinde dikkatli olun. HarcamalarÄ± kontrol altÄ±nda tutmak ve yatÄ±rÄ±mlarÄ± gÃ¶zden geÃ§irmek iyi olur.",
"SaÄŸlÄ±ÄŸÄ±nÄ±za dikkat edin. Mars 6. evde enerjik hissettirse de, aÅŸÄ±rÄ± yÃ¼klenmekten kaÃ§Ä±nÄ±n. DÃ¼zenli egzersiz ve dengeli beslenme Ã¶nemli.",
}

sources := []string{
"Ptolemy - Tetrabiblos II.3",
"William Lilly - Christian Astrology",
"Guido Bonatti - Book of Astronomy",
"Abu Ma'shar - Great Introduction",
}

// Rastgele yanÄ±t seÃ§
answer := answers[time.Now().UnixNano()%int64(len(answers))]
confidence := 70 + int(time.Now().UnixNano()%20)

return answer, sources, confidence
}

// marshalContext context'i JSON'a dÃ¶nÃ¼ÅŸtÃ¼r
func (s *CopilotService) marshalContext(context map[string]interface{}) string {
data, _ := json.Marshal(context)
return string(data)
}

// getDefaultContext varsayÄ±lan baÄŸlamÄ± getir
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

// generateSessionID benzersiz session ID oluÅŸtur
func (s *CopilotService) generateSessionID() string {
bytes := make([]byte, 16)
rand.Read(bytes)
return hex.EncodeToString(bytes)
}
