// Package platform is the L0 kernel. It owns bootstrap and knows nothing about
// any domain concept (L0.md 1).
//
// An application's main.go contains only an App value and a call to Run.
package platform

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"core-platform/contract/core"
	"core-platform/platform/clock"
	"core-platform/platform/config"
	"core-platform/platform/database"
	"core-platform/platform/health"
	"core-platform/platform/httpx"
	"core-platform/platform/logger"
	"core-platform/platform/migrate"
	"core-platform/platform/module"
)

// Build metadata, stamped with -ldflags at release time.
var (
	Commit    = "dev"
	BuildTime = "unknown"
)

// Run boots the application and blocks until a termination signal arrives.
//
// The seven steps are fixed (L0.md 2.2); an application never reorders them.
func Run(app core.App) error {
	// 1. Config
	cfg, err := config.Load(app.Name, app.ConfigPath)
	if err != nil {
		return err
	}

	// 2. Logger
	log, err := logger.New(cfg.Log.Level, cfg.Log.Format)
	if err != nil {
		return fmt.Errorf("platform: logger: %w", err)
	}
	log.Info("starting", "app", app.Name, "version", app.Version, "env", cfg.Env)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 3. Database
	db, err := database.Connect(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() { _ = database.Close(db) }()

	deps := core.Deps{Logger: log, Clock: clock.Real{}, DB: db, Config: configView{c: cfg}}

	// 4. Modules: validate the graph before touching any schema, so a bad
	// dependency fails before a partially migrated database is left behind.
	enabled, err := enabledModules(cfg, app.Modules)
	if err != nil {
		return err
	}
	mgr, err := module.New(app.Modules, enabled)
	if err != nil {
		return err
	}
	log.Info("modules resolved", "order", mgr.Order())

	if err := migrateAll(ctx, db, app, mgr, deps); err != nil {
		return err
	}

	// 5. Module start, with reverse-order rollback on failure.
	if err := mgr.Start(ctx, deps); err != nil {
		return err
	}
	defer mgr.Stop(context.Background())

	// 5b. Cross-module services are resolved after every module has started.
	//
	// This must happen after Start, not before: a provider accessor returns the
	// service that Start built, so asking earlier yields a nil pointer wrapped in
	// a non-nil interface. The nil check would pass and the first request would
	// dereference it. Starting first makes the accessors meaningful.
	deps = wireProviders(deps, app.Modules, mgr)

	// 6. HTTP
	srv, err := buildServer(cfg, app, mgr, deps, db, log)
	if err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.Address())
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	// 7. Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return fmt.Errorf("platform: serve: %w", err)
	case sig := <-stop:
		log.Info("shutdown signal", "signal", sig.String())
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.Background(), cfg.HTTP.ShutdownTimeout)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown incomplete", "err", err.Error())
		return err
	}
	log.Info("stopped cleanly")
	return nil
}

// buildServer wires the middleware chain and mounts every enabled module.
//
// Middleware order is fixed by L0.md 4.4.
func buildServer(
	cfg *config.Config,
	app core.App,
	mgr *module.Manager,
	deps core.Deps,
	db *gorm.DB,
	log core.Logger,
) (*http.Server, error) {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.HTTPErrorHandler = httpx.ErrorHandler(log)
	e.RouteNotFound("", httpx.NotFound)
	e.RouteNotFound("/*", httpx.NotFound)

	// The chain is global so health and version responses carry the same
	// request id and security headers as business endpoints (L0.md 4.4).
	// Auth is added per route group by the identity module, never here.
	e.Use(
		httpx.RequestID(),
		httpx.Recover(),
		httpx.Timeout(cfg.HTTP.RequestTimeout),
		httpx.BodyLimit(cfg.HTTP.BodyLimit),
		httpx.SecureHeaders(),
		httpx.CORS(cfg.Security.AllowedOrigins),
	)

	h := health.New(app.Name, app.Version, Commit, BuildTime)
	h.Add("database", func(c context.Context) error { return database.Ping(c, db) })
	h.Register(e.Group(""))

	v1 := e.Group("/api/v1")

	for _, mod := range app.Modules {
		if !mgr.IsEnabled(mod.Name()) {
			continue
		}
		mod.RegisterRoutes(v1.Group(""), deps)
	}

	// Registered after every module so the route table is complete. It is a
	// developer convenience, so it is omitted in production where an attacker
	// benefits more from it than an operator does.
	if cfg.Env != "production" {
		e.GET("/", httpx.Index(app.Name, app.Version, cfg.Env, httpx.RoutesOf(e)))
	}

	return &http.Server{
		Addr:              cfg.Address(),
		Handler:           e,
		ReadHeaderTimeout: 10 * time.Second,
	}, nil
}

// wireProviders fills the cross-module services in Deps by asking each module
// whether it implements the corresponding provider interface.
//
// Interrogation rather than registration is deliberate: a module cannot forget
// to register itself, and an application that swaps the identity implementation
// for its own is wired automatically as long as it satisfies the interface.
//
// A provider that returns nil is skipped rather than stored. Storing a typed nil
// inside an interface produces the classic Go trap: the interface compares
// unequal to nil while every method call on it panics.
func wireProviders(deps core.Deps, mods []core.Module, mgr *module.Manager) core.Deps {
	for _, mod := range mods {
		if !mgr.IsEnabled(mod.Name()) {
			continue
		}
		if p, ok := mod.(core.IdentityProvider); ok && deps.Identity == nil {
			if id := p.Identity(); id != nil {
				deps.Identity = id
			}
		}
		if p, ok := mod.(core.FileProvider); ok && deps.Files == nil {
			if f := p.Files(); f != nil {
				deps.Files = f
			}
		}
		if p, ok := mod.(core.NotificationProvider); ok && deps.Notifications == nil {
			if n := p.Notifications(); n != nil {
				deps.Notifications = n
			}
		}
		if p, ok := mod.(core.EconomyProvider); ok && deps.Economy == nil {
			if e := p.Economy(); e != nil {
				deps.Economy = e
			}
		}
	}
	return deps
}

// migrateAll applies application migrations first, then module migrations.
//
// Module sources are discovered by interface, not by a list in main.go: a
// module that owns tables cannot be started in a binary whose migrations were
// forgotten, because there is no separate list to forget.
func migrateAll(
	ctx context.Context,
	db *gorm.DB,
	app core.App,
	mgr *module.Manager,
	deps core.Deps,
) error {
	sources := make([]core.MigrationSource, 0, len(app.Migrations)+len(app.Modules))
	sources = append(sources, app.Migrations...)

	// Merging by schema is intentional. Core modules share the default "public"
	// schema because their tables are queried without a prefix, so several
	// modules legitimately contribute files to the same history. What must not
	// happen is two sources claiming a schema with independent bookkeeping —
	// that would mean two schema_migrations tables disagreeing about the same
	// schema. Merging into one source per schema is what keeps that impossible.
	bySchema := make(map[string]int, len(sources))
	merged := make([]core.MigrationSource, 0, len(sources))
	for _, s := range sources {
		if i, ok := bySchema[s.Schema]; ok {
			m, err := mergeSources(merged[i], s)
			if err != nil {
				return err
			}
			merged[i] = m
			continue
		}
		bySchema[s.Schema] = len(merged)
		merged = append(merged, s)
	}

	for _, mod := range app.Modules {
		provider, ok := mod.(core.MigrationProvider)
		if !ok {
			continue
		}
		src := provider.MigrationSource()
		if src.FS == nil {
			continue
		}
		if i, ok := bySchema[src.Schema]; ok {
			m, err := mergeSources(merged[i], src)
			if err != nil {
				return fmt.Errorf("platform: module %q: %w", mod.Name(), err)
			}
			merged[i] = m
			continue
		}
		bySchema[src.Schema] = len(merged)
		merged = append(merged, src)
	}

	if err := migrate.Run(ctx, db, merged); err != nil {
		return err
	}
	return mgr.Migrate(ctx, deps)
}

// mergeSources combines two sources that target the same schema.
//
// Sharing a schema is normal for core modules, so the same directory is fine as
// long as the files do not collide. Collisions are detected when the directory
// is merged, which is where the actual filenames are known — checking the
// directory string here would reject the common case for no reason.
func mergeSources(a, b core.MigrationSource) (core.MigrationSource, error) {
	if a.FS == nil {
		return b, nil
	}
	if b.FS == nil {
		return a, nil
	}
	return core.MigrationSource{
		Schema: a.Schema,
		FS:     multiFS{a.FS, b.FS},
		Dir:    ".",
	}, nil
}

// enabledModules turns config/modules.yaml into a lookup.
//
// Config drift is fail-fast: a module named in config but absent from the
// binary means the deployment is inconsistent, so the process refuses to run.
func enabledModules(cfg *config.Config, mods []core.Module) (map[string]bool, error) {
	present := make(map[string]bool, len(mods))
	out := make(map[string]bool, len(mods))
	for _, m := range mods {
		present[m.Name()] = true
		out[m.Name()] = true // default enabled
	}

	for name, state := range cfg.Modules {
		switch state {
		case "ENABLED", "DISABLED", "MAINTENANCE", "EXPERIMENTAL", "DEPRECATED":
		default:
			return nil, fmt.Errorf("platform: module %q has unknown state %q", name, state)
		}

		if !present[name] {
			if state == "DISABLED" {
				continue // naming an absent module as disabled is harmless
			}
			return nil, fmt.Errorf(
				"platform: config enables module %q but it is not compiled into this binary "+
					"(config drift; run corectl upgrade or fix config/modules.yaml)", name)
		}

		switch state {
		case "ENABLED", "DEPRECATED":
			out[name] = true
		case "EXPERIMENTAL":
			out[name] = cfg.Env != "production"
		default: // DISABLED, MAINTENANCE
			out[name] = false
		}
	}
	return out, nil
}
