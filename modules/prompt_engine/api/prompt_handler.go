package api

import (
"net/http"

"modules/prompt_engine/model"
"modules/prompt_engine/service"

"github.com/labstack/echo/v4"
)

type PromptHandler struct {
promptService *service.PromptService
}

func NewPromptHandler(promptService *service.PromptService) *PromptHandler {
return &PromptHandler{
promptService: promptService,
}
}

// CreatePrompt yeni prompt oluÅŸturur
func (h *PromptHandler) CreatePrompt(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1)

var req model.CreatePromptRequest
if err := c.Bind(&req); err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": "Invalid request",
})
}

response, err := h.promptService.CreatePrompt(userID, &req)
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusCreated, response)
}

// ExecutePrompt prompt Ã§alÄ±ÅŸtÄ±rÄ±r
func (h *PromptHandler) ExecutePrompt(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1)

var req model.ExecutePromptRequest
if err := c.Bind(&req); err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": "Invalid request",
})
}

response, err := h.promptService.ExecutePrompt(userID, &req)
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, response)
}

// GetTemplates prompt template'lerini getirir
func (h *PromptHandler) GetTemplates(c echo.Context) error {
category := c.QueryParam("category")
templates, err := h.promptService.GetTemplates(category)
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}
return c.JSON(http.StatusOK, templates)
}

// GetTemplateByName isme gÃ¶re template getirir
func (h *PromptHandler) GetTemplateByName(c echo.Context) error {
name := c.Param("name")
template, err := h.promptService.GetTemplateByName(name)
if err != nil {
return c.JSON(http.StatusNotFound, map[string]string{
"error": err.Error(),
})
}
return c.JSON(http.StatusOK, template)
}

// DeleteTemplate template siler
func (h *PromptHandler) DeleteTemplate(c echo.Context) error {
name := c.Param("name")
if err := h.promptService.DeleteTemplate(name); err != nil {
return c.JSON(http.StatusNotFound, map[string]string{
"error": err.Error(),
})
}
return c.JSON(http.StatusOK, map[string]string{
"message": "Template deleted successfully",
})
}
