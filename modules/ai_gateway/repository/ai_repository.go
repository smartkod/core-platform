package repository

import (
"modules/ai_gateway/model"

"gorm.io/gorm"
)

type AIRepository struct {
db *gorm.DB
}

func NewAIRepository(db *gorm.DB) *AIRepository {
return &AIRepository{
db: db,
}
}

func (r *AIRepository) GetCache(provider, model, promptHash string) (*model.AICache, error) {
var cache model.AICache
err := r.db.Where("provider = ? AND model = ? AND prompt_hash = ? AND expires_at > ?",
provider, model, promptHash, gorm.Expr("NOW()")).
First(&cache).Error
if err != nil {
return nil, err
}
return &cache, nil
}

func (r *AIRepository) SaveCache(cache *model.AICache) error {
return r.db.Create(cache).Error
}

func (r *AIRepository) SaveUsage(usage *model.AIUsage) error {
return r.db.Create(usage).Error
}

func (r *AIRepository) GetActiveProviders() ([]model.AIProvider, error) {
var providers []model.AIProvider
err := r.db.Where("is_active = ?", true).
Order("priority ASC").
Find(&providers).Error
return providers, err
}
