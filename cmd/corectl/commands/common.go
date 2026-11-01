// Package commands implements the corectl command set.
package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"core-platform/cmd/corectl/levels"
)

// coreModulePath is resolved at runtime so a fork of the core keeps working.
func coreModulePath() string {
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Path != "" {
		return bi.Main.Path
	}
	return "core-platform"
}

// coreRoot locates the core checkout on disk.
//
// This matters because the generated go.mod carries a local `replace`
// directive so a brand-new project builds before the core is published. A
// wrong path here would silently point the app at the wrong source, so an
// unresolved root is reported instead of guessed.
func coreRoot() (string, error) {
	if v := os.Getenv("CORE_PLATFORM_ROOT"); v != "" {
		abs, err := filepath.Abs(v)
		if err != nil {
			return "", err
		}
		if isCoreRoot(abs) {
			return abs, nil
		}
		return "", fmt.Errorf("CORE_PLATFORM_ROOT=%s is not a core checkout", v)
	}

	candidates := []string{}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, wd)
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Dir(exe))
	}

	for _, start := range candidates {
		dir := start
		for {
			if isCoreRoot(dir) {
				return dir, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	return "", fmt.Errorf(
		"cannot locate the core-platform checkout; run corectl from the core repository " +
			"or set CORE_PLATFORM_ROOT")
}

func isCoreRoot(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		return false
	}
	// platform/run.go only exists in the core, never in an application.
	_, err := os.Stat(filepath.Join(dir, "platform", "run.go"))
	return err == nil
}

// missingModules reports which requested modules do not exist in this core
// build. Scaffolding them anyway would produce a project that cannot compile,
// so the command refuses instead of generating a broken skeleton.
func missingModules(root string, mods []levels.Module) []string {
	var missing []string
	for _, m := range mods {
		if _, err := os.Stat(filepath.Join(root, "mod", m.Name)); err != nil {
			missing = append(missing, m.Name)
		}
	}
	return missing
}

func fatalMissing(missing []string, level string) error {
	return fmt.Errorf(
		"level %s is not available in this core version yet: module(s) %s are missing from mod/. "+
			"Use --level=L0 for now; generating the rest would produce a project that does not compile",
		level, strings.Join(missing, ", "))
}
