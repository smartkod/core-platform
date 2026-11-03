// Package authz owns roles, permissions and scoped grants.
//
// It answers exactly one question — "may this actor perform this action in this
// scope" — and it answers "no" unless an active grant says otherwise. Business
// authority lives here rather than on the user record, because a person can be
// an owner in one organization and staff in another, and a single role column
// cannot express that without a second user account.
package authz

import (
	"context"
	"fmt"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"core-platform/contract/core"
	"core-platform/mod/authz/api"
	"core-platform/mod/authz/migrations"
	"core-platform/mod/authz/repository"
	"core-platform/mod/authz/service"
)

// Module is the authorization module.
type Module struct {
	svc     *service.Authz
	handler *api.Handler
}

// New builds the module.
func New() *Module { return &Module{} }

// Name implements core.Module.
func (m *Module) Name() string { return "authz" }

// Version implements core.Module.
func (m *Module) Version() string { return "0.1.0" }

// Dependencies ensures accounts exist before a grant references one.
func (m *Module) Dependencies() []string { return []string{"identity"} }

// MigrationSource is the schema this module owns.
func (m *Module) MigrationSource() core.MigrationSource { return migrations.Source() }

// Migrate is a no-op: the runner applies MigrationSource before Start.
func (m *Module) Migrate(context.Context, *gorm.DB) error { return nil }

// Start wires the repository, the service and the handler.
func (m *Module) Start(ctx context.Context, d core.Deps) error {
	if d.DB == nil {
		return fmt.Errorf("authz: no database in dependencies")
	}

	repo := repository.NewStore(d.DB)
	m.svc = service.New(repo, d.Users)
	m.handler = api.NewHandler(m.svc)

	if d.Logger != nil {
		d.Logger.Info("authz ready")
	}
	return nil
}

// RegisterRoutes mounts the authorization endpoints.
func (m *Module) RegisterRoutes(g *echo.Group, d core.Deps) {
	if m.handler == nil {
		panic("authz: RegisterRoutes called before Start")
	}
	if d.Identity == nil {
		panic("authz: no identity in dependencies; grants cannot be public")
	}
	m.handler.Register(g, d.Identity)
}

// Stop releases resources.
func (m *Module) Stop(context.Context) error { return nil }

// Authorizer implements core.AuthorizerProvider.
func (m *Module) Authorizer() core.Authorizer { return m.svc }

// Service exposes the concrete service for modules compiled in the same binary
// that need RevokeScope when a tenant goes away.
func (m *Module) Service() *service.Authz { return m.svc }
