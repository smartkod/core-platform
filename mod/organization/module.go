// Package organization owns business profiles, branches, memberships, regions
// and service areas.
//
// It is the tenant boundary of the platform: every other module that stores
// tenant data points at an organization id, and every tenant-scoped route is
// authorized against a membership in that organization. Nothing here knows how
// a sector sells, books or bills; that lives in an engine.
package organization

import (
	"context"
	"fmt"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"core-platform/contract/core"
	"core-platform/mod/organization/api"
	"core-platform/mod/organization/migrations"
	"core-platform/mod/organization/repository"
	"core-platform/mod/organization/service"
)

// Module is the organization module.
type Module struct {
	svc     *service.Organizations
	handler *api.Handler
}

// New builds the module.
func New() *Module { return &Module{} }

// Name implements core.Module.
func (m *Module) Name() string { return "organization" }

// Version implements core.Module.
func (m *Module) Version() string { return "0.1.0" }

// Dependencies: accounts must exist, and every organization action is decided
// by the authorizer rather than by a role stored on the user.
func (m *Module) Dependencies() []string { return []string{"identity", "authz"} }

// MigrationSource is the schema this module owns.
func (m *Module) MigrationSource() core.MigrationSource { return migrations.Source() }

// Migrate is a no-op: the runner applies MigrationSource before Start.
func (m *Module) Migrate(context.Context, *gorm.DB) error { return nil }

// Start wires the repository, the service and the handler.
func (m *Module) Start(ctx context.Context, d core.Deps) error {
	if d.DB == nil {
		return fmt.Errorf("organization: no database in dependencies")
	}
	if d.Authorizer == nil {
		// Serving organization routes without an authorizer would mean either
		// allowing everyone or silently denying everyone. Both are worse than
		// refusing to start.
		return fmt.Errorf("organization: no authorizer in dependencies")
	}

	m.svc = service.New(repository.NewStore(d.DB), d.Authorizer)

	var public echo.MiddlewareFunc
	if d.Identity != nil {
		public = d.Identity.OptionalMiddleware()
	}
	m.handler = api.NewHandler(m.svc, d.Authorizer, public)

	if d.Logger != nil {
		d.Logger.Info("organization ready")
	}
	return nil
}

// RegisterRoutes mounts the organization endpoints.
func (m *Module) RegisterRoutes(g *echo.Group, d core.Deps) {
	if m.handler == nil {
		panic("organization: RegisterRoutes called before Start")
	}
	if d.Identity == nil {
		panic("organization: no identity in dependencies")
	}
	m.handler.Register(g, d.Identity, d.WriteLimiter)
}

// Stop releases resources.
func (m *Module) Stop(context.Context) error { return nil }

// Organizations implements core.OrganizationProvider.
func (m *Module) Organizations() core.Organizations { return m.svc }
