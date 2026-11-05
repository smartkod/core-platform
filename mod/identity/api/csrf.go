package api

import (
	"github.com/labstack/echo/v4"

	"core-platform/platform/httpx"
)

// CSRFProtect is the identity module's configured view of the kernel check.
//
// The middleware itself lives in platform/httpx because it has to run for every
// cookie-authenticated write, not only the identity routes: a cross-site form
// post against an organization endpoint is exactly as dangerous as one against
// the login endpoint.
func CSRFProtect(secret []byte) echo.MiddlewareFunc {
	return httpx.CSRFProtect(httpx.CSRFConfig{
		Secret:       secret,
		AccessCookie: AccessCookieName,
		CSRFCookie:   CSRFCookieName,
		HeaderName:   CSRFHeaderName,
	})
}

// IssueCSRFToken is the identity module's signing helper. Tests use it to mint
// a token for a known session; production only calls it from setCookies.
func IssueCSRFToken(secret []byte, session string) string {
	return httpx.IssueCSRFToken(secret, session)
}
