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
"version": "0.11.0",
})
})

// Root
e.GET("/", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]string{
"message": "Core Platform API",
"status":  "running",
})
})

// --- AUTH ROUTES ---
authGroup := e.Group("/api/auth")
authGroup.POST("/register", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]string{
"message": "Register endpoint working!",
"email":   "test@test.com",
})
})
authGroup.POST("/login", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]string{
"message": "Login endpoint working!",
"token":   "fake-jwt-token",
})
})

// --- USERS ROUTES ---
userGroup := e.Group("/api/users")
userGroup.GET("/me", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]interface{}{
"id":      1,
"email":   "test@test.com",
"name":    "Test User",
"message": "User profile",
})
})

// --- WALLET ROUTES ---
walletGroup := e.Group("/api/wallet")
walletGroup.GET("", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]interface{}{
"id":         1,
"user_id":    1,
"balance":    1000.50,
"currency":   "TRY",
"is_active":  true,
})
})
walletGroup.GET("/balance", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]interface{}{
"balance":  1000.50,
"currency": "TRY",
})
})
walletGroup.POST("/deposit", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]string{
"message": "Deposit successful",
"amount":  "100.00",
})
})
walletGroup.POST("/withdraw", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]string{
"message": "Withdraw successful",
"amount":  "50.00",
})
})
walletGroup.POST("/transfer", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]string{
"message": "Transfer successful",
})
})
walletGroup.GET("/transactions", func(c echo.Context) error {
return c.JSON(http.StatusOK, []map[string]interface{}{
{"id": 1, "type": "deposit", "amount": 100.00, "status": "completed"},
{"id": 2, "type": "withdraw", "amount": -50.00, "status": "completed"},
})
})

// --- REWARDED ADS ROUTES ---
adGroup := e.Group("/api/ads")
adGroup.GET("/status", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]interface{}{
"can_watch":         true,
"watched_today":     3,
"remaining_today":   2,
"token_per_ad":      10,
"daily_limit":       5,
"daily_token_limit": 50,
})
})
adGroup.POST("/watch", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]interface{}{
"success":      true,
"message":      "Ad watched successfully! You earned 10 tokens.",
"reward_value": 10,
"total_today":  4,
"remaining":    1,
})
})

// --- PAYMENT ROUTES ---
paymentGroup := e.Group("/api/payments")
paymentGroup.GET("/methods", func(c echo.Context) error {
return c.JSON(http.StatusOK, []map[string]string{
{"id": "paynow", "name": "PayNow", "icon": "ÃƒÆ’Ã¢â‚¬ÂÃƒâ€¦Ã‚Â¸ÃƒÆ’Ã¢â‚¬Â¦Ãƒâ€šÃ‚Â¸ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€šÃ‚Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¸ÃƒÆ’Ã¢â‚¬ÂÃƒâ€¦Ã‚Â¸ÃƒÆ’Ã¢â‚¬Â¦Ãƒâ€šÃ‚Â¸ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€šÃ‚Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¬"},
{"id": "grabpay", "name": "GrabPay", "icon": "ÃƒÆ’Ã¢â‚¬ÂÃƒâ€¦Ã‚Â¸ÃƒÆ’Ã¢â‚¬Â¦Ãƒâ€šÃ‚Â¸ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€šÃ‚Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¸ÃƒÆ’Ã¢â‚¬ÂÃƒâ€¦Ã‚Â¸ÃƒÆ’Ã¢â‚¬Â¦Ãƒâ€šÃ‚Â¸ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€šÃ‚Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¬"},
{"id": "shopeepay", "name": "ShopeePay", "icon": "ÃƒÆ’Ã¢â‚¬ÂÃƒâ€¦Ã‚Â¸ÃƒÆ’Ã¢â‚¬Â¦Ãƒâ€šÃ‚Â¸ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€šÃ‚Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¸ÃƒÆ’Ã¢â‚¬ÂÃƒâ€¦Ã‚Â¸ÃƒÆ’Ã¢â‚¬Â¦Ãƒâ€šÃ‚Â¸ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€šÃ‚Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¬"},
{"id": "card", "name": "Credit/Debit Card", "icon": "ÃƒÆ’Ã¢â‚¬ÂÃƒâ€¦Ã‚Â¸ÃƒÆ’Ã¢â‚¬Â¦Ãƒâ€šÃ‚Â¸ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬ÂÃ‚Â¢ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â³"},
{"id": "fpx", "name": "FPX", "icon": "ÃƒÆ’Ã¢â‚¬ÂÃƒâ€¦Ã‚Â¸ÃƒÆ’Ã¢â‚¬Â¦Ãƒâ€šÃ‚Â¸ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€šÃ‚Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â²ÃƒÆ’Ã¢â‚¬ÂÃƒâ€¦Ã‚Â¸ÃƒÆ’Ã¢â‚¬Â¦Ãƒâ€šÃ‚Â¸ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€šÃ‚Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¾"},
})
})
paymentGroup.POST("/create", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]interface{}{
"payment_id":   "pay_12345",
"status":       "pending",
"amount":       100.00,
"currency":     "SGD",
"redirect_url": "https://payment.example.com/pay_12345",
})
})
paymentGroup.GET("/:id", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]interface{}{
"payment_id": c.Param("id"),
"status":     "completed",
"amount":     100.00,
"currency":   "SGD",
})
})

// --- PREMIUM ROUTES ---
premiumGroup := e.Group("/api/premium")
premiumGroup.GET("/plans", func(c echo.Context) error {
return c.JSON(http.StatusOK, []map[string]interface{}{
{"id": 1, "name": "Monthly", "price": 49.90, "currency": "TRY", "duration": 30},
{"id": 2, "name": "Yearly", "price": 499.90, "currency": "TRY", "duration": 365},
{"id": 3, "name": "Lifetime", "price": 999.90, "currency": "TRY", "duration": 0},
})
})
premiumGroup.GET("/me", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]interface{}{
"is_active":      true,
"plan_name":      "Yearly",
"days_remaining": 265,
})
})
premiumGroup.POST("/purchase", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]string{
"message": "Premium purchased successfully!",
})
})
premiumGroup.POST("/cancel", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]string{
"message": "Premium subscription canceled",
})
})

// --- NOTIFICATIONS ROUTES ---
notifGroup := e.Group("/api/notifications")
notifGroup.GET("", func(c echo.Context) error {
return c.JSON(http.StatusOK, []map[string]interface{}{
{
"id":         1,
"type":       "inapp",
"title":      "HoÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â¦ÃƒÆ’Ã¢â‚¬Â¦Ãƒâ€šÃ‚Â¸ geldiniz!",
"content":    "Core Platform'a hoÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â¦ÃƒÆ’Ã¢â‚¬Â¦Ãƒâ€šÃ‚Â¸ geldiniz!",
"is_read":    false,
"is_sent":    true,
"created_at": "2026-07-31T10:00:00Z",
},
{
"id":         2,
"type":       "email",
"title":      "Premium ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¼yeliÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚ÂÃƒÆ’Ã¢â‚¬Â¦Ãƒâ€šÃ‚Â¸iniz aktif!",
"content":    "Premium ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¼yeliÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚ÂÃƒÆ’Ã¢â‚¬Â¦Ãƒâ€šÃ‚Â¸iniz baÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â¦ÃƒÆ’Ã¢â‚¬Â¦Ãƒâ€šÃ‚Â¸arÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚ÂÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â±yla aktifleÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â¦ÃƒÆ’Ã¢â‚¬Â¦Ãƒâ€šÃ‚Â¸tirildi.",
"is_read":    true,
"is_sent":    true,
"created_at": "2026-07-31T09:00:00Z",
},
})
})
notifGroup.GET("/unread", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]interface{}{
"unread_count": 5,
})
})
notifGroup.PUT("/:id/read", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]string{
"message": "Notification " + c.Param("id") + " marked as read",
})
})
notifGroup.PUT("/read-all", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]string{
"message": "All notifications marked as read",
})
})
notifGroup.GET("/preferences", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]interface{}{
"email":       true,
"push":        true,
"in_app":      true,
"sms":         false,
"marketing":   true,
"transaction": true,
"system":      true,
})
})
notifGroup.PUT("/preferences", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]string{
"message": "Preferences updated successfully",
})
})

// --- AI GATEWAY ROUTES ---
aiGroup := e.Group("/api/ai")
aiGroup.POST("/chat", func(c echo.Context) error {
var req map[string]interface{}
if err := c.Bind(&req); err != nil {
return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
}
return c.JSON(http.StatusOK, map[string]interface{}{
"id":       "ai_resp_123",
"provider": "openai",
"model":    "gpt-3.5-turbo",
"content":  "Bu bir test yanÃ„Â±tÃ„Â±dÃ„Â±r. GerÃƒÂ§ek AI entegrasyonu iÃƒÂ§in OpenAI API anahtarÃ„Â± gereklidir.",
"usage": map[string]interface{}{
"prompt_tokens":     10,
"completion_tokens": 15,
"total_tokens":      25,
},
"created_at": "2026-07-31T10:00:00Z",
})
})
aiGroup.GET("/providers", func(c echo.Context) error {
return c.JSON(http.StatusOK, []map[string]interface{}{
{"id": 1, "name": "openai", "is_active": true, "priority": 1},
{"id": 2, "name": "anthropic", "is_active": false, "priority": 2},
{"id": 3, "name": "gemini", "is_active": false, "priority": 3},
})
})

// --- SEARCH ROUTES ---
searchGroup := e.Group("/api/search")
searchGroup.GET("", func(c echo.Context) error {
query := c.QueryParam("q")
return c.JSON(http.StatusOK, map[string]interface{}{
"results": []map[string]interface{}{
{
"id": "1",
"type": "user",
"title": "Ahmet YÃƒâ€Ã‚Â±lmaz",
"description": "ahmet@example.com",
"url": "/users/1",
"score": 0.95,
},
{
"id": "2",
"type": "file",
"title": "proje_raporu.pdf",
"description": "document",
"url": "/files/2",
"score": 0.85,
},
},
"total": 2,
"page": 1,
"limit": 20,
"total_pages": 1,
})
})
searchGroup.GET("/history", func(c echo.Context) error {
return c.JSON(http.StatusOK, []map[string]interface{}{
{
"id": 1,
"query": "astroloji",
"category": "all",
"result_count": 15,
"created_at": "2026-07-31T10:00:00Z",
},
{
"id": 2,
"query": "burÃƒÆ’Ã‚Â§ yorumu",
"category": "content",
"result_count": 8,
"created_at": "2026-07-31T09:00:00Z",
},
})
})
searchGroup.GET("/popular", func(c echo.Context) error {
return c.JSON(http.StatusOK, []map[string]interface{}{
{"query": "astroloji", "count": 120},
{"query": "burÃƒÆ’Ã‚Â§ yorumu", "count": 95},
{"query": "aÃƒÆ’Ã¢â‚¬Â¦Ãƒâ€¦Ã‚Â¸k", "count": 80},
{"query": "kariyer", "count": 65},
{"query": "saÃƒÆ’Ã¢â‚¬ÂÃƒâ€¦Ã‚Â¸lÃƒÆ’Ã¢â‚¬ÂÃƒâ€šÃ‚Â±k", "count": 50},
})
})

// --- FILE STORAGE ROUTES ---
fileGroup := e.Group("/api/files")
fileGroup.GET("", func(c echo.Context) error {
return c.JSON(http.StatusOK, []map[string]interface{}{
{
"id": 1,
"file_name": "avatar.jpg",
"file_size": 102400,
"file_type": "image",
"mime_type": "image/jpeg",
"url": "http://localhost:8081/uploads/2026/07/31/avatar.jpg",
"is_public": true,
"created_at": "2026-07-31T10:00:00Z",
},
{
"id": 2,
"file_name": "document.pdf",
"file_size": 512000,
"file_type": "document",
"mime_type": "application/pdf",
"url": "http://localhost:8081/uploads/2026/07/31/document.pdf",
"is_public": false,
"created_at": "2026-07-31T09:00:00Z",
},
})
})
fileGroup.GET("/:id", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]interface{}{
"id": c.Param("id"),
"file_name": "avatar.jpg",
"file_size": 102400,
"file_type": "image",
"url": "http://localhost:8081/uploads/2026/07/31/avatar.jpg",
})
})
fileGroup.POST("/upload", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]interface{}{
"id": 3,
"file_name": "uploaded_file.jpg",
"file_size": 204800,
"file_type": "image",
"mime_type": "image/jpeg",
"url": "http://localhost:8081/uploads/2026/07/31/uploaded_file.jpg",
"is_public": true,
"created_at": "2026-07-31T11:00:00Z",
})
})
fileGroup.DELETE("/:id", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]string{
"message": "File deleted successfully",
})
})

// --- ANALYTICS ROUTES ---
analyticsGroup := e.Group("/api/analytics")
analyticsGroup.GET("/daily", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]interface{}{
"date":          "2026-07-31",
"active_users":  150,
"new_users":     12,
"total_views":   1250,
"premium_users": 45,
})
})
analyticsGroup.GET("/users", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]interface{}{
"total_users":      1250,
"active_today":     150,
"new_this_week":    85,
"premium_percent":  3.6,
})
})
analyticsGroup.GET("/revenue", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]interface{}{
"today":      1250.50,
"this_week":  8750.00,
"this_month": 32500.00,
"currency":   "TRY",
})
})

// Start server
e.Logger.Fatal(e.Start(":8081"))
}
