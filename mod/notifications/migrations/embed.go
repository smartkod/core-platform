// Package migrations exposes the notifications schema as a core.MigrationSource.
//
// The ordinal 0003 is unique across every module sharing the "public" schema, so
// the merged history applies identity, files and notifications in a defined
// order.
package migrations

import (
	"embed"
	"io/fs"

	"core-platform/contract/core"
)

//go:embed *.sql
var files embed.FS

// Source is the notifications migration source.
func Source() core.MigrationSource {
	return core.MigrationSource{Schema: "public", FS: files, Dir: "."}
}

// FS exposes the embedded files for tests.
func FS() fs.FS { return files }
