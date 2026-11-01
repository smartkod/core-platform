// Package database opens and verifies the PostgreSQL connection.
package database

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"core-platform/platform/config"
)

// Connect opens the pool and waits for the database to accept queries.
//
// Retry exists because in a compose stack the app can be scheduled before
// Postgres finishes booting even with a healthcheck, and because Docker
// Desktop's internal DNS can fail briefly while a network is being created.
func Connect(ctx context.Context, c *config.Config) (*gorm.DB, error) {
	// GORM's Info level prints every statement, which buries the application's
	// own logs. SQL tracing is therefore opt-in via LOG_LEVEL=debug rather than
	// implied by APP_ENV=development.
	lvl := gormlogger.Warn
	if c.Log.Level == "debug" {
		lvl = gormlogger.Info
	}

	var lastErr error
	for attempt := 0; attempt < 10; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		db, err := gorm.Open(postgres.Open(c.DSN()), &gorm.Config{
			Logger:                 gormlogger.Default.LogMode(lvl),
			SkipDefaultTransaction: true,
			NowFunc:                func() time.Time { return time.Now().UTC() },
		})
		if err == nil {
			sqlDB, err := db.DB()
			if err != nil {
				return nil, fmt.Errorf("database: pool: %w", err)
			}
			sqlDB.SetMaxOpenConns(c.DB.MaxOpenConns)
			sqlDB.SetMaxIdleConns(c.DB.MaxOpenConns / 2)
			sqlDB.SetConnMaxLifetime(time.Hour)

			pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			err = sqlDB.PingContext(pingCtx)
			cancel()
			if err == nil {
				return db, nil
			}
		}
		lastErr = err
		// Back off gently: the first few retries cover a container that is
		// still starting, the later ones cover a slow DNS or network setup.
		time.Sleep(time.Duration(attempt+1) * 300 * time.Millisecond)
	}
	return nil, fmt.Errorf(
		"database: cannot reach %s:%d after 10 attempts: %w\n"+
			"hint: check that the database is running and that DB_HOST resolves "+
			"(inside docker compose the host is the service name, e.g. \"postgres\")",
		c.DB.Host, c.DB.Port, lastErr)
}

// Ping reports database reachability for the readiness probe.
func Ping(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// Close releases the pool.
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
