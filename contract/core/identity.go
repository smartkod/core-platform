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
	// are present (PRD.md §6.1).
	Middleware() echo.MiddlewareFunc
	// OptionalMiddleware attaches the principal when credentials are present
	// and lets anonymous requests through. Public discovery endpoints use it
	// to personalize results without requiring a session.
	OptionalMiddleware() echo.MiddlewareFunc
	// RequireRole rejects requests whose platform role is not in allowed.
	//
	// Platform roles are only for platform-wide duties (administration,
	// moderation). Business-scoped authority is expressed through the
	// Authorizer, never through this method.
	RequireRole(allowed ...string) echo.MiddlewareFunc
	// CurrentUser returns the authenticated principal for this request.
	CurrentUser(c echo.Context) (*Principal, error)
}

// Principal is the authenticated caller.
//
// It deliberately carries no organization: which tenant a request acts for is
// decided per request from the route and the stored resource, and checked by
// the Authorizer.
type Principal struct {
	UserID    string
	Email     string
	Role      string
	Scopes    []string
	TokenID   string
	SessionID string
}

// IsAdmin reports whether the principal holds the platform admin role.
func (p *Principal) IsAdmin() bool { return p != nil && p.Role == RoleAdmin }

// Platform roles. They describe platform-wide duties only.
const (
	RoleUser      = "user"
	RoleAdmin     = "admin"
	RoleModerator = "moderator"
)

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

// Account statuses.
const (
	// AccountActive can authenticate and act.
	AccountActive = "active"
	// AccountSuspended can authenticate nowhere; suspension is reversible.
	AccountSuspended = "suspended"
	// AccountClosed is a user-requested closure awaiting deletion. The record
	// survives so audits and ledgers stay referentially valid.
	AccountClosed = "closed"
	// AccountDeleted is the final soft-delete state.
	AccountDeleted = "deleted"
)

// UserStore is the read side of the identity data other modules need.
//
// Keeping this narrow means a module can display a user's name without being
// able to mutate accounts.
type UserStore interface {
	GetUser(ctx context.Context, id string) (*User, error)
	// FindUserByEmail returns a user or ErrNotFound. It exists so an
	// invitation acceptance can be matched to an existing account without
	// exposing a listing surface.
	FindUserByEmail(ctx context.Context, email string) (*User, error)
}

// User is the public projection of an account. It deliberately omits the
// password hash and any token material.
type User struct {
	ID              string
	Email           string
	Phone           string
	Name            string
	Locale          string
	AvatarURL       string
	Role            string
	Status          string
	EmailVerifiedAt *time.Time
	PhoneVerifiedAt *time.Time
	ClosedAt        *time.Time
	CreatedAt       time.Time
}
