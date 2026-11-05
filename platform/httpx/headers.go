package httpx

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// SecureHeaderConfig is the response-header policy.
type SecureHeaderConfig struct {
	// Production turns on Strict-Transport-Security. It must stay off outside
	// production: HSTS over plain HTTP would pin a scheme the server cannot
	// serve, and a browser would then refuse to reach the API until the header
	// expires.
	Production bool
	// ContentSecurityPolicy overrides the default policy. The default
	// (`default-src 'none'`) is correct for a JSON API: nothing on the response
	// is meant to be loaded or executed.
	ContentSecurityPolicy string
}

// SecureHeaders sets the baseline hardening headers.
//
// Every header here answers a specific failure, not a checklist:
//
//   - Content-Security-Policy stops a response from being repurposed as a
//     document that loads or executes anything. An API returns data, so the
//     only correct policy is "nothing is allowed".
//   - X-Content-Type-Options stops a browser from guessing a type and running
//     a JSON body as a script.
//   - X-Frame-Options and frame-ancestors stop the API from being embedded, so
//     a click-jacked frame cannot be used to reach an authenticated endpoint.
//   - Referrer-Policy stops URLs — which can contain an id or a token in a
//     query string — from leaking to third parties.
//   - Strict-Transport-Security, in production only, pins HTTPS so a single
//     plaintext request cannot be downgraded.
//
// X-XSS-Protection is deliberately absent: modern browsers ignore it, and the
// legacy auditor it enabled could itself be abused. The protection that matters
// is CSP plus the client rendering untrusted values as text.
func SecureHeaders(cfg SecureHeaderConfig) echo.MiddlewareFunc {
	csp := cfg.ContentSecurityPolicy
	if csp == "" {
		csp = "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'"
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Response().Header()
			h.Set("Content-Security-Policy", csp)
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "no-referrer")
			h.Set("Cross-Origin-Opener-Policy", "same-origin")
			if cfg.Production {
				// Two years, subdomains included: long enough to survive a
				// forgetful deployment, short enough to be revocable.
				h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			}
			return next(c)
		}
	}
}

// NoStore tells every cache that the response is specific to one signed-in
// caller.
//
// Authentication responses are the reason: a proxy that caches a login reply
// would hand one user's tokens to the next requester.
func NoStore() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().Method != http.MethodGet {
				c.Response().Header().Set("Cache-Control", "no-store")
			}
			return next(c)
		}
	}
}
