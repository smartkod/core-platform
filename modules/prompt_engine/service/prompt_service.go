package service

import (
"bytes"
"errors"
"fmt"
"text/template"
"time"

"modules/prompt_engine/model"
"modules/prompt_engine/repository"

"gorm.io/gorm"
)

type PromptService struct {
promptRepo *repository.PromptRepository
}

func NewPromptService(db *gorm.DB) *PromptService {
return &PromptService{
promptRepo: repository.NewPromptRepository(db),
}
}

// CreatePrompt yeni prompt oluÅŸturur
func (s *PromptService) CreatePrompt(userID uint, req *model.CreatePromptRequest) (*model.PromptResponse, error) {
// VarsayÄ±lan versiyon
if req.Version == "" {
req.Version = "1.0.0"
}

prompt := &model.PromptTemplate{
Name:        req.Name,
Category:    req.Category,
Description: req.Description,
Template:    req.Template,
Variables:   s.convertToString(req.Variables),
Version:     req.Version,
IsActive:    true,
CreatedBy:   userID,
}

if err := s.promptRepo.Create(prompt); err != nil {
return nil, err
}

return s.toResponse(prompt), nil
}

// ExecutePrompt prompt'u Ã§alÄ±ÅŸtÄ±rÄ±r
func (s *PromptService) ExecutePrompt(userID uint, req *model.ExecutePromptRequest) (*model.ExecutePromptResponse, error) {
// Template'i bul
prompt, err := s.promptRepo.FindByName(req.TemplateName)
if err != nil {
return nil, errors.New("template not found")
}

if !prompt.IsActive {
return nil, errors.New("template is not active")
}

// Template'i parse et
tmpl, err := template.New(prompt.Name).Parse(prompt.Template)
if err != nil {
return nil, fmt.Errorf("invalid template: %w", err)
}

// Template'i Ã§alÄ±ÅŸtÄ±r
startTime := time.Now()
var result bytes.Buffer
err = tmpl.Execute(&result, req.Variables)
duration := time.Since(startTime).Milliseconds()

if err != nil {
// Hata kaydet
execution := &model.PromptExecution{
TemplateID:   prompt.ID,
UserID:       userID,
Input:        s.convertToStringMap(req.Variables),
Success:      false,
ErrorMessage: err.Error(),
Duration:     duration,
}
s.promptRepo.SaveExecution(execution)
return nil, fmt.Errorf("template execution failed: %w", err)
}

// BaÅŸarÄ±lÄ± kaydÄ±
execution := &model.PromptExecution{
TemplateID: prompt.ID,
UserID:     userID,
Input:      s.convertToStringMap(req.Variables),
Output:     result.String(),
Success:    true,
Duration:   duration,
}
s.promptRepo.SaveExecution(execution)

// KullanÄ±m sayÄ±sÄ±nÄ± artÄ±r
s.promptRepo.IncrementUsage(prompt.ID)

return &model.ExecutePromptResponse{
Result:     result.String(),
Duration:   duration,
TemplateID: prompt.ID,
}, nil
}

// GetTemplates tÃ¼m prompt template'lerini getirir
func (s *PromptService) GetTemplates(category string) ([]model.PromptResponse, error) {
templates, err := s.promptRepo.GetTemplates(category)
if err != nil {
return nil, err
}

responses := make([]model.PromptResponse, len(templates))
for i, t := range templates {
responses[i] = *s.toResponse(&t)
}
return responses, nil
}

// GetTemplateByName isme gÃ¶re template getirir
func (s *PromptService) GetTemplateByName(name string) (*model.PromptResponse, error) {
prompt, err := s.promptRepo.FindByName(name)
if err != nil {
return nil, err
}
return s.toResponse(prompt), nil
}

// DeleteTemplate template siler
func (s *PromptService) DeleteTemplate(name string) error {
return s.promptRepo.DeleteByName(name)
}

// toResponse model'den response oluÅŸturur
func (s *PromptService) toResponse(prompt *model.PromptTemplate) *model.PromptResponse {
return &model.PromptResponse{
ID:          prompt.ID,
Name:        prompt.Name,
Category:    prompt.Category,
Description: prompt.Description,
Template:    prompt.Template,
Variables:   s.convertToStringSlice(prompt.Variables),
Version:     prompt.Version,
IsActive:    prompt.IsActive,
UsageCount:  prompt.UsageCount,
CreatedAt:   prompt.CreatedAt,
}
}

func (s *PromptService) convertToString(variables []string) string {
result := "["
for i, v := range variables {
if i > 0 {
result += ","
}
result += fmt.Sprintf("%q", v)
}
result += "]"
return result
}

func (s *PromptService) convertToStringSlice(variables string) []string {
// Basit parse - JSON'dan dÃ¶nÃ¼ÅŸtÃ¼r
if variables == "" || variables == "[]" {
return []string{}
}
// TODO: Proper JSON parse
return []string{}
}

func (s *PromptService) convertToStringMap(variables map[string]interface{}) string {
result := "{"
i := 0
for k, v := range variables {
if i > 0 {
result += ","
}
result += fmt.Sprintf("%q:%q", k, v)
i++
}
result += "}"
return result
}
