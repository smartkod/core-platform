package api

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/labstack/echo/v4"

	"core-platform/contract/core"
	"core-platform/mod/identity/model"
	"core-platform/mod/identity/repository"
	"core-platform/mod/identity/service"
)

// Identity implements core.Identity.
type Identity struct {
	tokens *service.Tokens
	users  *repository.Users
}

// NewIdentity builds the authentication middleware provider.
func NewIdentity(tokens *service.Tokens, users *repository.Users) *Identity {
	return &Identity{tokens: tokens, users: users}
}

// Middleware authenticates a request.
//
// Order matters: the Authorization header is checked first and, if present,
// cookies are ignored entirely. Accepting a valid Bearer token alongside a
// stale cookie would let one request be authenticated as two different users
// depending on which check ran first (PRD.md §6.1).
func (i *Identity) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			raw := accessToken(c)
			if raw == "" {
				return core.ErrUnauthorized
			}

			claims, err := i.tokens.ParseAccess(raw)
			if err != nil {
				return err
			}

			u, err := i.users.ByID(c.Request().Context(), claims.Subject)
			if err != nil {
				return core.ErrUnauthorized
			}
			// A token minted before a password change carries an old version and
			// must stop working even though its signature is still valid.
			if claims.Ver != u.TokenVersion {
				return core.ErrUnauthorized
			}
			if u.Status != model.StatusActive {
				return core.ErrForbidden
			}

			c.Set(core.ContextUserKey, &core.Principal{
				UserID:    u.ID,
				Email:     u.Email,
				Role:      u.Role,
				Scopes:    claims.Scope,
				TokenID:   claims.ID,
				SessionID: claims.SID,
			})
			return next(c)
		}
	}
}

// OptionalMiddleware attaches the principal when credentials are present and
// lets anonymous requests through.
//
// A malformed or expired token is not treated as anonymous: silently dropping a
// bad credential would make a broken client look like a signed-out user, and
// would hide a revoked token from the caller.
func (i *Identity) OptionalMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			raw := accessToken(c)
			if raw == "" {
				return next(c)
			}

			claims, err := i.tokens.ParseAccess(raw)
			if err != nil {
				return core.ErrUnauthorized
			}
			u, err := i.users.ByID(c.Request().Context(), claims.Subject)
			if err != nil {
				return core.ErrUnauthorized
			}
			if claims.Ver != u.TokenVersion {
				return core.ErrUnauthorized
			}
			if u.Status != model.StatusActive {
				return core.ErrForbidden
			}

			c.Set(core.ContextUserKey, &core.Principal{
				UserID:    u.ID,
				Email:     u.Email,
				Role:      u.Role,
				Scopes:    claims.Scope,
				TokenID:   claims.ID,
				SessionID: claims.SID,
			})
			return next(c)
		}
	}
}

// RequireRole allows only the listed roles.
func (i *Identity) RequireRole(allowed ...string) echo.MiddlewareFunc {
	permitted := make(map[string]bool, len(allowed))
	for _, r := range allowed {
		permitted[r] = true
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			p := core.UserFrom(c)
			if p == nil {
				return core.ErrUnauthorized
			}
			if !permitted[p.Role] {
				return core.ErrForbidden
			}
			return next(c)
		}
	}
}

// CurrentUser returns the authenticated principal.
func (i *Identity) CurrentUser(c echo.Context) (*core.Principal, error) {
	p := core.UserFrom(c)
	if p == nil {
		return nil, core.ErrUnauthorized
	}
	return p, nil
}

// accessToken extracts the raw access token, preferring the Authorization
// header over the cookie.
func accessToken(c echo.Context) string {
	if h := c.Request().Header.Get("Authorization"); h != "" {
		parts := strings.SplitN(h, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
		// A malformed Authorization header is an error, not a reason to fall
		// back to cookies: falling back would hide client bugs.
		return ""
	}
	if ck, err := c.Cookie(AccessCookieName); err == nil {
		return ck.Value
	}
	return ""
}

// RateLimiters bundles the identity-specific limits.
type RateLimiters struct {
	Login   echo.MiddlewareFunc
	Refresh echo.MiddlewareFunc
	// Signup guards account creation, password reset requests and the reset
	// itself. Unlike Login it never refunds: those endpoints create state
	// (an account, an e-mail) on success, so a successful call is exactly the
	// thing an abuser wants to repeat. Refunding them would make the limit
	// apply only to failures, which is no limit at all.
	Signup echo.MiddlewareFunc
}

// Limiter is the rate-limiter contract the identity middleware needs.
//
// Refund is part of it on purpose: the limiter must be able to forgive a
// successful attempt, so only failures count toward the budget.
type Limiter interface {
	Allow(key string) bool
	Refund(key string)
}

// NewRateLimiters builds the middleware from the shared limiter.
//
// Login is keyed by IP *and* e-mail: that stops a single address from spraying
// many accounts and stops a single account from being hammered from many
// addresses, without letting one attacker lock out every user at once.
//
// A successful login refunds its own budget. Brute force still stops after the
// limit, but a legitimate user (or everyone behind one NAT) never gets locked
// out by their own successful logins.
func NewRateLimiters(login, refresh Limiter) RateLimiters {
	// signup consumes the same window but never refunds.
	signup := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ipKey := "signup:ip:" + c.RealIP()
			email := peekEmail(c)
			emailKey := ""
			if email != "" {
				emailKey = "signup:email:" + email
			}

			if !login.Allow(ipKey) {
				return core.ErrRateLimited
			}
			if emailKey != "" && !login.Allow(emailKey) {
				return core.ErrRateLimited
			}
			return next(c)
		}
	}

	return RateLimiters{
		Signup: signup,
		Login: func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				ipKey := "login:ip:" + c.RealIP()
				email := peekEmail(c)
				emailKey := ""
				if email != "" {
					emailKey = "login:email:" + email
				}

				if !login.Allow(ipKey) {
					return core.ErrRateLimited
				}
				if emailKey != "" && !login.Allow(emailKey) {
					login.Refund(ipKey)
					return core.ErrRateLimited
				}

				err := next(c)

				// Only failures consume the budget.
				if err == nil {
					login.Refund(ipKey)
					if emailKey != "" {
						login.Refund(emailKey)
					}
				}
				return err
			}
		},
		Refresh: func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				key := "refresh:ip:" + c.RealIP()
				if !refresh.Allow(key) {
					return core.ErrRateLimited
				}
				err := next(c)
				if err == nil {
					refresh.Refund(key)
				}
				return err
			}
		},
	}
}

// peekEmail reads the e-mail from the body without consuming it.
//
// The body is read once, then replaced with an equivalent reader, because the
// handler downstream still needs to bind the same payload.
func peekEmail(c echo.Context) string {
	if c.Request().Body == nil {
		return ""
	}
	body, err := io.ReadAll(io.LimitReader(c.Request().Body, 1<<20))
	if err != nil {
		return ""
	}
	c.Request().Body = io.NopCloser(bytes.NewReader(body))

	var probe struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &probe); err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(probe.Email))
}

// AuthorizationHeader is the exact header name used for Bearer tokens.
const AuthorizationHeader = "Authorization"
