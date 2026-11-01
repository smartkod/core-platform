package api

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"core-platform/contract/core"
	"core-platform/mod/identity/service"
	"core-platform/platform/httpx"
)

// bind decodes a JSON body and turns a decode failure into the validation
// sentinel, so the client sees 422 with the standard envelope instead of a bare
// 400 from the framework.
func bind(c echo.Context, dst any) error {
	if err := c.Bind(dst); err != nil {
		return core.ErrValidation
	}
	return nil
}

// respond writes the platform success envelope.
func respond(c echo.Context, status int, data any) error {
	return httpx.RespondOK(c, status, data)
}

// pageParams reads limit/offset with a hard cap, so a client cannot ask for the
// whole table (L0.md 4.4).
func pageParams(c echo.Context) (limit, offset int) {
	limit = 20
	if v := c.QueryParam("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = min(n, 100)
		}
	}
	if v := c.QueryParam("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return limit, offset
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// audienceFor decides the transport: an explicit header, a browser-looking
// request, or the mobile/default path.
//
// Bearer always wins for authentication, but the audience decides whether the
// refresh token is returned in the body or in a cookie.
func audienceFor(c echo.Context) string {
	if c.Request().Header.Get("X-Client-Type") == "web" || c.QueryParam("client") == "web" {
		return service.AudienceWeb
	}
	// A request carrying an Origin header is almost certainly a browser.
	if c.Request().Header.Get("Origin") != "" {
		return service.AudienceWeb
	}
	return service.AudienceMobile
}

// refreshFromRequest reads the refresh token from the body or the cookie.
//
// Body first: an API client that sends an explicit value must never be
// silently overridden by a stale cookie.
func (h *Handler) refreshFromRequest(c echo.Context) string {
	var in struct {
		RefreshToken string `json:"refresh_token"`
	}
	if c.Request().Body != nil {
		_ = c.Bind(&in) // a missing body is not an error here
	}
	if in.RefreshToken != "" {
		return in.RefreshToken
	}
	if ck, err := c.Cookie(RefreshCookieName); err == nil {
		return ck.Value
	}
	return ""
}

// setCookies writes the HttpOnly pair used by browser clients, plus a
// readable CSRF token for the double-submit check.
func (h *Handler) setCookies(c echo.Context, s *service.Session) {
	c.SetCookie(&http.Cookie{
		Name:     AccessCookieName,
		Value:    s.Access.Token,
		Path:     "/",
		Domain:   "",
		MaxAge:   int(h.cfg.AccessTTL.Seconds()),
		Secure:   h.cfg.CookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	c.SetCookie(&http.Cookie{
		Name:     RefreshCookieName,
		Value:    s.Refresh,
		Path:     h.cfg.CookiePath,
		MaxAge:   int(timeUntil(s.RefreshExpires)),
		Secure:   h.cfg.CookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// The CSRF cookie is deliberately readable by JavaScript: the client copies
	// it into the X-CSRF-Token header, and an attacker on another origin can
	// neither read it nor set the header.
	c.SetCookie(&http.Cookie{
		Name:     CSRFCookieName,
		Value:    csrfToken(),
		Path:     "/",
		MaxAge:   int(timeUntil(s.RefreshExpires)),
		Secure:   h.cfg.CookieSecure,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearCookies(c echo.Context) {
	for _, name := range []string{AccessCookieName, RefreshCookieName, CSRFCookieName} {
		c.SetCookie(&http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			Secure:   h.cfg.CookieSecure,
			HttpOnly: name != CSRFCookieName,
			SameSite: http.SameSiteLaxMode,
		})
	}
}

func csrfToken() string {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

// timeUntil returns whole seconds from now until t, never negative.
func timeUntil(t time.Time) int64 {
	d := time.Until(t)
	if d <= 0 {
		return 0
	}
	return int64(d.Seconds())
}

// CSRFProtect enforces the double-submit check for cookie-authenticated
// requests only.
//
// Bearer requests are immune by design: a token that is not a cookie cannot be
// attached by a cross-site form post, and requiring a CSRF header from API
// clients would break them for no security gain (L1.md 3.1).
//
// Requests with no access cookie also pass: CSRF guards a browser's ambient
// credentials, and a fresh client (login, register, password flows) has none,
// so there is nothing a cross-site form post could hijack.
func CSRFProtect() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			method := c.Request().Method
			if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
				return next(c)
			}

			// Bearer mode: no CSRF requirement.
			if c.Request().Header.Get("Authorization") != "" {
				return next(c)
			}

			// Anonymous mode: no cookie credential, nothing to forge.
			if _, err := c.Cookie(AccessCookieName); err != nil {
				return next(c)
			}

			ck, err := c.Cookie(CSRFCookieName)
			if err != nil || ck.Value == "" {
				return core.ErrForbidden
			}
			header := c.Request().Header.Get(CSRFHeaderName)
			if subtle.ConstantTimeCompare([]byte(header), []byte(ck.Value)) != 1 {
				return core.ErrForbidden
			}
			return next(c)
		}
	}
}
