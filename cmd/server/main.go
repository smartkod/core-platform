// Command server is the reference binary for the platform itself.
//
// It runs the full core (kernel + identity + organization + catalog) so every
// capability can be exercised end to end without an application on top. An
// application's own main.go is generated from the same shape by corectl.
//
// Note what is absent: no booking, order, quote, lesson or payment module. The
// core describes who a business is and what it publishes; it never runs a
// sector transaction (PRD.md §3).
package main

import (
	"log"

	"core-platform/contract/core"
	"core-platform/mod/authz"
	"core-platform/mod/catalog"
	"core-platform/mod/files"
	"core-platform/mod/identity"
	"core-platform/mod/notifications"
	"core-platform/mod/organization"
	"core-platform/platform"
)

// version is stamped at build time: -ldflags "-X main.version=v0.1.0".
var version = "0.1.0-dev"

func main() {
	if err := platform.Run(core.App{
		Name:    "core-platform",
		Version: version,
		// The level is chosen here and nowhere else. Removing a line removes
		// the module from the binary entirely.
		Modules: []core.Module{
			identity.New(),
			authz.New(),
			files.New(),
			notifications.New(),
			organization.New(),
			catalog.New(),
		},
		ConfigPath: "config/config.yaml",
	}); err != nil {
		log.Fatal(err)
	}
}
