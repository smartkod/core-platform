// Package migrations exposes the authorization schema as a core.MigrationSource.
//
// The ordinal 0004 continues the shared "public" schema history. Every module
// that writes into the same schema must pick an unused ordinal, which is why the
// number is chosen once and never reused.
package migrations

import (
	"embed"
	"io/fs"

	"core-platform/contract/core"
)

//go:embed *.sql
var files embed.FS

// Source is the authorization migration source.
func Source() core.MigrationSource {
	return core.MigrationSource{Schema: "public", FS: files, Dir: "."}
}

// FS exposes the embedded files for tests.
func FS() fs.FS { return files }
