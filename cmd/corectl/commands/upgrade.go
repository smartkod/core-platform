package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Upgrade moves a project to a newer core version.
//
// It refuses to continue when the core reports a breaking change, because a
// silent major upgrade is how eight applications end up on eight incompatible
// versions (PRD.md §8.4).
func Upgrade(args []string) error {
	dir := "."
	dryRun := false
	for _, a := range args {
		switch a {
		case "--dry-run":
			dryRun = true
		default:
			dir = a
		}
	}

	goMod := filepath.Join(dir, "go.mod")
	body, err := os.ReadFile(goMod)
	if err != nil {
		return fmt.Errorf("upgrade: read go.mod: %w", err)
	}

	corePath := coreModulePath()
	current := requiredVersion(string(body), corePath)
	if current == "" {
		return fmt.Errorf("upgrade: %s does not require %s", goMod, corePath)
	}

	root, err := coreRoot()
	if err != nil {
		return err
	}
	latest, err := latestCoreVersion(root)
	if err != nil {
		return fmt.Errorf("upgrade: cannot determine latest core version: %w", err)
	}

	if current == latest {
		fmt.Printf("already at %s v%s\n", corePath, current)
		return nil
	}

	major := func(v string) string {
		v = strings.TrimPrefix(v, "v")
		if i := strings.IndexByte(v, '.'); i > 0 {
			return v[:i]
		}
		return v
	}
	if major(current) != major(latest) {
		return fmt.Errorf(
			"upgrade: %s v%s -> v%s is a MAJOR change and is not applied automatically; "+
				"read MIGRATION.md in the core and upgrade by hand",
			corePath, current, latest)
	}

	fmt.Printf("%s v%s -> v%s\n", corePath, current, latest)
	if dryRun {
		fmt.Println("dry run: no changes written")
		return nil
	}

	cmd := exec.Command("go", "get", corePath+"@v"+latest)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("upgrade: go get: %w", err)
	}

	fmt.Println("\nnext: run tests, then `corectl doctor`")
	return nil
}

// requiredVersion extracts the required version of corePath from go.mod text.
func requiredVersion(goMod, corePath string) string {
	for _, line := range strings.Split(goMod, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, corePath+" ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				return strings.TrimPrefix(fields[1], "v")
			}
		}
	}
	return ""
}

// latestCoreVersion reads the version from the core's CHANGELOG or VERSION file.
func latestCoreVersion(root string) (string, error) {
	if body, err := os.ReadFile(filepath.Join(root, "VERSION")); err == nil {
		return strings.TrimSpace(string(body)), nil
	}
	if body, err := os.ReadFile(filepath.Join(root, "CHANGELOG.md")); err == nil {
		for _, line := range strings.Split(string(body), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "## [") {
				rest := strings.TrimPrefix(line, "## [")
				if i := strings.IndexByte(rest, ']'); i > 0 {
					return rest[:i], nil
				}
			}
		}
	}
	return "", fmt.Errorf("no VERSION or CHANGELOG.md entry found in %s", root)
}
