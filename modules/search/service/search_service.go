package service

import (
"math"

"modules/search/model"
"modules/search/repository"

"gorm.io/gorm"
)

type SearchService struct {
searchRepo *repository.SearchRepository
}

func NewSearchService(db *gorm.DB) *SearchService {
return &SearchService{
searchRepo: repository.NewSearchRepository(db),
}
}

// Search genel arama yapar
func (s *SearchService) Search(userID uint, req *model.SearchRequest) (*model.SearchResponse, error) {
// Varsayılan değerler
if req.Limit <= 0 {
req.Limit = 20
}
if req.Offset < 0 {
req.Offset = 0
}
if req.Category == "" {
req.Category = "all"
}
if req.SortBy == "" {
req.SortBy = "relevance"
}
if req.SortOrder == "" {
req.SortOrder = "desc"
}

var results []model.SearchResult
var total int64

// Kategoriye göre arama
switch req.Category {
case "users":
users, err := s.searchRepo.SearchUsers(req.Query, req.Limit)
if err != nil {
return nil, err
}
for _, user := range users {
results = append(results, model.SearchResult{
ID:          user["id"].(uint),
Type:        "user",
Title:       user["name"].(string),
Description: user["email"].(string),
URL:         "/users/" + user["id"].(uint),
Score:       1.0,
Data:        user,
})
}
total = int64(len(users))

case "files":
files, err := s.searchRepo.SearchFiles(req.Query, req.Limit)
if err != nil {
return nil, err
}
for _, file := range files {
results = append(results, model.SearchResult{
ID:          file["id"].(uint),
Type:        "file",
Title:       file["file_name"].(string),
Description: file["file_type"].(string),
URL:         file["url"].(string),
Score:       1.0,
Data:        file,
})
}
total = int64(len(files))

default: // "all"
// Tüm kategorilerde ara
users, _ := s.searchRepo.SearchUsers(req.Query, req.Limit/2)
for _, user := range users {
results = append(results, model.SearchResult{
ID:          user["id"].(uint),
Type:        "user",
Title:       user["name"].(string),
Description: user["email"].(string),
URL:         "/users/" + user["id"].(uint),
Score:       1.0,
Data:        user,
})
}

files, _ := s.searchRepo.SearchFiles(req.Query, req.Limit/2)
for _, file := range files {
results = append(results, model.SearchResult{
ID:          file["id"].(uint),
Type:        "file",
Title:       file["file_name"].(string),
Description: file["file_type"].(string),
URL:         file["url"].(string),
Score:       1.0,
Data:        file,
})
}
total = int64(len(results))
}

// Arama geçmişini kaydet
history := &model.SearchHistory{
UserID:      userID,
Query:       req.Query,
Category:    req.Category,
ResultCount: int(total),
}
s.searchRepo.SaveSearchHistory(history)

// Sayfalama
totalPages := int(math.Ceil(float64(total) / float64(req.Limit)))
page := req.Offset/req.Limit + 1

return &model.SearchResponse{
Results:    results,
Total:      total,
Page:       page,
Limit:      req.Limit,
TotalPages: totalPages,
}, nil
}

// GetSearchHistory kullanıcının arama geçmişini getirir
func (s *SearchService) GetSearchHistory(userID uint, limit int) ([]model.SearchHistory, error) {
if limit <= 0 {
limit = 20
}
return s.searchRepo.GetSearchHistory(userID, limit)
}

// GetPopularSearches popüler aramaları getirir
func (s *SearchService) GetPopularSearches(limit int) ([]model.SearchSuggestion, error) {
if limit <= 0 {
limit = 10
}
return s.searchRepo.GetPopularSearches(limit)
}
