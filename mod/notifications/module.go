// Package notifications queues messages and delivers them reliably.
//
// The module exists because "send an e-mail" is never just that. A mail server
// can be down, a template can be wrong, a user can have disabled a channel, and
// a retry must not send the message twice. The outbox pattern answers all four:
// the message is written first, delivered later, and the attempt is recorded.
package notifications

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"core-platform/contract/core"
	"core-platform/mod/notifications/api"
	"core-platform/mod/notifications/migrations"
	"core-platform/mod/notifications/model"
	"core-platform/mod/notifications/repository"
	"core-platform/mod/notifications/service"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// Config is the module's resolved configuration.
type Config struct {
	// Mailer is the transport. Nil means e-mail is queued but never delivered,
	// which is the right default for a development environment with no SMTP.
	Mailer core.Mailer
	// MaxAttempts bounds retries before a message is abandoned.
	MaxAttempts int
	// BaseDelay is the first retry delay.
	BaseDelay time.Duration
	// MaxDelay caps the exponential backoff.
	MaxDelay time.Duration
	// WorkerInterval is how often the delivery loop runs. Zero disables it.
	WorkerInterval time.Duration
	// BatchSize is how many messages one pass may send.
	BatchSize int
}

// DefaultConfig returns a safe, worker-enabled setup.
func DefaultConfig() Config {
	c := service.DefaultConfig()
	return Config{
		MaxAttempts:    c.MaxAttempts,
		BaseDelay:      c.BaseDelay,
		MaxDelay:       c.MaxDelay,
		WorkerInterval: 10 * time.Second,
		BatchSize:      25,
	}
}

// Module is the notifications module.
type Module struct {
	cfg     Config
	svc     *service.Notifications
	repo    *repository.Notifications
	handler *api.Handler
	stop    chan struct{}
}

// New builds the module with defaults.
func New() *Module { return &Module{} }

// NewWithConfig builds the module with explicit settings.
func NewWithConfig(cfg Config) *Module { return &Module{cfg: cfg} }

// Name implements core.Module.
func (m *Module) Name() string { return "notifications" }

// Version implements core.Module.
func (m *Module) Version() string { return "0.1.0" }

// Dependencies ensures users exist before notifications reference them.
func (m *Module) Dependencies() []string { return []string{"identity"} }

// MigrationSource is the schema this module adds to the shared core schema.
func (m *Module) MigrationSource() core.MigrationSource { return migrations.Source() }

// Migrate is a no-op: the runner applies MigrationSource before Start.
func (m *Module) Migrate(context.Context, *gorm.DB) error { return nil }

// Start wires the service, the handler and the delivery worker.
func (m *Module) Start(ctx context.Context, d core.Deps) error {
	if d.DB == nil {
		return fmt.Errorf("notifications: no database in dependencies")
	}

	cfg := m.cfg
	if cfg.MaxAttempts <= 0 {
		def := DefaultConfig()
		cfg.MaxAttempts = def.MaxAttempts
		cfg.BaseDelay = def.BaseDelay
		cfg.MaxDelay = def.MaxDelay
		if cfg.WorkerInterval == 0 {
			cfg.WorkerInterval = def.WorkerInterval
		}
		if cfg.BatchSize == 0 {
			cfg.BatchSize = def.BatchSize
		}
	}

	renderer, err := service.NewRenderer(templateFS, "templates")
	if err != nil {
		return err
	}

	repo := repository.NewNotifications(d.DB)
	svc := service.New(repo, renderer, cfg.Mailer, service.Config{
		MaxAttempts: cfg.MaxAttempts,
		BaseDelay:   cfg.BaseDelay,
		MaxDelay:    cfg.MaxDelay,
	})

	m.cfg = cfg
	m.repo = repo
	m.svc = svc
	m.handler = api.NewHandler(svc)

	if cfg.WorkerInterval > 0 {
		m.stop = make(chan struct{})
		go m.worker(d)
	}

	if d.Logger != nil {
		d.Logger.Info("notifications ready",
			"templates", len(renderer.Codes()),
			"worker", cfg.WorkerInterval > 0,
			"mailer", cfg.Mailer != nil)
	}
	return nil
}

// worker drains the outbox on an interval.
//
// It stops when Stop closes the channel, and it logs the counts so a stuck queue
// is visible in the logs rather than discovered by a user who never got mail.
func (m *Module) worker(d core.Deps) {
	ticker := time.NewTicker(m.cfg.WorkerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stop:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			sent, failed, err := m.svc.DeliverPending(ctx, m.cfg.BatchSize)
			cancel()

			if d.Logger != nil {
				switch {
				case err != nil:
					d.Logger.Error("notification worker failed", "err", err.Error())
				case sent > 0 || failed > 0:
					d.Logger.Info("notification worker", "sent", sent, "failed", failed)
				}
			}
		}
	}
}

// RegisterRoutes mounts the endpoints.
func (m *Module) RegisterRoutes(g *echo.Group, d core.Deps) {
	if m.handler == nil {
		panic("notifications: RegisterRoutes called before Start")
	}
	if d.Identity == nil {
		panic("notifications: no identity in dependencies; an inbox cannot be public")
	}
	m.handler.Register(g, d.Identity, &prefs{m.repo}, m.codes)
}

func (m *Module) codes() []string { return nil }

// Stop shuts the worker down.
func (m *Module) Stop(context.Context) error {
	if m.stop != nil {
		close(m.stop)
		m.stop = nil
	}
	return nil
}

// Notifications implements core.NotificationProvider.
func (m *Module) Notifications() core.Notifications { return m.svc }

// DeliverNow runs one delivery pass immediately. Exposed for tests and for an
// admin trigger; the background worker uses the same code.
func (m *Module) DeliverNow(ctx context.Context, limit int) (int, int, error) {
	return m.svc.DeliverPending(ctx, limit)
}

// prefs adapts the repository to the API's preference interface.
type prefs struct{ repo *repository.Notifications }

func (p *prefs) Get(ctx context.Context, userID string) (*core.NotificationPreferences, error) {
	rec, err := p.repo.Preferences(ctx, userID)
	if err != nil || rec == nil {
		return nil, err
	}

	stored := rec.ChannelsMap()
	channels := make(map[core.NotificationChannel]bool, len(stored))
	for ch, on := range stored {
		channels[core.NotificationChannel(ch)] = on
	}
	return &core.NotificationPreferences{
		UserID:    rec.UserID,
		Channels:  channels,
		UpdatedAt: rec.UpdatedAt,
	}, nil
}

func (p *prefs) Save(ctx context.Context, in *core.NotificationPreferences) error {
	stored := make(map[string]bool, len(in.Channels))
	for ch, on := range in.Channels {
		stored[string(ch)] = on
	}

	encoded, err := json.Marshal(stored)
	if err != nil {
		return err
	}
	return p.repo.SavePreferences(ctx, &model.Preference{
		UserID:   in.UserID,
		Channels: string(encoded),
	})
}
