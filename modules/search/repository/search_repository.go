package repository

import (
"modules/search/model"

"gorm.io/gorm"
)

type SearchRepository struct {
db *gorm.DB
}

func NewSearchRepository(db *gorm.DB) *SearchRepository {
return &SearchRepository{
db: db,
}
}

// SaveSearchHistory arama geÃ§miÅŸini kaydeder
func (r *SearchRepository) SaveSearchHistory(history *model.SearchHistory) error {
return r.db.Create(history).Error
}

// GetSearchHistory kullanÄ±cÄ±nÄ±n arama geÃ§miÅŸini getirir
func (r *SearchRepository) GetSearchHistory(userID uint, limit int) ([]model.SearchHistory, error) {
var history []model.SearchHistory
err := r.db.Where("user_id = ?", userID).
Order("created_at DESC").
Limit(limit).
Find(&history).Error
return history, err
}

// GetPopularSearches popÃ¼ler aramalarÄ± getirir
func (r *SearchRepository) GetPopularSearches(limit int) ([]model.SearchSuggestion, error) {
var results []model.SearchSuggestion
err := r.db.Model(&model.SearchHistory{}).
Select("query, COUNT(*) as count").
Group("query").
Order("count DESC").
Limit(limit).
Scan(&results).Error
return results, err
}

// SearchUsers kullanÄ±cÄ± arar
func (r *SearchRepository) SearchUsers(query string, limit int) ([]map[string]interface{}, error) {
var results []map[string]interface{}
err := r.db.Table("users").
Select("id, email, name, avatar, is_active").
Where("name ILIKE ? OR email ILIKE ?", "%"+query+"%", "%"+query+"%").
Limit(limit).
Find(&results).Error
return results, err
}

// SearchFiles dosya arar
func (r *SearchRepository) SearchFiles(query string, limit int) ([]map[string]interface{}, error) {
var results []map[string]interface{}
err := r.db.Table("files").
Select("id, file_name, file_type, url, is_public").
Where("file_name ILIKE ?", "%"+query+"%").
Where("is_deleted = ?", false).
Limit(limit).
Find(&results).Error
return results, err
}
