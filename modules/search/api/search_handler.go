package api

import (
"net/http"
"strconv"

"modules/search/model"
"modules/search/service"

"github.com/labstack/echo/v4"
)

type SearchHandler struct {
searchService *service.SearchService
}

func NewSearchHandler(searchService *service.SearchService) *SearchHandler {
return &SearchHandler{
searchService: searchService,
}
}

// Search genel arama yapar
func (h *SearchHandler) Search(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1)

query := c.QueryParam("q")
if query == "" {
return c.JSON(http.StatusBadRequest, map[string]string{
"error": "Query parameter 'q' is required",
})
}

req := &model.SearchRequest{
Query:      query,
Category:   c.QueryParam("category"),
SortBy:     c.QueryParam("sort_by"),
SortOrder:  c.QueryParam("sort_order"),
}

if limitStr := c.QueryParam("limit"); limitStr != "" {
if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
req.Limit = val
}
}
if offsetStr := c.QueryParam("offset"); offsetStr != "" {
if val, err := strconv.Atoi(offsetStr); err == nil && val >= 0 {
req.Offset = val
}
}

response, err := h.searchService.Search(userID, req)
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, response)
}

// GetHistory arama geÃ§miÅŸini getirir
func (h *SearchHandler) GetHistory(c echo.Context) error {
// TODO: JWT'den user ID al
userID := uint(1)

limit := 20
if limitStr := c.QueryParam("limit"); limitStr != "" {
if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
limit = val
}
}

history, err := h.searchService.GetSearchHistory(userID, limit)
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, history)
}

// GetPopular popÃ¼ler aramalarÄ± getirir
func (h *SearchHandler) GetPopular(c echo.Context) error {
limit := 10
if limitStr := c.QueryParam("limit"); limitStr != "" {
if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
limit = val
}
}

suggestions, err := h.searchService.GetPopularSearches(limit)
if err != nil {
return c.JSON(http.StatusInternalServerError, map[string]string{
"error": err.Error(),
})
}

return c.JSON(http.StatusOK, suggestions)
}
