package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"core-platform/cmd/corectl/levels"
	"core-platform/cmd/corectl/scaffold"
)

// New implements `corectl new <app> --level=LN [--only=...] [--dir=...]`.
//
// Arguments are parsed by hand rather than with the flag package, because Go's
// flag parsing stops at the first non-flag argument and `corectl new app
// --level=L1` must not silently scaffold an L0 project.
func New(args []string) error {
	var (
		appName    string
		levelStr   = "L0"
		onlyStr    string
		target     string
		modulePath string
	)

	for _, a := range args {
		switch {
		case strings.HasPrefix(a, "--level="):
			levelStr = strings.TrimPrefix(a, "--level=")
		case strings.HasPrefix(a, "--only="):
			onlyStr = strings.TrimPrefix(a, "--only=")
		case strings.HasPrefix(a, "--dir="):
			target = strings.TrimPrefix(a, "--dir=")
		case strings.HasPrefix(a, "--module="):
			modulePath = strings.TrimPrefix(a, "--module=")
		case a == "--level" || a == "--only" || a == "--dir" || a == "--module":
			return fmt.Errorf("new: %s requires a value, e.g. %s=...", a, a)
		case strings.HasPrefix(a, "-"):
			return fmt.Errorf("new: unknown flag %q", a)
		default:
			if appName != "" {
				return fmt.Errorf("new: unexpected argument %q", a)
			}
			appName = a
		}
	}

	if appName == "" {
		return fmt.Errorf("new: application name is required")
	}
	if !validName(appName) {
		return fmt.Errorf("new: invalid application name %q (lowercase letters, digits, dash)", appName)
	}

	level, err := levels.Parse(levelStr)
	if err != nil {
		return err
	}

	var only []string
	if v := strings.TrimSpace(onlyStr); v != "" {
		for _, p := range strings.Split(v, ",") {
			if p = strings.TrimSpace(p); p != "" {
				only = append(only, p)
			}
		}
	}

	mods, err := levels.ModulesFor(level, only)
	if err != nil {
		return err
	}

	root, err := coreRoot()
	if err != nil {
		return err
	}
	if missing := missingModules(root, mods); len(missing) > 0 {
		return fatalMissing(missing, string(level))
	}

	if target == "" {
		target = "./" + appName
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	if entries, err := os.ReadDir(abs); err == nil && len(entries) > 0 {
		return fmt.Errorf("new: %s already exists and is not empty", abs)
	}

	if modulePath == "" {
		modulePath = appName
	}

	if err := scaffold.Render(abs, scaffold.Params{
		AppName:         appName,
		ModulePath:      modulePath,
		Level:           string(level),
		Modules:         levels.Names(mods),
		CoreModulePath:  coreModulePath(),
		CoreVersion:     "0.1.0",
		CoreReplacePath: filepath.ToSlash(root),
		GoVersion:       goMinorVersion(),
		HasIdentity:     hasModule(levels.Names(mods), "identity"),
	}); err != nil {
		return err
	}

	fmt.Printf("created %s (level %s)\n", abs, level)
	if len(mods) > 0 {
		fmt.Printf("modules: %s\n", strings.Join(levels.Names(mods), ", "))
	} else {
		fmt.Println("modules: none (kernel only)")
	}

	// Resolve dependencies and vendor them so the Docker build needs no
	// network and no access to the core checkout (see the Dockerfile header).
	// Failing here is not fatal: the project is already valid and the developer
	// may be offline.
	if err := tidyAndVendor(abs); err != nil {
		fmt.Printf("\nwarning: could not prepare vendor/ (%v)\n"+
			"run 'go mod tidy && go mod vendor' before 'docker compose up'\n", err)
	}

	fmt.Printf("\nnext:\n  cd %s\n  docker compose up\n", filepath.ToSlash(target))
	return nil
}

// tidyAndVendor runs the two Go commands the container build depends on.
func tidyAndVendor(dir string) error {
	for _, args := range [][]string{{"mod", "tidy"}, {"mod", "vendor"}} {
		cmd := exec.Command("go", args...)
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("go %s: %w", strings.Join(args, " "), err)
		}
	}
	return nil
}

// hasModule reports whether name is in mods.
func hasModule(mods []string, name string) bool {
	for _, s := range mods {
		if s == name {
			return true
		}
	}
	return false
}

// goMinorVersion returns the running toolchain as major.minor (e.g. "1.26").
//
// It is used for both the go directive and the Docker image tag so a project
// generated here builds the same way on the developer machine and in CI.
func goMinorVersion() string {
	v := runtime.Version() // go1.26.5
	v = strings.TrimPrefix(v, "go")
	parts := strings.Split(v, ".")
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return "1.22"
}

func validName(s string) bool {
	if s == "" || len(s) > 40 {
		return false
	}
	for i, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}
