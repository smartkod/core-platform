// Command server is the reference binary for the platform itself.
//
// It runs at L2 (kernel + identity + files + notifications + economy) so the
// engine can be exercised end to end without an application on top. An
// application's own main.go is generated from the same shape by corectl.
package main

import (
	"log"

	"core-platform/contract/core"
	"core-platform/mod/economy"
	"core-platform/mod/files"
	"core-platform/mod/identity"
	"core-platform/mod/notifications"
	"core-platform/platform"
)

// version is stamped at build time: -ldflags "-X main.version=v0.1.0".
var version = "0.1.0-dev"

func main() {
	if err := platform.Run(core.App{
		Name:    "core-platform",
		Version: version,
		// The level is chosen here and nowhere else. Removing a line removes
		// the module from the binary entirely (PRD.md 3.3).
		Modules: []core.Module{
			identity.New(),
			files.New(),
			notifications.New(),
			economy.New(),
		},
		ConfigPath: "config/config.yaml",
	}); err != nil {
		log.Fatal(err)
	}
}
