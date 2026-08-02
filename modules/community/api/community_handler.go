package api

import (
"net/http"

"modules/community/model"

"github.com/labstack/echo/v4"
)

type CommunityHandler struct{}

func NewCommunityHandler() *CommunityHandler {
return &CommunityHandler{}
}

// CreatePost topluluk paylaÅŸÄ±mÄ± oluÅŸturur
func (h *CommunityHandler) CreatePost(c echo.Context) error {
var req model.CommunityPostRequest
if err := c.Bind(&req); err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
}

return c.JSON(http.StatusCreated, map[string]interface{}{
"id":           1,
"title":        req.Title,
"content":      req.Content,
"category":     req.Category,
"tags":         req.Tags,
"is_anonymous": req.IsAnonymous,
"like_count":   0,
"comment_count": 0,
"helpful_count": 0,
"created_at":   "2026-07-31T10:00:00Z",
})
}

// GetPosts topluluk paylaÅŸÄ±mlarÄ±nÄ± getirir
func (h *CommunityHandler) GetPosts(c echo.Context) error {
category := c.QueryParam("category")

posts := []map[string]interface{}{
{
"id":            1,
"title":         "SatÃ¼rn 4. evde taÅŸÄ±nma deneyimim",
"content":       "SatÃ¼rn 4. ev transitinde taÅŸÄ±ndÄ±m ve hayatÄ±m deÄŸiÅŸti...",
"category":      "experience",
"tags":          []string{"saturn", "4th-house", "move"},
"is_anonymous":  true,
"like_count":    25,
"comment_count": 5,
"helpful_count": 12,
"created_at":    "2026-07-31T10:00:00Z",
},
{
"id":            2,
"title":         "JÃ¼piter 10. evde kariyer deÄŸiÅŸikliÄŸi",
"content":       "JÃ¼piter 10. ev transitinde iÅŸ deÄŸiÅŸtirdim ve Ã§ok mutluyum...",
"category":      "experience",
"tags":          []string{"jupiter", "10th-house", "career"},
"is_anonymous":  false,
"like_count":    18,
"comment_count": 3,
"helpful_count": 8,
"created_at":    "2026-07-30T10:00:00Z",
},
}

if category != "" {
filtered := []map[string]interface{}{}
for _, post := range posts {
if post["category"] == category {
filtered = append(filtered, post)
}
}
return c.JSON(http.StatusOK, filtered)
}

return c.JSON(http.StatusOK, posts)
}

// GetPost paylaÅŸÄ±m detayÄ±nÄ± getirir
func (h *CommunityHandler) GetPost(c echo.Context) error {
id := c.Param("id")
return c.JSON(http.StatusOK, map[string]interface{}{
"id":            id,
"title":         "SatÃ¼rn 4. evde taÅŸÄ±nma deneyimim",
"content":       "SatÃ¼rn 4. ev transitinde taÅŸÄ±ndÄ±m ve hayatÄ±m deÄŸiÅŸti...",
"category":      "experience",
"tags":          []string{"saturn", "4th-house", "move"},
"is_anonymous":  true,
"like_count":    25,
"comment_count": 5,
"helpful_count": 12,
"comments": []map[string]interface{}{
{
"id":             1,
"content":        "Ben de benzer bir deneyim yaÅŸadÄ±m!",
"is_anonymous":   true,
"helpful_count":  3,
"created_at":     "2026-07-31T11:00:00Z",
},
},
"created_at":   "2026-07-31T10:00:00Z",
})
}

// AddComment yorum ekler
func (h *CommunityHandler) AddComment(c echo.Context) error {
postID := c.Param("id")
var req map[string]interface{}
if err := c.Bind(&req); err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
}

return c.JSON(http.StatusCreated, map[string]interface{}{
"id":             1,
"post_id":        postID,
"content":        req["content"],
"is_anonymous":   req["is_anonymous"],
"helpful_count":  0,
"created_at":     "2026-07-31T10:00:00Z",
})
}

// LikePost paylaÅŸÄ±mÄ± beÄŸenir
func (h *CommunityHandler) LikePost(c echo.Context) error {
id := c.Param("id")
return c.JSON(http.StatusOK, map[string]string{
"message": "Post " + id + " liked successfully",
})
}

// MarkHelpful yorumu faydalÄ± olarak iÅŸaretler
func (h *CommunityHandler) MarkHelpful(c echo.Context) error {
id := c.Param("id")
return c.JSON(http.StatusOK, map[string]string{
"message": "Comment " + id + " marked as helpful",
})
}

// GetPatterns desen keÅŸfi sonuÃ§larÄ±nÄ± getirir
func (h *CommunityHandler) GetPatterns(c echo.Context) error {
category := c.QueryParam("category")

patterns := []map[string]interface{}{
{
"category":      "residence",
"sub_category":  "house_move",
"total_events":  1250,
"total_users":   980,
"common_houses": []string{"4", "2", "3"},
"common_planets": []string{"Moon", "Saturn", "Jupiter"},
"common_transits": []string{"Saturn 4th house", "Moon IC", "Jupiter 4th house"},
"insight":       "KullanÄ±cÄ±larÄ±n %65'i SatÃ¼rn 4. ev transitinde taÅŸÄ±nma deneyimi yaÅŸadÄ±.",
"confidence":    85,
"statistics": map[string]interface{}{
"average_age": 32,
"top_cities":  []string{"Ä°stanbul", "Ankara", "Ä°zmir"},
},
},
{
"category":      "career",
"sub_category":  "new_job",
"total_events":  850,
"total_users":   720,
"common_houses": []string{"10", "6", "2"},
"common_planets": []string{"Jupiter", "Saturn", "Sun"},
"common_transits": []string{"Jupiter 10th house", "Saturn 6th house"},
"insight":       "KullanÄ±cÄ±larÄ±n %55'i JÃ¼piter 10. ev transitinde iÅŸ deÄŸiÅŸtirdi.",
"confidence":    78,
"statistics": map[string]interface{}{
"average_age": 28,
"top_industries": []string{"Teknoloji", "Finans", "EÄŸitim"},
},
},
}

if category != "" {
filtered := []map[string]interface{}{}
for _, pattern := range patterns {
if pattern["category"] == category {
filtered = append(filtered, pattern)
}
}
return c.JSON(http.StatusOK, filtered)
}

return c.JSON(http.StatusOK, patterns)
}
