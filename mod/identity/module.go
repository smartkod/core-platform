// Package identity implements accounts and sessions.
//
// It is the first L1 module: authorization, files, notifications, organization
// and catalog all depend on knowing who the caller is.
package identity

import (
	"context"
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"core-platform/contract/core"
	"core-platform/mod/identity/api"
	"core-platform/mod/identity/migrations"
	"core-platform/mod/identity/repository"
	"core-platform/mod/identity/service"
	"core-platform/platform/security"
	"core-platform/platform/uuid"
)

// Config is the module's resolved configuration.
type Config struct {
	// AccessSecret signs access tokens.
	AccessSecret string
	// RefreshSecret must differ from AccessSecret. Refresh tokens are opaque
	// today, so nothing is signed with it yet; the separation is enforced now
	// so a future signed format cannot accidentally share a key.
	RefreshSecret string

	AccessTTL          time.Duration
	RefreshTTL         time.Duration
	RefreshTTLRemember time.Duration

	// PasswordMinLength is enforced at registration and reset.
	PasswordMinLength int
	// Issuer is embedded in access tokens and checked on parse.
	Issuer string
	// CookieSecure must be true when served over HTTPS.
	CookieSecure bool

	// LoginLimit and LoginWindow bound credentialed attempts.
	LoginLimit  int
	LoginWindow time.Duration

	// Mailer is optional; a no-op mailer is used when nil.
	Mailer service.Mailer
}

// DefaultConfig returns the values described in PRD.md §8.1.
func DefaultConfig() Config {
	return Config{
		AccessTTL:          15 * time.Minute,
		RefreshTTL:         7 * 24 * time.Hour,
		RefreshTTLRemember: 30 * 24 * time.Hour,
		PasswordMinLength:  10,
		Issuer:             "core-platform",
		LoginLimit:         10,
		LoginWindow:        time.Minute,
	}
}

// Module is the identity module.
type Module struct {
	cfg Config

	handler  *api.Handler
	identity *api.Identity
	limiter  *security.RateLimiter
}

// New builds the module with defaults.
//
// Every setting is read from Deps.Config in Start, so a module never captures a
// configuration value at construction time and cannot go stale after a reload.
func New() *Module { return &Module{} }

// NewWithConfig builds the module with explicit settings, for tests and for
// embedders that keep their own configuration.
func NewWithConfig(cfg Config) *Module { return &Module{cfg: cfg} }

// Name implements core.Module.
func (m *Module) Name() string { return "identity" }

// Version implements core.Module.
func (m *Module) Version() string { return "0.1.0" }

// Dependencies implements core.Module. Identity is the base of L1.
func (m *Module) Dependencies() []string { return nil }

// MigrationSource is the schema this module owns.
func (m *Module) MigrationSource() core.MigrationSource { return migrations.Source() }

// Migrate is a no-op: the shared runner applies MigrationSource before any
// module starts, so schema work is never duplicated here.
func (m *Module) Migrate(context.Context, *gorm.DB) error { return nil }

// Start wires the service graph.
func (m *Module) Start(ctx context.Context, d core.Deps) error {
	if d.DB == nil {
		return fmt.Errorf("identity: no database in dependencies")
	}

	cfg := m.cfg
	if cfg.AccessSecret == "" && d.Config != nil {
		if v := d.Config.String("identity.access_secret"); v != "" {
			cfg.AccessSecret = v
		}
	}
	if cfg.RefreshSecret == "" && d.Config != nil {
		if v := d.Config.String("identity.refresh_secret"); v != "" {
			cfg.RefreshSecret = v
		}
	}
	if d.Config != nil && d.Config.Env() == "production" {
		// In production the platform config has already refused to boot without
		// secrets, so reaching here with an empty value means a module was
		// wired without config at all. Fail rather than invent one.
		cfg.CookieSecure = true
	}
	if cfg.Issuer == "" {
		cfg.Issuer = "core-platform"
	}
	if cfg.PasswordMinLength <= 0 {
		cfg.PasswordMinLength = 10
	}
	if cfg.LoginLimit <= 0 {
		cfg.LoginLimit = 10
	}
	if cfg.LoginWindow <= 0 {
		cfg.LoginWindow = time.Minute
	}
	if cfg.AccessTTL <= 0 {
		cfg.AccessTTL = 15 * time.Minute
	}
	if cfg.RefreshTTL <= 0 {
		cfg.RefreshTTL = 7 * 24 * time.Hour
	}
	if cfg.RefreshTTLRemember <= 0 {
		cfg.RefreshTTLRemember = 30 * 24 * time.Hour
	}

	if cfg.AccessSecret == "" {
		return fmt.Errorf("identity: an access token secret is required")
	}
	if cfg.RefreshSecret == "" {
		return fmt.Errorf("identity: a refresh token secret is required")
	}
	if cfg.AccessSecret == cfg.RefreshSecret {
		return fmt.Errorf("identity: access and refresh secrets must differ")
	}
	m.cfg = cfg

	tokens, err := service.NewTokens(service.TokenConfig{
		AccessSecret:       cfg.AccessSecret,
		RefreshSecret:      cfg.RefreshSecret,
		AccessTTL:          cfg.AccessTTL,
		RefreshTTL:         cfg.RefreshTTL,
		RefreshTTLRemember: cfg.RefreshTTLRemember,
		Issuer:             cfg.Issuer,
	})
	if err != nil {
		return err
	}

	users := repository.NewUsers(d.DB)
	tokenRepo := repository.NewTokens(d.DB)
	pw := service.NewPassword(cfg.PasswordMinLength)

	auth := service.NewAuth(users, tokenRepo, tokens, pw, cfg.Mailer, service.AuthConfig{
		RefreshTTL:         cfg.RefreshTTL,
		RefreshTTLRemember: cfg.RefreshTTLRemember,
	})

	m.identity = api.NewIdentity(tokens, users)
	m.handler = api.NewHandler(auth, users, api.HandlerConfig{
		CookieSecure: cfg.CookieSecure,
		AccessTTL:    cfg.AccessTTL,
	})
	m.limiter = security.NewRateLimiter(cfg.LoginLimit, cfg.LoginWindow)

	if d.Logger != nil {
		d.Logger.Info("identity ready",
			"access_ttl", cfg.AccessTTL.String(),
			"refresh_ttl", cfg.RefreshTTL.String())
	}
	return nil
}

// RegisterRoutes mounts the identity endpoints.
func (m *Module) RegisterRoutes(g *echo.Group, _ core.Deps) {
	if m.handler == nil {
		// Start did not run. Failing loudly beats serving an empty route table
		// that looks healthy.
		panic("identity: RegisterRoutes called before Start")
	}

	// CSRF double-submit applies to every identity route, and Echo group
	// middleware only sticks to routes registered after the Use call, so it
	// comes first. The middleware itself passes anonymous requests through,
	// which is what keeps login and register working.
	g.Use(api.CSRFProtect())

	limiters := api.NewRateLimiters(m.limiter, m.limiter)
	m.handler.Register(g, m.identity, limiters)
}

// Stop releases resources.
func (m *Module) Stop(context.Context) error { return nil }

// Identity exposes the middleware so the composition root can hand it to other
// modules through core.Deps in a later revision.
func (m *Module) Identity() core.Identity { return m.identity }

// Users implements core.UserProvider, the read-only account projection.
func (m *Module) Users() core.UserStore {
	if m.identity == nil {
		return nil
	}
	return m.identity
}

// NewUUID is exported for tests that need to predict an identifier.
func NewUUID() string { return uuid.New() }
