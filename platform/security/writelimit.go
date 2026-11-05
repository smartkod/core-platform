package security

import (
	"github.com/labstack/echo/v4"

	"core-platform/contract/core"
)

// PerUser caps how many resources one account may create per window.
//
// The key is the user id, never the address: creating a branch or an offering is
// something an account does, so rotating IPs must not reset the budget and a
// shared office network must not exhaust it for everyone. It is deliberately
// applied to creation routes only — editing an existing object is already
// bounded by the object itself, and throttling it would punish normal
// housekeeping without stopping anyone.
//
// The check runs after the identity middleware, so the principal is always
// present. A missing principal is a wiring bug, not a caller error, and is
// reported as unauthorized rather than silently allowed: a limiter that fails
// open would be worse than none, because it would look like protection.
func PerUser(limiter *RateLimiter, scope string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			p := core.UserFrom(c)
			if p == nil {
				return core.ErrUnauthorized
			}
			if !limiter.Allow("write:" + scope + ":" + p.UserID) {
				return core.ErrRateLimited
			}
			return next(c)
		}
	}
}
