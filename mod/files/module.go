// Package files stores uploads and controls who can read them.
//
// Files are an L1 module because almost every application needs a place to put
// a user's avatar, receipt or attachment, and doing it well is not obvious: the
// interesting work is content validation, ownership and the fact that the
// metadata and the bytes can disagree.
package files

import (
	"context"
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"core-platform/contract/core"
	"core-platform/mod/files/api"
	"core-platform/mod/files/migrations"
	"core-platform/mod/files/repository"
	"core-platform/mod/files/service"
	"core-platform/platform/storage"
)

// Config is the module's resolved configuration.
type Config struct {
	// Driver is "local" or "s3".
	Driver string
	// LocalRoot is the storage directory when Driver is "local".
	LocalRoot string
	// PublicBaseURL prefixes generated download links.
	PublicBaseURL string
	// MaxSize is the per-upload byte limit.
	MaxSize int64
	// AllowedTypes is the closed list of accepted content types.
	AllowedTypes []string
	// SignedURLTTL is how long a minted link stays valid.
	SignedURLTTL time.Duration
}

// DefaultConfig returns a safe, self-contained setup.
func DefaultConfig() Config {
	return Config{
		Driver:       "local",
		LocalRoot:    "data/uploads",
		MaxSize:      10 << 20,
		SignedURLTTL: 15 * time.Minute,
	}
}

// Module is the files module.
type Module struct {
	cfg     Config
	service *service.Files
	handler *api.Handler
	store   core.Storage
}

// New builds the module with defaults.
func New() *Module { return &Module{} }

// NewWithConfig builds the module with explicit settings.
func NewWithConfig(cfg Config) *Module { return &Module{cfg: cfg} }

// Name implements core.Module.
func (m *Module) Name() string { return "files" }

// Version implements core.Module.
func (m *Module) Version() string { return "0.1.0" }

// Dependencies ensures users exist before files reference them.
func (m *Module) Dependencies() []string { return []string{"identity"} }

// MigrationSource is the schema this module adds to the shared core schema.
func (m *Module) MigrationSource() core.MigrationSource { return migrations.Source() }

// Migrate is a no-op: the runner applies MigrationSource before Start.
func (m *Module) Migrate(context.Context, *gorm.DB) error { return nil }

// Start wires the storage driver, the service and the handler.
func (m *Module) Start(ctx context.Context, d core.Deps) error {
	if d.DB == nil {
		return fmt.Errorf("files: no database in dependencies")
	}

	cfg := m.cfg
	if cfg.Driver == "" {
		cfg.Driver = "local"
	}
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = 10 << 20
	}
	if cfg.SignedURLTTL <= 0 {
		cfg.SignedURLTTL = 15 * time.Minute
	}
	if cfg.LocalRoot == "" {
		cfg.LocalRoot = "data/uploads"
	}

	store, err := m.buildStorage(cfg)
	if err != nil {
		return err
	}

	valid := service.NewValidator(cfg.AllowedTypes, cfg.MaxSize)
	files := service.NewFiles(store, repository.NewFiles(d.DB), valid)

	m.cfg = cfg
	m.store = store
	m.service = files
	m.handler = api.NewHandler(files, api.Config{
		PublicBaseURL: cfg.PublicBaseURL,
		MaxSize:       cfg.MaxSize,
		SignedURLTTL:  cfg.SignedURLTTL,
	})

	if d.Logger != nil {
		d.Logger.Info("files ready",
			"driver", store.Driver(),
			"max_size_mb", cfg.MaxSize/(1<<20))
	}
	return nil
}

// buildStorage selects the driver.
//
// An unknown driver name is an error rather than a silent fallback to disk: a
// deployment that believes it is writing to S3 while writing to a container's
// ephemeral filesystem loses every file on the next restart.
func (m *Module) buildStorage(cfg Config) (core.Storage, error) {
	switch cfg.Driver {
	case "local":
		return storage.NewLocal(cfg.LocalRoot, cfg.PublicBaseURL)
	case "s3":
		return nil, fmt.Errorf(
			"files: the s3 driver is not implemented yet; use driver \"local\" " +
				"(the core.Storage interface is ready, so adding it touches no other code)")
	default:
		return nil, fmt.Errorf("files: unknown driver %q (want \"local\")", cfg.Driver)
	}
}

// RegisterRoutes mounts the endpoints. The identity middleware is required, so
// the module refuses to serve anything without it.
func (m *Module) RegisterRoutes(g *echo.Group, d core.Deps) {
	if m.handler == nil {
		panic("files: RegisterRoutes called before Start")
	}
	if d.Identity == nil {
		panic("files: no identity in dependencies; file routes cannot be public")
	}
	m.handler.Register(g, d.Identity)
}

// Stop releases resources. Local disk needs none.
func (m *Module) Stop(context.Context) error { return nil }

// Storage exposes the driver for other modules in the same binary.
func (m *Module) Storage() core.Storage { return m.store }

// Service exposes the file service for other modules in the same binary.
func (m *Module) Service() core.Files { return m.service }

// Files implements core.FileProvider so other modules can reach the service
// through Deps without importing this package.
func (m *Module) Files() core.Files { return m.service }
