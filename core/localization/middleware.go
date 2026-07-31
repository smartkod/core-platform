package localization

import (
    "strings"

    "github.com/labstack/echo/v4"
)

// Middleware dil middleware'i
type Middleware struct {
    translator *Translator
    defaultLocale string
    supportedLocales []string
}

// NewMiddleware yeni middleware oluşturur
func NewMiddleware(translator *Translator, defaultLocale string, supportedLocales []string) *Middleware {
    return &Middleware{
        translator: translator,
        defaultLocale: defaultLocale,
        supportedLocales: supportedLocales,
    }
}

// Middleware dil middleware'ini döndürür
func (m *Middleware) Middleware() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // Dil tespiti:
            // 1. Query parametresi: ?lang=tr
            // 2. Cookie: lang=tr
            // 3. Header: Accept-Language: tr-TR,tr;q=0.9
            // 4. Varsayılan dil

            locale := m.getLocale(c)

            // Context'e locale'i ekle
            c.Set("locale", locale)

            // Translator'ı context'e ekle
            c.Set("translator", m.translator)

            return next(c)
        }
    }
}

// getLocale locale tespiti yapar
func (m *Middleware) getLocale(c echo.Context) string {
    // 1. Query parametresinden kontrol et
    if lang := c.QueryParam("lang"); lang != "" {
        if m.isSupported(lang) {
            return lang
        }
    }

    // 2. Cookie'den kontrol et
    if cookie, err := c.Cookie("lang"); err == nil {
        if m.isSupported(cookie.Value) {
            return cookie.Value
        }
    }

    // 3. Header'dan kontrol et
    if acceptLang := c.Request().Header.Get("Accept-Language"); acceptLang != "" {
        // tr-TR,tr;q=0.9,en;q=0.8 -> tr
        parts := strings.Split(acceptLang, ",")
        for _, part := range parts {
            lang := strings.Split(part, ";")[0]
            lang = strings.TrimSpace(lang)
            // tr-TR -> tr
            lang = strings.Split(lang, "-")[0]
            if m.isSupported(lang) {
                return lang
            }
        }
    }

    // 4. Varsayılan dil
    return m.defaultLocale
}

// isSupported locale destekleniyor mu kontrol eder
func (m *Middleware) isSupported(locale string) bool {
    for _, supported := range m.supportedLocales {
        if supported == locale {
            return true
        }
    }
    return false
}

// GetLocale context'ten locale alır
func GetLocale(c echo.Context) string {
    if locale, ok := c.Get("locale").(string); ok {
        return locale
    }
    return "en"
}

// GetTranslator context'ten translator alır
func GetTranslator(c echo.Context) *Translator {
    if translator, ok := c.Get("translator").(*Translator); ok {
        return translator
    }
    return nil
}

// T context'ten çeviri yapar
func T(c echo.Context, key string, data interface{}) string {
    translator := GetTranslator(c)
    if translator == nil {
        return key
    }
    locale := GetLocale(c)
    return translator.Translate(locale, key, data)
}
