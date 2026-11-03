package migrations_test

import (
	"context"
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"core-platform/contract/core"
	authzmigrations "core-platform/mod/authz/migrations"
	catalogmigrations "core-platform/mod/catalog/migrations"
	filesmigrations "core-platform/mod/files/migrations"
	identitymigrations "core-platform/mod/identity/migrations"
	notificationsmigrations "core-platform/mod/notifications/migrations"
	organizationmigrations "core-platform/mod/organization/migrations"
	"core-platform/platform/migrate"
)

// TestAllCoreMigrationsApply applies every core module migration to one empty
// schema in module order. It only runs when TEST_DATABASE_DSN is set.
func TestAllCoreMigrationsApply(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	sources := []core.MigrationSource{
		identitymigrations.Source(),
		authzmigrations.Source(),
		filesmigrations.Source(),
		notificationsmigrations.Source(),
		organizationmigrations.Source(),
		catalogmigrations.Source(),
	}
	// Apply one by one so a failure names the module that broke.
	for _, src := range sources {
		if err := migrate.RunSource(ctx, db, src); err != nil {
			t.Fatalf("migration %s: %v", src.Schema, err)
		}
	}

	// A second pass must be a no-op: this is what a redeploy looks like.
	for _, src := range sources {
		if err := migrate.RunSource(ctx, db, src); err != nil {
			t.Fatalf("re-run %s: %v", src.Schema, err)
		}
	}

	var roles int64
	if err := db.Raw("SELECT count(*) FROM roles").Scan(&roles).Error; err != nil {
		t.Fatal(err)
	}
	if roles < 6 {
		t.Fatalf("seeded roles = %d, want at least 6", roles)
	}
}
