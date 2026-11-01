package core

// App is what an application passes to platform.Run.
//
// The generated main.go contains only this value and a call to Run, keeping
// bootstrap logic out of application code (L0.md 2.2).
type App struct {
	// Name identifies the application in logs, /version and DB defaults.
	Name string
	// Version is stamped at build time via ldflags.
	Version string
	// Modules are wired explicitly. This is the only place a level is chosen:
	// an L1 application lists L0 + L1 modules and nothing else, so the Go
	// compiler leaves the rest out of the binary entirely.
	Modules []Module
	// Migrations are applied before modules start.
	Migrations []MigrationSource
	// ConfigPath points at the optional YAML file. Empty means env only.
	ConfigPath string
	// BaseDir is the working directory used to resolve relative paths.
	BaseDir string
}
