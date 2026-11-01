// Package migrate runs numbered SQL migrations per schema.
//
// Guarantees (L0.md 4.5):
//   - each schema owns its own schema_migrations table
//   - each file runs in its own transaction; failure rolls back
//   - a changed file hash is an error, never a silent skip
//   - concurrent deploys are serialised with an advisory lock
package migrate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"

	"gorm.io/gorm"

	"core-platform/contract/core"
)

var schemaPattern = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

// Run applies every pending migration for every source, in order.
func Run(ctx context.Context, db *gorm.DB, sources []core.MigrationSource) error {
	for _, src := range sources {
		if err := RunSource(ctx, db, src); err != nil {
			return err
		}
	}
	return nil
}

// RunSource applies one source's migrations.
func RunSource(ctx context.Context, db *gorm.DB, src core.MigrationSource) error {
	if !schemaPattern.MatchString(src.Schema) {
		return fmt.Errorf("migrate: invalid schema %q", src.Schema)
	}

	if err := db.WithContext(ctx).Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %q", src.Schema)).Error; err != nil {
		return fmt.Errorf("migrate: create schema %s: %w", src.Schema, err)
	}

	// Serialise concurrent deploys. The lock is released when the session ends.
	if err := db.WithContext(ctx).Exec("SELECT pg_advisory_lock(hashtext(?))", src.Schema).Error; err != nil {
		return fmt.Errorf("migrate: advisory lock: %w", err)
	}
	defer func() {
		_ = db.Exec("SELECT pg_advisory_unlock(hashtext(?))", src.Schema).Error
	}()

	table := fmt.Sprintf("%s.schema_migrations", src.Schema)
	if err := db.WithContext(ctx).Exec(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		version    text PRIMARY KEY,
		checksum   text NOT NULL,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`, table)).Error; err != nil {
		return fmt.Errorf("migrate: create %s: %w", table, err)
	}

	applied := map[string]string{}
	var rows []struct {
		Version  string
		Checksum string
	}
	if err := db.WithContext(ctx).Raw(fmt.Sprintf(
		"SELECT version, checksum FROM %s", table)).Scan(&rows).Error; err != nil {
		return fmt.Errorf("migrate: read %s: %w", table, err)
	}
	for _, r := range rows {
		applied[r.Version] = r.Checksum
	}

	files, err := listFiles(src.FS, src.Dir)
	if err != nil {
		return err
	}

	for _, name := range files {
		body, err := fs.ReadFile(src.FS, name)
		if err != nil {
			return fmt.Errorf("migrate: read %s: %w", name, err)
		}
		sum := checksum(body)
		version := path.Base(name)

		if prev, ok := applied[version]; ok {
			if prev != sum {
				return fmt.Errorf(
					"migrate: %s/%s was modified after being applied (applied %s, now %s); "+
						"write a new migration instead",
					src.Schema, version, prev[:8], sum[:8])
			}
			continue
		}

		if err := applyOne(ctx, db, src.Schema, table, version, sum, string(body)); err != nil {
			return err
		}
	}
	return nil
}

func applyOne(ctx context.Context, db *gorm.DB, schema, table, version, sum, body string) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL search_path TO " + quoteIdent(schema)).Error; err != nil {
			return err
		}
		if strings.TrimSpace(body) != "" {
			if err := tx.Exec(body).Error; err != nil {
				return fmt.Errorf("migrate: %s/%s: %w", schema, version, err)
			}
		}
		return tx.Exec(
			fmt.Sprintf("INSERT INTO %s (version, checksum) VALUES (?, ?)", table),
			version, sum).Error
	})
}

func listFiles(fsys fs.FS, dir string) ([]string, error) {
	entries, err := fs.ReadDir(fsys, dirOrRoot(dir))
	if err != nil {
		return nil, fmt.Errorf("migrate: read dir %q: %w", dir, err)
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		out = append(out, path.Join(dir, e.Name()))
	}
	sort.Strings(out)
	return out, nil
}

func dirOrRoot(dir string) string {
	if dir == "" {
		return "."
	}
	return dir
}

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func checksum(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
