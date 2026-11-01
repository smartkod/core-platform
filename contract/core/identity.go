package core

import (
	"context"
	"time"

	"github.com/labstack/echo/v4"
)

// Identity is the authentication service other modules and applications use.
//
// It lives here, in the only package an application may import, so an
// application can call it without depending on the identity module's
// internals, and a replacement implementation stays possible.
type Identity interface {
	// Middleware authenticates a request. It reads the Authorization header
	// first and only falls back to cookies, so Bearer always wins when both
	// are present (L1.md 3.1).
	Middleware() echo.MiddlewareFunc
	// RequireRole rejects requests whose user role is not in allowed.
	RequireRole(allowed ...string) echo.MiddlewareFunc
	// CurrentUser returns the authenticated principal for this request.
	CurrentUser(c echo.Context) (*Principal, error)
}

// Principal is the authenticated caller.
type Principal struct {
	UserID    string
	Email     string
	Role      string
	Scopes    []string
	TokenID   string
	SessionID string
}

// IsAdmin reports whether the principal holds the admin role.
func (p *Principal) IsAdmin() bool { return p != nil && p.Role == RoleAdmin }

// RoleAdmin is the single privileged role in V1.
const RoleAdmin = "admin"

// ContextUserKey is the echo context key holding the *Principal. A constant
// keeps handlers and middleware from disagreeing on a string literal.
const ContextUserKey = "core.principal"

// UserFrom returns the principal stored by the identity middleware, or nil.
func UserFrom(c echo.Context) *Principal {
	if v, ok := c.Get(ContextUserKey).(*Principal); ok {
		return v
	}
	return nil
}

// UserStore is the read side of the identity data other modules need.
//
// Keeping this narrow means a module can display a user's name without being
// able to mutate accounts.
type UserStore interface {
	GetUser(ctx context.Context, id string) (*User, error)
}

// User is the public projection of an account. It deliberately omits the
// password hash and any token material.
type User struct {
	ID              string
	Email           string
	Name            string
	Locale          string
	AvatarURL       string
	Role            string
	Status          string
	EmailVerifiedAt *time.Time
	CreatedAt       time.Time
}
