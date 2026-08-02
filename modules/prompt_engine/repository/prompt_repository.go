package repository

import (
"modules/prompt_engine/model"

"gorm.io/gorm"
)

type PromptRepository struct {
db *gorm.DB
}

func NewPromptRepository(db *gorm.DB) *PromptRepository {
return &PromptRepository{
db: db,
}
}

func (r *PromptRepository) Create(prompt *model.PromptTemplate) error {
return r.db.Create(prompt).Error
}

func (r *PromptRepository) FindByName(name string) (*model.PromptTemplate, error) {
var prompt model.PromptTemplate
err := r.db.Where("name = ?", name).First(&prompt).Error
if err != nil {
return nil, err
}
return &prompt, nil
}

func (r *PromptRepository) GetTemplates(category string) ([]model.PromptTemplate, error) {
var templates []model.PromptTemplate
query := r.db.Where("is_active = ?", true)
if category != "" {
query = query.Where("category = ?", category)
}
err := query.Order("usage_count DESC").Find(&templates).Error
return templates, err
}

func (r *PromptRepository) IncrementUsage(id uint) error {
return r.db.Model(&model.PromptTemplate{}).Where("id = ?", id).
Update("usage_count", gorm.Expr("usage_count + 1")).Error
}

func (r *PromptRepository) SaveExecution(execution *model.PromptExecution) error {
return r.db.Create(execution).Error
}

func (r *PromptRepository) DeleteByName(name string) error {
return r.db.Where("name = ?", name).Delete(&model.PromptTemplate{}).Error
}
