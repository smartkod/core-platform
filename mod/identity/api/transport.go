package api

import (
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
//
// The CSRF token is signed with the user's own session, and every rotation of
// the access token rotates it: the pair is written together, so they can never
// drift apart.
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
	// it into the X-CSRF-Token header. The value is signed with the access
	// token, so a token planted by another host is useless here.
	c.SetCookie(&http.Cookie{
		Name:     CSRFCookieName,
		Value:    httpx.IssueCSRFToken(h.csrfSecret(), s.Access.Token),
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

// csrfSecret is the key that signs CSRF tokens. It reuses the refresh secret
// rather than introducing another one: the purpose is domain separation from
// anything a client can see, not a distinct key hierarchy.
func (h *Handler) csrfSecret() []byte {
	return []byte(h.cfg.RefreshSecret)
}

// timeUntil returns whole seconds from now until t, never negative.
func timeUntil(t time.Time) int64 {
	d := time.Until(t)
	if d <= 0 {
		return 0
	}
	return int64(d.Seconds())
}
