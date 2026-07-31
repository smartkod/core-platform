package main

import (
"net/http"

"github.com/labstack/echo/v4"
"github.com/labstack/echo/v4/middleware"
)

func main() {
e := echo.New()
e.Use(middleware.Logger())
e.Use(middleware.Recover())
e.Use(middleware.CORS())

// Health
e.GET("/health", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]string{
"status":  "ok",
"service": "Core Platform",
"version": "0.15.0",
})
})

// Root

	// --- AI COPILOT ROUTES ---
	copilotGroup := e.Group("/api/copilot")
	copilotGroup.POST("/ask", func(c echo.Context) error {
		var req map[string]interface{}
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"id":      "copilot_123",
			"answer":  "Bu dönemde, 4. ev ve IC temaları aktif. Taşınma, ev değişikliği veya ailevi konular öne çıkabilir.",
			"context": map[string]interface{}{
				"user_id":         1,
				"learning_level":  "intermediate",
				"birth_chart":     map[string]string{"sun_sign": "Leo", "moon_sign": "Taurus"},
				"current_transits": map[string]string{"sun": "Leo", "moon": "Scorpio"},
			},
			"sources":    []string{"Ptolemy - Tetrabiblos II.3", "William Lilly - Christian Astrology"},
			"confidence": 85,
			"created_at": "2026-07-31T10:00:00Z",
		})
	})
	copilotGroup.GET("/history", func(c echo.Context) error {
		return c.JSON(http.StatusOK, []map[string]interface{}{
			{"id": 1, "role": "user", "content": "Bugun burcum nedir?", "created_at": "2026-07-31T10:00:00Z"},
			{"id": 2, "role": "assistant", "content": "Bugun burcunuz Aslan.", "created_at": "2026-07-31T10:01:00Z"},
		})
	})
e.GET("/", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]string{
"message": "Core Platform API",
"status":  "running",
})
})

// --- WALLET ROUTES ---
walletGroup := e.Group("/api/wallet")
walletGroup.GET("", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]interface{}{
"balance":      50,
"total_earned": 150,
"total_spent":  100,
})
})
walletGroup.GET("/balance", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]interface{}{
"balance": 50,
})
})
walletGroup.POST("/earn", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]interface{}{
"id":          1,
"type":        "task",
"amount":      10,
"description": "Gorev tamamlandi",
"balance":     60,
"created_at":  "2026-07-31T10:00:00Z",
})
})
walletGroup.POST("/spend", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]interface{}{
"id":          1,
"type":        "ai_reading",
"amount":      -5,
"description": "AI yorum satin alindi",
"balance":     45,
"created_at":  "2026-07-31T10:00:00Z",
})
})
walletGroup.GET("/transactions", func(c echo.Context) error {
return c.JSON(http.StatusOK, []map[string]interface{}{
{
"id":          1,
"type":        "daily_bonus",
"amount":      5,
"description": "Gunluk giris bonusu",
"balance":     50,
"created_at":  "2026-07-31T10:00:00Z",
},
{
"id":          2,
"type":        "spend",
"amount":      -10,
"description": "AI yorum satin alindi",
"balance":     45,
"created_at":  "2026-07-31T09:00:00Z",
},
})
})
walletGroup.POST("/daily-bonus", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]interface{}{
"id":          1,
"type":        "daily_bonus",
"amount":      5,
"description": "Gunluk giris bonusu: 5 jeton",
"balance":     55,
"created_at":  "2026-07-31T10:00:00Z",
})
})

// --- REWARDED ADS ROUTES ---
adGroup := e.Group("/api/ads")
adGroup.GET("/status", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]interface{}{
"can_watch":        true,
"watched_today":    3,
"remaining_today":  2,
"token_per_ad":     10,
"daily_limit":      5,
"daily_token_limit": 50,
})
})
adGroup.POST("/watch", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]interface{}{
"success":      true,
"message":      "Reklam izlendi! 10 jeton kazandiniz.",
"reward_value": 10,
"total_today":  4,
"remaining":    1,
})
})

// --- PREMIUM ROUTES ---
premiumGroup := e.Group("/api/premium")
premiumGroup.GET("/plans", func(c echo.Context) error {
return c.JSON(http.StatusOK, []map[string]interface{}{
{"id": 1, "name": "Aylik", "price": 49.90, "currency": "TRY", "duration": 30},
{"id": 2, "name": "Yillik", "price": 499.90, "currency": "TRY", "duration": 365},
{"id": 3, "name": "Omur Boyu", "price": 999.90, "currency": "TRY", "duration": 0},
})
})
premiumGroup.GET("/me", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]interface{}{
"is_active":      false,
"days_remaining": 0,
})
})
premiumGroup.POST("/purchase", func(c echo.Context) error {
var req map[string]interface{}
if err := c.Bind(&req); err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
}
return c.JSON(http.StatusOK, map[string]interface{}{
"is_active":      true,
"plan_id":        req["plan_id"],
"message":        "Premium satin alindi!",
"days_remaining": 30,
})
})
premiumGroup.POST("/cancel", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]string{
"message": "Premium aboneligi iptal edildi.",
})
})

// --- LIFEWEAVE ROUTES ---
lwGroup := e.Group("/api/lifeweave")
lwGroup.GET("/events", func(c echo.Context) error {
return c.JSON(http.StatusOK, []map[string]interface{}{
{"id": 1, "title": "Yeni Eve Tasindim", "category": "residence"},
{"id": 2, "title": "Yeni Ise Basladim", "category": "career"},
})
})
lwGroup.GET("/timeline", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]interface{}{
"2026": []map[string]interface{}{
{"id": 1, "title": "Yeni Eve Tasindim"},
{"id": 2, "title": "Yeni Ise Basladim"},
},
})
})

// --- COMMUNITY ROUTES ---
communityGroup := e.Group("/api/community")
communityGroup.GET("/posts", func(c echo.Context) error {
return c.JSON(http.StatusOK, []map[string]interface{}{
{
"id":          1,
"title":       "Saturn 4. evde tasinma deneyimim",
"category":    "experience",
"is_anonymous": true,
"like_count":  25,
"created_at":  "2026-07-31T10:00:00Z",
},
})
})
communityGroup.GET("/patterns", func(c echo.Context) error {
return c.JSON(http.StatusOK, []map[string]interface{}{
{
"category":     "residence",
"total_events": 1250,
"insight":      "Kullanicilarin %65'i Saturn 4. ev transitinde tasinma deneyimi yasadi.",
"confidence":   85,
},
})
})

// Start server
e.Logger.Fatal(e.Start(":8081"))
}
