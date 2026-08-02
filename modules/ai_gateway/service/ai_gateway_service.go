package service

import (
"crypto/sha256"
"encoding/hex"
"errors"
"fmt"
"time"

"modules/ai_gateway/clients"
"modules/ai_gateway/model"
"modules/ai_gateway/repository"

"gorm.io/gorm"
)

type AIGatewayService struct {
aiRepo      *repository.AIRepository
openAIClient *clients.OpenAIClient
}

func NewAIGatewayService(db *gorm.DB, openAIKey, openAIBaseURL string) *AIGatewayService {
return &AIGatewayService{
aiRepo:       repository.NewAIRepository(db),
openAIClient: clients.NewOpenAIClient(openAIKey, openAIBaseURL),
}
}

// Chat AI ile sohbet
func (s *AIGatewayService) Chat(userID uint, req *model.AIRequest) (*model.AIResponse, error) {
// 1. Ã–nbellek kontrolÃ¼
promptHash := s.hashPrompt(req.Prompt)
cached, err := s.aiRepo.GetCache(req.Provider, req.Model, promptHash)
if err == nil && cached != nil {
return &model.AIResponse{
ID:        fmt.Sprintf("cache_%d", cached.ID),
Provider:  req.Provider,
Model:     req.Model,
Content:   cached.Response,
CreatedAt: cached.CreatedAt,
}, nil
}

// 2. Provider'a gÃ¶re istek yap
var response string
switch req.Provider {
case "openai":
response, err = s.openAIClient.Chat(req.Prompt, req.Model, req.MaxTokens, req.Temperature)
case "anthropic":
// TODO: Anthropic entegrasyonu
return nil, errors.New("anthropic not implemented yet")
case "gemini":
// TODO: Gemini entegrasyonu
return nil, errors.New("gemini not implemented yet")
default:
return nil, fmt.Errorf("unsupported provider: %s", req.Provider)
}

if err != nil {
return nil, err
}

// 3. Ã–nbelleÄŸe kaydet
cache := &model.AICache{
Provider:   req.Provider,
Model:      req.Model,
PromptHash: promptHash,
Response:   response,
ExpiresAt:  time.Now().Add(24 * time.Hour),
}
s.aiRepo.SaveCache(cache)

// 4. KullanÄ±m istatistiklerini kaydet
usage := &model.AIUsage{
Provider: req.Provider,
Model:    req.Model,
UserID:   userID,
}
s.aiRepo.SaveUsage(usage)

return &model.AIResponse{
ID:        fmt.Sprintf("resp_%d", time.Now().UnixNano()),
Provider:  req.Provider,
Model:     req.Model,
Content:   response,
CreatedAt: time.Now(),
}, nil
}

// hashPrompt prompt'u hash'ler
func (s *AIGatewayService) hashPrompt(prompt string) string {
hash := sha256.Sum256([]byte(prompt))
return hex.EncodeToString(hash[:])
}

// GetProviders aktif provider'larÄ± getirir
func (s *AIGatewayService) GetProviders() ([]model.AIProvider, error) {
return s.aiRepo.GetActiveProviders()
}
