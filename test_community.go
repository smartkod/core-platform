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

e.GET("/health", func(c echo.Context) error {
return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
})

communityGroup := e.Group("/api/community")
communityGroup.GET("/posts", func(c echo.Context) error {
return c.JSON(http.StatusOK, []map[string]interface{}{
{"id": 1, "title": "Test Post", "category": "experience"},
})
})
communityGroup.GET("/patterns", func(c echo.Context) error {
return c.JSON(http.StatusOK, []map[string]interface{}{
{"category": "residence", "insight": "Test insight"},
})
})

e.Logger.Fatal(e.Start(":8081"))
}
