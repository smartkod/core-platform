// Package economy is the L2 money module: a token wallet with an append-only
// ledger and a rewarded-ads engine that pays through the same funnel.
//
// It is an L2 module: it depends on identity for authentication but never on
// another L1 module's internals. The public surface for applications lives in
// contract/economy; this package only wires it.
package economy

import (
	"context"
	"fmt"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"core-platform/contract/core"
	econ "core-platform/contract/economy"
	"core-platform/mod/economy/api"
	economymigrations "core-platform/mod/economy/migrations"
	"core-platform/mod/economy/repository"
	"core-platform/mod/economy/service"
)

// Module is the economy module.
type Module struct {
	handler    *api.Handler
	wallet     *service.Wallet
	ads        *service.Ads
	provider   *provider
	currencies []string
}

// Config is the module's resolved configuration.
type Config struct {
	// Currencies is the closed allow-list of codes the wallet accepts.
	// Empty enables just TOKEN.
	Currencies []string
}

// New builds the module with defaults.
func New() *Module { return &Module{} }

// NewWithConfig builds the module with explicit settings.
func NewWithConfig(cfg Config) *Module { return &Module{currencies: cfg.Currencies} }

// Name implements core.Module.
func (m *Module) Name() string { return "economy" }

// Version implements core.Module.
func (m *Module) Version() string { return "0.1.0" }

// Dependencies ensures the wallet is only reachable by authenticated users.
func (m *Module) Dependencies() []string { return []string{"identity"} }

// MigrationSource is the schema this module adds to the shared core schema.
func (m *Module) MigrationSource() core.MigrationSource {
	return economymigrations.Source()
}

// Migrate is a no-op: the runner applies MigrationSource before Start.
func (m *Module) Migrate(context.Context, *gorm.DB) error { return nil }

// Start wires the repository, services and handler.
func (m *Module) Start(ctx context.Context, d core.Deps) error {
	if d.DB == nil {
		return fmt.Errorf("economy: no database in dependencies")
	}

	store := repository.NewStore(d.DB)
	m.wallet = service.NewWallet(store, m.currencies, d.Clock)
	m.ads = service.NewAds(m.wallet, store, nil)
	m.handler = api.NewHandler(m.wallet, m.ads)
	m.provider = &provider{wallet: m.wallet, ads: m.ads}

	if d.Logger != nil {
		d.Logger.Info("economy ready", "module", m.Name())
	}
	return nil
}

// RegisterRoutes mounts the endpoints.
func (m *Module) RegisterRoutes(g *echo.Group, d core.Deps) {
	if m.handler == nil {
		panic("economy: RegisterRoutes called before Start")
	}
	if d.Identity == nil {
		panic("economy: no identity in dependencies; wallet routes cannot be public")
	}
	m.handler.Register(g, d.Identity)
}

// Stop releases resources. The module holds no background workers yet.
func (m *Module) Stop(context.Context) error { return nil }

// provider implements core.EconomyProvider so the application reaches the
// contracts through Deps.Economy without importing this package.
type provider struct {
	wallet econ.Wallet
	ads    econ.Ads
}

func (p *provider) Economy() core.Economy { return p }

// Wallet exposes the ledger contract.
func (p *provider) Wallet() econ.Wallet { return p.wallet }

// Ads exposes the rewarded-ads contract.
func (p *provider) Ads() econ.Ads { return p.ads }

// Wallet exposes the service for tests inside the same binary.
func (m *Module) Wallet() econ.Wallet { return m.wallet }

// Ads exposes the service for tests inside the same binary.
func (m *Module) Ads() econ.Ads { return m.ads }
