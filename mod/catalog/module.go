// Package catalog owns the central taxonomy and the sector-independent
// offering record.
//
// It stores what a business offers, not how it is sold. A future engine attaches
// its own record through ExtensionRef; the core never reads that record, which
// is what lets a booking engine and an e-commerce engine share one catalogue
// without sharing a schema.
package catalog

import (
	"context"
	"fmt"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"core-platform/contract/core"
	"core-platform/mod/catalog/api"
	"core-platform/mod/catalog/migrations"
	"core-platform/mod/catalog/repository"
	"core-platform/mod/catalog/service"
)

// Module is the catalog module.
type Module struct {
	svc     *service.Catalog
	handler *api.Handler
}

// New builds the module.
func New() *Module { return &Module{} }

// Name implements core.Module.
func (m *Module) Name() string { return "catalog" }

// Version implements core.Module.
func (m *Module) Version() string { return "0.1.0" }

// Dependencies: an offering belongs to an organization, and every write is
// authorized against that tenant.
func (m *Module) Dependencies() []string { return []string{"identity", "authz", "organization"} }

// MigrationSource is the schema this module owns.
func (m *Module) MigrationSource() core.MigrationSource { return migrations.Source() }

// Migrate is a no-op: the runner applies MigrationSource before Start.
func (m *Module) Migrate(context.Context, *gorm.DB) error { return nil }

// Start wires the repository, the service and the handler.
func (m *Module) Start(ctx context.Context, d core.Deps) error {
	if d.DB == nil {
		return fmt.Errorf("catalog: no database in dependencies")
	}
	if d.Authorizer == nil {
		return fmt.Errorf("catalog: no authorizer in dependencies")
	}
	if d.Organizations == nil {
		return fmt.Errorf("catalog: no organization service in dependencies")
	}

	m.svc = service.New(repository.NewStore(d.DB), d.Organizations, d.Authorizer)
	m.handler = api.NewHandler(m.svc, d.Authorizer)

	if d.Logger != nil {
		d.Logger.Info("catalog ready")
	}
	return nil
}

// RegisterRoutes mounts the catalog endpoints.
func (m *Module) RegisterRoutes(g *echo.Group, d core.Deps) {
	if m.handler == nil {
		panic("catalog: RegisterRoutes called before Start")
	}
	if d.Identity == nil {
		panic("catalog: no identity in dependencies")
	}

	var public echo.MiddlewareFunc
	if d.Identity != nil {
		public = d.Identity.OptionalMiddleware()
	}
	m.handler.Register(g, d.Identity, public)
}

// Stop releases resources.
func (m *Module) Stop(context.Context) error { return nil }

// Catalog implements core.CatalogProvider.
func (m *Module) Catalog() core.Catalog { return m.svc }
