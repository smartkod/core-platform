package httpx

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"core-platform/contract/core"
)

// Cookie names are part of the wire contract between the API and a browser
// client, so they live in the kernel next to the header names. The identity
// module imports these rather than redefining them: two spellings of "access"
// would mean the CSRF middleware and the login handler disagree about which
// cookie authenticates a request.
const (
	CookieAccess  = "access"
	CookieRefresh = "refresh"
	CookieCSRF    = "csrf"
)

// CSRFConfig describes how a signed, session-bound double-submit token is
// validated.
type CSRFConfig struct {
	// Secret signs the token. An empty secret fails closed: every
	// cookie-authenticated write is rejected rather than accepted unsigned.
	Secret []byte
	// AccessCookie marks a request as cookie-authenticated.
	AccessCookie string
	// CSRFCookie holds the token the client copies into HeaderName.
	CSRFCookie string
	// HeaderName is the header the client must set.
	HeaderName string
}

// IssueCSRFToken returns a token bound to one session.
//
// The value is `nonce.signature`, where the signature covers both the nonce and
// the session credential. A plain random value (the classic double-submit
// cookie) proves only that the caller could read *a* cookie on this origin; if
// a sibling subdomain can set cookies, it can plant a token of its own. Binding
// the signature to the access credential means an injected token is useless
// against the victim's session.
func IssueCSRFToken(secret []byte, session string) string {
	if len(secret) == 0 {
		return ""
	}
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	nonce := base64.RawURLEncoding.EncodeToString(buf)
	return nonce + "." + csrfMAC(secret, nonce, session)
}

// CSRFProtect enforces the check for cookie-authenticated writes only.
//
// Bearer requests are immune by design: a token that is not a cookie cannot be
// attached by a cross-site form post, and requiring a CSRF header from API
// clients would break them for no security gain (PRD.md §6.1).
//
// Requests with no access cookie also pass: CSRF guards a browser's ambient
// credentials, and a fresh client (login, register, password flows) has none,
// so there is nothing a cross-site form post could hijack.
func CSRFProtect(cfg CSRFConfig) echo.MiddlewareFunc {
	if cfg.AccessCookie == "" {
		cfg.AccessCookie = CookieAccess
	}
	if cfg.CSRFCookie == "" {
		cfg.CSRFCookie = CookieCSRF
	}
	if cfg.HeaderName == "" {
		cfg.HeaderName = CSVHeaderCSRF
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			switch c.Request().Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				return next(c)
			}

			// Bearer mode: a cross-site form cannot set this header.
			if c.Request().Header.Get("Authorization") != "" {
				return next(c)
			}

			// Anonymous mode: no cookie credential, nothing to forge.
			access, err := c.Cookie(cfg.AccessCookie)
			if err != nil || access.Value == "" {
				return next(c)
			}

			ck, err := c.Cookie(cfg.CSRFCookie)
			if err != nil || ck.Value == "" {
				return core.ErrForbidden
			}
			header := c.Request().Header.Get(cfg.HeaderName)
			// The header must equal the cookie (double submit) and the value
			// must carry a valid signature for *this* session.
			if subtle.ConstantTimeCompare([]byte(header), []byte(ck.Value)) != 1 {
				return core.ErrForbidden
			}
			if !ValidCSRFToken(cfg.Secret, access.Value, ck.Value) {
				return core.ErrForbidden
			}
			return next(c)
		}
	}
}

// ValidCSRFToken reports whether token was issued by IssueCSRFToken for the
// given session under this secret.
func ValidCSRFToken(secret []byte, session, token string) bool {
	if len(secret) == 0 || token == "" {
		return false
	}
	nonce, mac, ok := strings.Cut(token, ".")
	if !ok || nonce == "" || mac == "" {
		return false
	}
	want := csrfMAC(secret, nonce, session)
	return subtle.ConstantTimeCompare([]byte(mac), []byte(want)) == 1
}

func csrfMAC(secret []byte, nonce, session string) string {
	m := hmac.New(sha256.New, secret)
	m.Write([]byte(nonce))
	m.Write([]byte{'|'})
	m.Write([]byte(session))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}
