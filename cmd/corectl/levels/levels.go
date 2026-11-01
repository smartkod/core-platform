// Package levels describes what each platform level contains.
//
// This is the single source of truth used by the scaffolder, the docs generator
// and the CI "demo builds" gate. Adding a module to a level means editing one
// table here.
package levels

import "fmt"

// Level is a platform tier an application can start at.
type Level string

const (
	L0 Level = "L0"
	L1 Level = "L1"
	L2 Level = "L2"
	L3 Level = "L3"
)

// Module describes one optional module a level can include.
type Module struct {
	Name string
	// Level is the level that introduces this module.
	Level Level
	// Requires lists other module names.
	Requires []string
}

// available lists every module the scaffold knows about, in start order.
//
// Core (L0) modules are always present and are not listed here.
var available = []Module{
	{Name: "identity", Level: L1},
	{Name: "files", Level: L1, Requires: []string{"identity"}},
	{Name: "notifications", Level: L1, Requires: []string{"identity"}},
	{Name: "economy", Level: L2, Requires: []string{"identity"}},
	{Name: "ai", Level: L3, Requires: []string{"identity"}},
}

// Parse validates a level string.
func Parse(s string) (Level, error) {
	switch Level(s) {
	case L0, L1, L2, L3:
		return Level(s), nil
	}
	return "", fmt.Errorf("unknown level %q (want L0, L1, L2 or L3)", s)
}

// rank orders levels so "does this level include that one" is answerable.
func rank(l Level) int {
	switch l {
	case L0:
		return 0
	case L1:
		return 1
	case L2, L3:
		return 2
	}
	return -1
}

// ModulesFor returns the modules a level includes, filtered by only when it is
// non-empty.
//
// L2 and L3 are siblings: both start from L1, so choosing L2 does not include
// AI and choosing L3 does not include the economy.
func ModulesFor(l Level, only []string) ([]Module, error) {
	if l == L0 {
		if len(only) > 0 {
			return nil, fmt.Errorf("--only has no effect at L0: the kernel cannot be split")
		}
		return nil, nil
	}

	// Sibling levels contribute their own module on top of L1.
	base := l
	extra := Level("")
	switch l {
	case L2:
		base, extra = L1, L2
	case L3:
		base, extra = L1, L3
	}

	var out []Module
	for _, m := range available {
		if rank(m.Level) <= rank(base) || (extra != "" && m.Level == extra) {
			out = append(out, m)
		}
	}

	if len(only) == 0 {
		return withDependencies(out), nil
	}

	// --only narrows the set but never drops a dependency.
	return selectOnly(out, only)
}

func selectOnly(all []Module, only []string) ([]Module, error) {
	want := map[string]bool{}
	for _, name := range only {
		want[name] = true
	}

	byName := map[string]Module{}
	for _, m := range all {
		byName[m.Name] = m
	}

	for name := range want {
		if _, ok := byName[name]; !ok {
			return nil, fmt.Errorf("module %q is not part of this level", name)
		}
	}

	// Pull in dependencies transitively.
	changed := true
	for changed {
		changed = false
		for name := range want {
			for _, dep := range byName[name].Requires {
				if !want[dep] {
					want[dep] = true
					changed = true
				}
			}
		}
	}

	var out []Module
	for _, m := range all {
		if want[m.Name] {
			out = append(out, m)
		}
	}
	return withDependencies(out), nil
}

// withDependencies guarantees that every named dependency exists in the set.
func withDependencies(mods []Module) []Module {
	present := map[string]bool{}
	for _, m := range mods {
		present[m.Name] = true
	}

	var missing []string
	for _, m := range mods {
		for _, dep := range m.Requires {
			if !present[dep] {
				missing = append(missing, fmt.Sprintf("%s requires %s", m.Name, dep))
			}
		}
	}
	if len(missing) == 0 {
		return mods
	}

	// The caller sees a consistent set: fill in what is missing.
	for _, m := range available {
		if !present[m.Name] && needsFill(m.Name, mods) {
			mods = append(mods, m)
			present[m.Name] = true
		}
	}
	return mods
}

func needsFill(name string, mods []Module) bool {
	for _, m := range mods {
		for _, dep := range m.Requires {
			if dep == name {
				return true
			}
		}
	}
	return false
}

// Names returns just the module names, in order.
func Names(mods []Module) []string {
	out := make([]string, 0, len(mods))
	for _, m := range mods {
		out = append(out, m.Name)
	}
	return out
}
