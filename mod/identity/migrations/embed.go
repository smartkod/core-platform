// Package migrations exposes the identity schema as a core.MigrationSource.
//
// The SQL files are embedded so the module carries its own schema; nothing in
// platform/ knows what tables identity owns.
//
// File names carry an ordinal that is unique across every module sharing the
// "public" schema: identity owns 0001, files owns 0002. Core modules share that
// schema because their tables are queried without a prefix, so the ordinals form
// one ordered history. Adding a module means taking the next free number — the
// merger rejects duplicates rather than silently skipping one.
package migrations

import (
	"embed"
	"io/fs"

	"core-platform/contract/core"
)

//go:embed *.sql
var files embed.FS

// Source is the identity migration source.
//
// Core modules own the default schema: their tables are queried without a
// prefix, and every application expects to find `users` in the same place. A
// domain module that wants isolation names its own schema and sets it as the
// connection's search_path — the migration runner handles the bookkeeping
// either way.
func Source() core.MigrationSource {
	return core.MigrationSource{Schema: "public", FS: files, Dir: "."}
}

// FS exposes the embedded files for tests.
func FS() fs.FS { return files }
