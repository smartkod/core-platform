// Package economy is the L2 money module: a token wallet with an append-only
// ledger, a rewarded-ads engine, and the purchase pipeline (products, orders,
// payments, entitlements).
//
// It is an L2 module: it depends on identity for authentication but never on
// another L1 module's internals. The public surface for applications lives in
// contract/economy; this package only wires it.
package economy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"core-platform/contract/core"
	econ "core-platform/contract/economy"
	"core-platform/mod/economy/api"
	economymigrations "core-platform/mod/economy/migrations"
	"core-platform/mod/economy/provider"
	"core-platform/mod/economy/repository"
	"core-platform/mod/economy/service"
)

// Module is the economy module.
type Module struct {
	cfg      Config
	handler  *api.Handler
	wallet   *service.Wallet
	ads      *service.Ads
	orders   *service.Orders
	ents     *service.Entitlements
	jobs     *service.Jobs
	provider *providerSet
	// stop closes the job worker; nil when it is disabled.
	stop chan struct{}
}

// Config is the module's resolved configuration.
type Config struct {
	// Currencies is the closed allow-list of codes the wallet accepts.
	// Empty enables just TOKEN.
	Currencies []string
	// ManualInstructions is shown on a bank-transfer checkout.
	ManualInstructions string
	// TestWebhookSecret enables the "test" payment adapter. The module refuses
	// to register it when APP_ENV is production, whatever this holds.
	TestWebhookSecret string
	// OrderTTL is how long a pending order can be paid.
	OrderTTL time.Duration
	// JobInterval is how often the expiry/audit sweep runs. Zero disables the
	// worker, which is what tests and one-shot commands want.
	JobInterval time.Duration
}

// New builds the module with defaults.
func New() *Module { return &Module{} }

// NewWithConfig builds the module with explicit settings.
func NewWithConfig(cfg Config) *Module { return &Module{cfg: cfg} }

// Name implements core.Module.
func (m *Module) Name() string { return "economy" }

// Version implements core.Module.
func (m *Module) Version() string { return "0.1.0" }

// Dependencies ensures the wallet and orders are only reachable by
// authenticated users.
func (m *Module) Dependencies() []string { return []string{"identity"} }

// MigrationSource is the schema this module adds to the shared core schema.
func (m *Module) MigrationSource() core.MigrationSource {
	return economymigrations.Source()
}

// Migrate is a no-op: the runner applies MigrationSource before Start.
func (m *Module) Migrate(context.Context, *gorm.DB) error { return nil }

// Start wires the repository, services, providers and handler.
func (m *Module) Start(ctx context.Context, d core.Deps) error {
	if d.DB == nil {
		return fmt.Errorf("economy: no database in dependencies")
	}

	// Configuration is resolved once, at start: explicit Config wins, then the
	// deployment's YAML/env through the narrow ConfigView (same pattern as the
	// identity module's secrets).
	m.cfg = m.resolveConfig(d)

	store := repository.NewStore(d.DB)
	m.wallet = service.NewWallet(store, m.cfg.Currencies, d.Clock)

	var clock func() time.Time
	if d.Clock != nil {
		clock = func() time.Time { return d.Clock.Now().UTC() }
	}
	m.ads = service.NewAds(m.wallet, store, clock)
	m.ents = service.NewEntitlements(store, clock)

	providers, err := m.buildProviders(d)
	if err != nil {
		return err
	}
	m.orders = service.NewOrders(store, m.wallet, providers, clock, m.cfg.OrderTTL)
	m.jobs = service.NewJobs(store, clock)
	m.handler = api.NewHandler(api.Deps{
		Wallet:       m.wallet,
		Ads:          m.ads,
		Orders:       m.orders,
		Entitlements: m.ents,
	})
	m.provider = &providerSet{
		wallet: m.wallet,
		ads:    m.ads,
		orders: m.orders,
		ents:   m.ents,
	}

	if d.Logger != nil {
		names := make([]string, 0, len(providers))
		for name := range providers {
			names = append(names, name)
		}
		d.Logger.Info("economy ready", "module", m.Name(), "payment_providers", names)
	}

	// The sweep is started only when an interval is configured. The deployment
	// default is one minute; setting economy.job_interval to 0s disables it
	// (tests and one-shot commands rely on that).
	if m.cfg.JobInterval > 0 {
		m.stop = make(chan struct{})
		go m.worker(d)
	}
	return nil
}

// worker runs the expiry and audit sweep on an interval.
//
// It stops when Stop closes the channel. A failed sweep is logged and retried on
// the next tick: the job is idempotent, so a transient database error must not
// take down the module.
func (m *Module) worker(d core.Deps) {
	ticker := time.NewTicker(m.cfg.JobInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stop:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			report, err := m.jobs.RunOnce(ctx)
			cancel()

			if d.Logger == nil {
				continue
			}
			switch {
			case err != nil:
				d.Logger.Error("economy job failed", "err", err.Error())
			case report.Skipped:
				// Another instance held the lock; nothing to report.
			case len(report.LedgerDrift) > 0:
				// A drift row is an incident: the ledger and the balance must
				// always agree, so this is an error, not an info line.
				d.Logger.Error("economy ledger drift",
					"accounts", len(report.LedgerDrift),
					"expired_orders", report.ExpiredOrders)
			case report.ExpiredOrders > 0:
				d.Logger.Info("economy job",
					"expired_orders", report.ExpiredOrders,
					"active_entitlements", report.ActiveEntitlements)
			}
		}
	}
}

// resolveConfig fills the gaps from the deployment configuration.
func (m *Module) resolveConfig(d core.Deps) Config {
	cfg := m.cfg
	if d.Config == nil {
		return cfg
	}
	if len(cfg.Currencies) == 0 {
		if v := d.Config.String("economy.currencies"); v != "" {
			for _, c := range strings.Split(v, ",") {
				if c = strings.TrimSpace(c); c != "" {
					cfg.Currencies = append(cfg.Currencies, strings.ToUpper(c))
				}
			}
		}
	}
	if cfg.OrderTTL <= 0 {
		cfg.OrderTTL = d.Config.Duration("economy.order_ttl")
	}
	if cfg.JobInterval == 0 {
		cfg.JobInterval = d.Config.Duration("economy.job_interval")
	}
	if cfg.ManualInstructions == "" {
		cfg.ManualInstructions = d.Config.String("economy.manual_instructions")
	}
	if cfg.TestWebhookSecret == "" {
		cfg.TestWebhookSecret = d.Config.String("economy.test_webhook_secret")
	}
	return cfg
}

// buildProviders registers the adapters the deployment may use.
//
// The test adapter is a development tool: registering it in production would
// let a signed-but-forgeable webhook settle orders, so APP_ENV is checked here
// rather than left to configuration discipline.
func (m *Module) buildProviders(d core.Deps) (map[string]econ.PaymentProvider, error) {
	out := map[string]econ.PaymentProvider{
		"token":  provider.Token{},
		"manual": provider.Manual{Instructions: m.cfg.ManualInstructions},
	}

	if m.cfg.TestWebhookSecret != "" {
		env := ""
		if d.Config != nil {
			env = d.Config.Env()
		}
		if env == "production" {
			if d.Logger != nil {
				d.Logger.Warn("economy: test payment provider ignored in production")
			}
		} else {
			out["test"] = provider.Test{Secret: m.cfg.TestWebhookSecret}
		}
	}
	return out, nil
}

// RegisterRoutes mounts the endpoints.
func (m *Module) RegisterRoutes(g *echo.Group, d core.Deps) {
	if m.handler == nil {
		panic("economy: RegisterRoutes called before Start")
	}
	if d.Identity == nil {
		panic("economy: no identity in dependencies; economy routes cannot be public")
	}
	m.handler.Register(g, d.Identity)
}

// Stop releases resources: the job worker is told to exit, and Stop is
// idempotent so a second call is a no-op.
func (m *Module) Stop(context.Context) error {
	if m.stop != nil {
		select {
		case <-m.stop:
			// Already stopped.
		default:
			close(m.stop)
		}
		m.stop = nil
	}
	return nil
}

// providerSet implements core.EconomyProvider so the application reaches the
// contracts through Deps.Economy without importing this package.
type providerSet struct {
	wallet *service.Wallet
	ads    *service.Ads
	orders *service.Orders
	ents   *service.Entitlements
}

func (p *providerSet) Economy() core.Economy { return p }

// Wallet exposes the ledger contract.
func (p *providerSet) Wallet() econ.Wallet { return p.wallet }

// Ads exposes the rewarded-ads contract.
func (p *providerSet) Ads() econ.Ads { return p.ads }

// Orders exposes the purchase contract.
func (p *providerSet) Orders() econ.Orders { return p.orders }

// Entitlements exposes the granted-rights contract.
func (p *providerSet) Entitlements() econ.Entitlements { return p.ents }

// Wallet exposes the service for tests inside the same binary.
func (m *Module) Wallet() econ.Wallet { return m.wallet }

// Ads exposes the service for tests inside the same binary.
func (m *Module) Ads() econ.Ads { return m.ads }

// Orders exposes the service for tests inside the same binary.
func (m *Module) Orders() econ.Orders { return m.orders }

// Entitlements exposes the service for tests inside the same binary.
func (m *Module) Entitlements() econ.Entitlements { return m.ents }
