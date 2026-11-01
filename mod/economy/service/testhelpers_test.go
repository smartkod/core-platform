package service_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"core-platform/mod/economy/migrations"
	"core-platform/mod/economy/repository"
	"core-platform/mod/economy/service"
	"core-platform/platform/migrate"
)

// fakeClock is a controllable clock so reward timing is testable without
// sleeping.
type fakeClock struct{ t time.Time }

func (c *fakeClock) Now() time.Time          { return c.t }
func (c *fakeClock) Advance(d time.Duration) { c.t = c.t.Add(d) }

func newClock() *fakeClock {
	return &fakeClock{t: time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)}
}

// economyDB opens TEST_DATABASE_DSN and applies the economy schema. The test
// skips when no DSN exists rather than passing silently, mirroring the
// identity integration tests.
func economyDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set; skipping database-backed economy tests")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                 gormlogger.Default.LogMode(gormlogger.Silent),
		SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := migrate.RunSource(context.Background(), db, migrations.Source()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// freshWallet builds a wallet isolated to a new user id (no truncation: other
// package binaries share the database).
func freshWallet(t *testing.T, db *gorm.DB, clock *fakeClock) (*service.Wallet, string) {
	t.Helper()
	w := service.NewWallet(repository.NewStore(db), nil, clock)
	return w, uuid.NewString()
}

// setDial writes known dials so a test does not inherit a previous run's
// singleton row. The admin id is a fixed non-null uuid.
func setDial(t *testing.T, db *gorm.DB, signup, watch, daily, monthly int64, click bool) {
	t.Helper()
	err := db.Exec(
		`INSERT INTO reward_settings (id, signup_grant, ad_watch_reward, daily_reward_cap,
		     monthly_reward_cap, click_reward_enabled, updated_at, updated_by)
		 VALUES (1, ?, ?, ?, ?, ?, now(), ?)
		 ON CONFLICT (id) DO UPDATE SET
		     signup_grant = EXCLUDED.signup_grant,
		     ad_watch_reward = EXCLUDED.ad_watch_reward,
		     daily_reward_cap = EXCLUDED.daily_reward_cap,
		     monthly_reward_cap = EXCLUDED.monthly_reward_cap,
		     click_reward_enabled = EXCLUDED.click_reward_enabled`,
		signup, watch, daily, monthly, click, uuid.NewString(),
	).Error
	if err != nil {
		t.Fatalf("set dial: %v", err)
	}
}
