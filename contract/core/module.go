package core

import (
	"context"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// Module is the contract every platform module and application module
// implements. The manager starts modules in dependency order and stops them
// in reverse order.
type Module interface {
	// Name is the unique module identifier used in dependencies and config.
	Name() string
	// Version is informational only.
	Version() string
	// Dependencies lists module names that must start before this one.
	Dependencies() []string
	// Migrate applies the module's schema. It must be safe to call when there
	// is nothing to do.
	Migrate(ctx context.Context, db *gorm.DB) error
	// RegisterRoutes mounts the module's HTTP routes.
	RegisterRoutes(g *echo.Group, d Deps)
	// Start acquires runtime resources. Stop must undo it.
	//
	// Deps is passed here rather than stored on the module so a module never
	// holds a database handle before the pool is proven reachable.
	Start(ctx context.Context, d Deps) error
	// Stop releases runtime resources. It must be idempotent.
	Stop(ctx context.Context) error
}

// Deps is the set of core services a module may use.
//
// Adding a field is a minor version change; removing or changing one is major
// (see PRD.md §8.4).
//
// Every service below is optional: it is populated by the bootstrap only when
// a module that provides it is compiled into the binary and enabled. A module
// that requires one must check for nil in Start and fail loudly rather than
// serve a half-working surface.
type Deps struct {
	Logger Logger
	Clock  Clock
	DB     *gorm.DB
	Config ConfigView
	// Identity is populated by the bootstrap when an identity module is present.
	// Files, notifications and every application module authenticate through it
	// rather than importing the identity module, which is what keeps modules
	// independent of one another.
	Identity Identity
	// Users is the read-only account projection. An engine uses it to show a
	// name, never to mutate an account.
	Users UserStore
	// Files is populated by the bootstrap when the files module is present.
	Files Files
	// Notifications is populated by the bootstrap when the notifications module
	// is present.
	Notifications Notifications
	// Authorizer is the authorization service (role + permission + scope).
	// It is the only way a module may ask "may this actor do this here".
	Authorizer Authorizer
	// Organizations owns business profiles, branches, memberships and service
	// areas.
	Organizations Organizations
	// Catalog owns the central taxonomy and the sector-independent offering
	// record.
	Catalog Catalog
	// WriteLimiter bounds resource creation per account. It is a middleware
	// rather than a service because every module that creates rows should apply
	// the same policy, and a module must not build its own limiter: separate
	// limiters would multiply the quota by the number of modules.
	//
	// Nil means "not configured", and a module then applies no limit. The
	// bootstrap always sets it, so nil only happens in tests.
	WriteLimiter echo.MiddlewareFunc
}

// ConfigView is the slice of configuration a module is allowed to read.
//
// It is an interface rather than the concrete config struct so modules stay
// decoupled from the file format and from each other, and so tests can supply a
// map. Keys are dotted and namespaced, e.g. "identity.access_secret".
//
// An empty key is always a programming error, not a configuration value: the
// adapter reports it as empty and the module's Start fails, which is the same
// outcome as a missing setting.
type ConfigView interface {
	Env() string
	String(key string) string
	Int(key string) int
	Duration(key string) time.Duration
	Bool(key string) bool
}

// Logger is the subset of the platform logger that modules may use.
type Logger interface {
	Info(msg string, kv ...any)
	Warn(msg string, kv ...any)
	Error(msg string, kv ...any)
	With(kv ...any) Logger
}

// Clock makes time injectable so scheduled behaviour is testable without
// sleeping.
type Clock interface {
	Now() time.Time
}
