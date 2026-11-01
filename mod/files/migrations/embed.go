// Package migrations exposes the files schema as a core.MigrationSource.
package migrations

import (
	"embed"
	"io/fs"

	"core-platform/contract/core"
)

//go:embed *.sql
var files embed.FS

// Source is the files migration source.
//
// It shares the "public" schema with identity because both are core modules
// whose tables are queried without a prefix. A domain module would name its own
// schema instead.
func Source() core.MigrationSource {
	return core.MigrationSource{Schema: "public", FS: files, Dir: "."}
}

// FS exposes the embedded files for tests.
func FS() fs.FS { return files }
