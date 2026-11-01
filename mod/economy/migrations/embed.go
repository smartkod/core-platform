// Package migrations exposes the economy schema as a core.MigrationSource.
package migrations

import (
	"embed"
	"io/fs"

	"core-platform/contract/core"
)

//go:embed *.sql
var files embed.FS

// Source is the economy migration source.
//
// Core modules share the "public" schema because their tables are queried
// without a prefix; see mod/files for the same decision and rationale.
func Source() core.MigrationSource {
	return core.MigrationSource{Schema: "public", FS: files, Dir: "."}
}

// FS exposes the embedded files for tests.
func FS() fs.FS { return files }
