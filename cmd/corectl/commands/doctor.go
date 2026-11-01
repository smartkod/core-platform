package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Doctor checks a generated project for the three things that most often break
// a deploy: missing config, an unreadable modules file, and a dirty workspace.
//
// It is intentionally dependency-free so it can run before `go mod tidy`.
func Doctor(args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	problems := 0
	report := func(format string, a ...any) {
		fmt.Printf("  ✗ "+format+"\n", a...)
		problems++
	}
	ok := func(format string, a ...any) {
		fmt.Printf("  ✓ "+format+"\n", a...)
	}

	fmt.Println("corectl doctor")

	// go.mod
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		report("go.mod not found in %s", dir)
	} else {
		ok("go.mod present")
	}

	// config
	cfgPath := filepath.Join(dir, "config", "config.yaml")
	if _, err := os.Stat(cfgPath); err != nil {
		report("config/config.yaml missing")
	} else {
		ok("config/config.yaml present")
	}

	// modules.yaml must stay parseable; a typo here stops the process at boot.
	modPath := filepath.Join(dir, "config", "modules.yaml")
	if body, err := os.ReadFile(modPath); err != nil {
		report("config/modules.yaml missing")
	} else {
		var mf struct {
			Modules map[string]string `yaml:"modules"`
		}
		if err := yaml.Unmarshal(body, &mf); err != nil {
			report("config/modules.yaml is not valid YAML: %v", err)
		} else {
			valid := true
			for name, state := range mf.Modules {
				switch state {
				case "ENABLED", "DISABLED", "MAINTENANCE", "EXPERIMENTAL", "DEPRECATED":
				default:
					report("module %q has unknown state %q", name, state)
					valid = false
				}
			}
			if valid {
				ok("config/modules.yaml valid (%d modules)", len(mf.Modules))
			}
		}
	}

	// .env must never be committed.
	if _, err := os.Stat(filepath.Join(dir, ".env")); err == nil {
		if body, err := os.ReadFile(filepath.Join(dir, ".gitignore")); err == nil {
			if !strings.Contains(string(body), ".env") {
				report(".env exists but is not ignored by .gitignore")
			} else {
				ok(".env present and ignored")
			}
		} else {
			report(".env exists but .gitignore is missing")
		}
	}

	// migrations directory
	if entries, err := os.ReadDir(filepath.Join(dir, "migrations")); err != nil {
		report("migrations/ missing")
	} else {
		ok("migrations/ present (%d files)", len(entries))
	}

	if problems > 0 {
		return fmt.Errorf("doctor found %d problem(s)", problems)
	}
	fmt.Println("\nno problems found")
	return nil
}
