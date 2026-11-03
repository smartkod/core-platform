// Command depcheck enforces the module dependency boundaries from PRD.md §8.4.
//
// It exists because the boundary is the whole point of the architecture: the
// kernel must never import a module, modules must reach each other only through
// contract/core, and the contract must stay free of implementation packages.
// A code review cannot guarantee that; a CI step can.
//
// The rules are expressed as data so they read as a statement of intent rather
// than as branching logic:
//
//	platform/**              must not import any module
//	mod/identity/**          must not import another module
//	mod/authz/**             must not import another module
//	mod/files/**             must not import another module
//	mod/notifications/**     must not import another module
//	mod/organization/**      must not import another module
//	mod/catalog/**           must not import another module
//	contract/**              must not import platform or modules
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

// rule is one policy entry.
type rule struct {
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

// coreModules are the modules that make up the core platform itself. None of
// them may import another module: they cooperate through contract/core only.
var coreModules = []string{
	modulePrefix + "/mod/identity",
	modulePrefix + "/mod/authz",
	modulePrefix + "/mod/files",
	modulePrefix + "/mod/notifications",
	modulePrefix + "/mod/organization",
	modulePrefix + "/mod/catalog",
}

// rules is the complete policy.
func rules() []rule {
	rs := []rule{
		{
			name:      "kernel must not import modules",
			prefixes:  []string{modulePrefix + "/platform"},
			forbidden: []string{modulePrefix + "/mod/"},
			reason:    "the kernel is the foundation; importing a module would invert the dependency direction",
		},
		{
			name:      "contract must stay dependency-free",
			prefixes:  []string{modulePrefix + "/contract"},
			forbidden: []string{modulePrefix + "/platform", modulePrefix + "/mod/"},
			reason:    "contract is the only package an application imports; it must not drag the implementation with it",
		},
	}

	// Every module may use its own subpackages, but must not reach into
	// another module. Listing the siblings explicitly (rather than banning all
	// of mod/) keeps that distinction checkable per module.
	for _, m := range coreModules {
		var siblings []string
		for _, other := range coreModules {
			if other != m {
				siblings = append(siblings, other)
			}
		}
		rs = append(rs, rule{
			name:      "modules must not import each other",
			prefixes:  []string{m},
			forbidden: siblings,
			reason:    "modules talk through contract/core, never by importing another module's packages",
		})
	}
	return rs
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
	for _, r := range rules() {
		for _, pkg := range pkgs {
			if pkg.Standard || !matchesAny(pkg.ImportPath, r.prefixes) {
				continue
			}
			for _, imp := range pkg.Imports {
				if !matchesAny(imp, r.forbidden) {
					continue
				}
				// A package importing itself by prefix is not a violation.
				if strings.HasPrefix(imp, pkg.ImportPath) {
					continue
				}
				violations = append(violations, fmt.Sprintf(
					"%s\n    imports %s\n    rule: %s\n    why:  %s",
					pkg.ImportPath, imp, r.name, r.reason))
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
		if path == p || strings.HasPrefix(path, p+"/") {
			return true
		}
	}
	return false
}
