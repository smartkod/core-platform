package middleware

import (
"encoding/json"
"html"
"net/http"
"reflect"
"strings"

"github.com/labstack/echo/v4"
)

// XSSProtection XSS saldÃ„Â±rÃ„Â±larÃ„Â±na karÃ…Å¸Ã„Â± koruma saÃ„Å¸lar
func XSSProtection() echo.MiddlewareFunc {
return func(next echo.HandlerFunc) echo.HandlerFunc {
return func(c echo.Context) error {
// GET parametrelerini temizle
for key, values := range c.QueryParams() {
for i, v := range values {
values[i] = html.EscapeString(v)
}
c.QueryParams()[key] = values
}

// Form verilerini temizle
if c.Request().Method == "POST" || c.Request().Method == "PUT" {
if err := c.Request().ParseForm(); err == nil {
for key, values := range c.Request().Form {
for i, v := range values {
values[i] = html.EscapeString(v)
}
c.Request().Form[key] = values
}
}
}

// JSON body'yi temizle
if strings.HasPrefix(c.Request().Header.Get("Content-Type"), "application/json") {
body := c.Request().Body
var data interface{}
if err := json.NewDecoder(body).Decode(&data); err == nil {
sanitized := sanitizeJSON(data)
// Body'yi tekrar oluÃ…Å¸tur
newBody, _ := json.Marshal(sanitized)
c.Request().Body = io.NopCloser(strings.NewReader(string(newBody)))
}
}

return next(c)
}
}
}

// sanitizeJSON JSON verisini temizler
func sanitizeJSON(data interface{}) interface{} {
switch v := data.(type) {
case string:
return html.EscapeString(v)
case map[string]interface{}:
result := make(map[string]interface{})
for key, value := range v {
result[key] = sanitizeJSON(value)
}
return result
case []interface{}:
result := make([]interface{}, len(v))
for i, item := range v {
result[i] = sanitizeJSON(item)
}
return result
default:
return v
}
}