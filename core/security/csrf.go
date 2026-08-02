package security

import (
    "crypto/rand"
    "encoding/base64"
    "net/http"

    "github.com/labstack/echo/v4"
)

// CSRF CSRF korumasÄ±
type CSRF struct {
    tokenLength int
    cookieName  string
    headerName  string
}

// NewCSRF yeni CSRF oluÅŸturur
func NewCSRF() *CSRF {
    return &CSRF{
        tokenLength: 32,
        cookieName:  "csrf_token",
        headerName:  "X-CSRF-Token",
    }
}

// Middleware CSRF middleware'i
func (c *CSRF) Middleware() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(ctx echo.Context) error {
            // GET, HEAD, OPTIONS, TRACE metodlarÄ± iÃ§in kontrol etme
            method := ctx.Request().Method
            if method == "GET" || method == "HEAD" || method == "OPTIONS" || method == "TRACE" {
                return next(ctx)
            }

            // Cookie'den token al
            cookie, err := ctx.Cookie(c.cookieName)
            if err != nil {
                return ctx.JSON(http.StatusForbidden, map[string]string{
                    "error": "CSRF token not found",
                })
            }

            // Header'dan token al
            token := ctx.Request().Header.Get(c.headerName)
            if token == "" {
                return ctx.JSON(http.StatusForbidden, map[string]string{
                    "error": "CSRF token missing in header",
                })
            }

            // Token'larÄ± karÅŸÄ±laÅŸtÄ±r
            if cookie.Value != token {
                return ctx.JSON(http.StatusForbidden, map[string]string{
                    "error": "CSRF token mismatch",
                })
            }

            return next(ctx)
        }
    }
}

// GenerateToken yeni CSRF token oluÅŸturur
func (c *CSRF) GenerateToken() (string, error) {
    bytes := make([]byte, c.tokenLength)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(bytes), nil
}

// SetCookie CSRF token'Ä± cookie olarak ayarlar
func (c *CSRF) SetCookie(ctx echo.Context, token string) {
    cookie := &http.Cookie{
        Name:     c.cookieName,
        Value:    token,
        Path:     "/",
        HttpOnly: true,
        Secure:   false,
        SameSite: http.SameSiteStrictMode,
        MaxAge:   3600,
    }
    ctx.SetCookie(cookie)
}
