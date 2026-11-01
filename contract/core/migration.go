package core

import "io/fs"

// MigrationSource describes one schema's SQL migrations.
//
// Each source owns its own schema and its own schema_migrations table, so the
// core schema and an application schema never share history (L0.md 4.5).
type MigrationSource struct {
	// Schema is the PostgreSQL schema name. It must match ^[a-z_][a-z0-9_]*$
	// and is created if missing.
	Schema string
	// FS holds the migration files.
	FS fs.FS
	// Dir is the subdirectory inside FS. Empty means the root.
	Dir string
}

// MigrationProvider is implemented by modules that own database schema.
//
// The bootstrap collects these automatically, so a module never forgets to
// register its migrations and never depends on the application listing them.
type MigrationProvider interface {
	MigrationSource() MigrationSource
}
