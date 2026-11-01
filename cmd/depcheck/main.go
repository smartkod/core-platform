// Command depcheck enforces the module dependency boundaries from PRD.md 4.1.
//
// It exists because the boundary is the whole point of the architecture: an L1
// application must not be able to import the L2 or L3 modules, and the kernel
// must never import a module. A code review cannot guarantee that; a CI step can.
//
// The rules are expressed as data so they read as a statement of intent rather
// than as branching logic:
//
//	platform/**            must not import any module
//	mod/identity/**        must not import a higher-level module
//	mod/files/**           must not import a higher-level module
//	mod/notifications/**   must not import a higher-level module
//	contract/**            must not import platform or modules
//
// Usage:
//
//	go run ./cmd/depcheck          # checks the whole repository
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
)

// handler is one rule.
type handler struct {
	// name is what the rule protects, used in the failure message.
	name string
	// prefixes are the package prefixes the rule applies to.
	prefixes []string
	// forbidden are package prefixes that must not appear among their imports.
	forbidden []string
	// reason explains why the rule exists.
	reason string
}

// modulePrefix is the module path of this repository.
const modulePrefix = "core-platform"

// rules is the complete policy.
func rules() []handler {
	l1Modules := []string{
		modulePrefix + "/mod/files",
		modulePrefix + "/mod/identity",
		modulePrefix + "/mod/notifications",
	}
	higherLevel := []string{
		modulePrefix + "/mod/economy",
		modulePrefix + "/mod/ai",
	}

	return []handler{
		{
			name:      "kernel must not import modules",
			prefixes:  []string{modulePrefix + "/platform"},
			forbidden: []string{modulePrefix + "/mod/"},
			reason:    "the kernel is the foundation; importing a module would invert the dependency direction",
		},
		{
			name:      "L1 modules must not import L2 or L3 modules",
			prefixes:  l1Modules,
			forbidden: higherLevel,
			reason:    "an L1-only application must build without economy or ai; this is what makes level reuse possible",
		},
		{
			name:      "L1 modules must not import each other's internals",
			prefixes:  []string{modulePrefix + "/mod/files", modulePrefix + "/mod/notifications"},
			forbidden: []string{modulePrefix + "/mod/identity/"},
			reason:    "modules talk through contract/core, never by importing another module's packages",
		},
		{
			name:      "contract must stay dependency-free",
			prefixes:  []string{modulePrefix + "/contract"},
			forbidden: []string{modulePrefix + "/platform", modulePrefix + "/mod/"},
			reason:    "contract is the only package an application imports; it must not drag the implementation with it",
		},
	}
}

// pkgImports is one entry of `go list -json` output.
type pkgImports struct {
	ImportPath string
	Imports    []string
	Deps       []string
	Standard   bool
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "depcheck: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("depcheck: all boundaries hold")
}

func run() error {
	pkgs, err := listPackages()
	if err != nil {
		return err
	}

	var violations []string
	for _, rule := range rules() {
		for _, pkg := range pkgs {
			if pkg.Standard || !matchesAny(pkg.ImportPath, rule.prefixes) {
				continue
			}
			for _, imp := range pkg.Imports {
				if !matchesAny(imp, rule.forbidden) {
					continue
				}
				// A package importing itself by prefix is not a violation; this
				// only matters for a rule whose prefixes overlap its forbidden
				// list, which none do today, but the check keeps it honest.
				if strings.HasPrefix(imp, pkg.ImportPath) {
					continue
				}
				violations = append(violations, fmt.Sprintf(
					"%s\n    imports %s\n    rule: %s\n    why:  %s",
					pkg.ImportPath, imp, rule.name, rule.reason))
			}
		}
	}

	if len(violations) > 0 {
		sort.Strings(violations)
		return fmt.Errorf("dependency boundary violated:\n\n  %s",
			strings.Join(violations, "\n\n  "))
	}
	return nil
}

// listPackages runs `go list` and decodes the stream of JSON objects.
func listPackages() ([]pkgImports, error) {
	cmd := exec.Command("go", "list", "-e", "-json", "./...")
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list: %w", err)
	}

	var pkgs []pkgImports
	dec := json.NewDecoder(strings.NewReader(string(out)))
	for dec.More() {
		var p pkgImports
		if err := dec.Decode(&p); err != nil {
			return nil, fmt.Errorf("decode go list output: %w", err)
		}
		pkgs = append(pkgs, p)
	}
	return pkgs, nil
}

func matchesAny(path string, prefixes []string) bool {
	for _, p := range prefixes {
		if path == p || strings.HasPrefix(path, p+"/") || strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}
