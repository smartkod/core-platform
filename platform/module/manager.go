// Package module manages the module lifecycle.
//
// Ordering is a topological sort (Kahn). An unknown dependency is fail-closed:
// the process refuses to start rather than silently skipping a module.
package module

import (
	"context"
	"fmt"
	"sort"

	"core-platform/contract/core"
)

// Status is the lifecycle state of a module.
type Status string

const (
	StatusRegistered Status = "registered"
	StatusMigrated   Status = "migrated"
	StatusStarted    Status = "started"
	StatusFailed     Status = "failed"
	StatusStopped    Status = "stopped"
)

// Info is a read-only snapshot for admin endpoints and doctor output.
type Info struct {
	Name         string
	Version      string
	Dependencies []string
	Status       Status
}

// Manager owns the registered modules and their start/stop order.
type Manager struct {
	order   []core.Module
	byName  map[string]core.Module
	status  map[string]Status
	enabled map[string]bool // nil means "all enabled"
}

// New builds a manager and validates the dependency graph up front.
//
// enabled maps module name -> whether it should run. A nil map enables all.
func New(mods []core.Module, enabled map[string]bool) (*Manager, error) {
	m := &Manager{
		byName:  make(map[string]core.Module, len(mods)),
		status:  make(map[string]Status, len(mods)),
		enabled: enabled,
	}

	for _, mod := range mods {
		name := mod.Name()
		if name == "" {
			return nil, fmt.Errorf("module: empty name")
		}
		if _, dup := m.byName[name]; dup {
			return nil, fmt.Errorf("module: duplicate name %q", name)
		}
		m.byName[name] = mod
		m.status[name] = StatusRegistered
	}

	order, err := topoSort(mods, m.byName)
	if err != nil {
		return nil, err
	}
	m.order = order
	return m, nil
}

// Order returns the start order (names only).
func (m *Manager) Order() []string {
	out := make([]string, 0, len(m.order))
	for _, mod := range m.order {
		out = append(out, mod.Name())
	}
	return out
}

// Infos returns every module's snapshot in start order.
func (m *Manager) Infos() []Info {
	out := make([]Info, 0, len(m.order))
	for _, mod := range m.order {
		out = append(out, Info{
			Name:         mod.Name(),
			Version:      mod.Version(),
			Dependencies: mod.Dependencies(),
			Status:       m.status[mod.Name()],
		})
	}
	return out
}

// IsEnabled reports whether a module will run.
func (m *Manager) IsEnabled(name string) bool {
	if m.enabled == nil {
		return true
	}
	v, ok := m.enabled[name]
	return ok && v
}

// Migrate runs every enabled module's migration in start order.
func (m *Manager) Migrate(ctx context.Context, deps core.Deps) error {
	for _, mod := range m.order {
		if !m.IsEnabled(mod.Name()) {
			continue
		}
		if err := mod.Migrate(ctx, deps.DB); err != nil {
			m.status[mod.Name()] = StatusFailed
			return fmt.Errorf("module %s: migrate: %w", mod.Name(), err)
		}
		m.status[mod.Name()] = StatusMigrated
	}
	return nil
}

// Start launches every enabled module, rolling back in reverse order on error.
func (m *Manager) Start(ctx context.Context, deps core.Deps) error {
	started := make([]core.Module, 0, len(m.order))

	for _, mod := range m.order {
		if !m.IsEnabled(mod.Name()) {
			continue
		}
		if err := mod.Start(ctx, deps); err != nil {
			m.status[mod.Name()] = StatusFailed
			m.stopReverse(ctx, started)
			return fmt.Errorf("module %s: start: %w", mod.Name(), err)
		}
		m.status[mod.Name()] = StatusStarted
		started = append(started, mod)
	}
	return nil
}

// Stop shuts every started module down in reverse order.
func (m *Manager) Stop(ctx context.Context) {
	started := make([]core.Module, 0, len(m.order))
	for _, mod := range m.order {
		if m.status[mod.Name()] == StatusStarted {
			started = append(started, mod)
		}
	}
	m.stopReverse(ctx, started)
}

func (m *Manager) stopReverse(ctx context.Context, mods []core.Module) {
	for i := len(mods) - 1; i >= 0; i-- {
		mod := mods[i]
		_ = mod.Stop(ctx)
		m.status[mod.Name()] = StatusStopped
	}
}

// topoSort is Kahn's algorithm with cycle detection.
func topoSort(mods []core.Module, byName map[string]core.Module) ([]core.Module, error) {
	indegree := make(map[string]int, len(mods))
	dependents := make(map[string][]string, len(mods))

	for _, mod := range mods {
		name := mod.Name()
		if _, ok := indegree[name]; !ok {
			indegree[name] = 0
		}
		for _, dep := range mod.Dependencies() {
			if _, known := byName[dep]; !known {
				return nil, fmt.Errorf(
					"module %q depends on unknown module %q (fail-closed)", name, dep)
			}
			indegree[name]++
			dependents[dep] = append(dependents[dep], name)
		}
	}

	queue := make([]string, 0, len(mods))
	for name, deg := range indegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}
	sort.Strings(queue) // deterministic order for identical graphs

	out := make([]core.Module, 0, len(mods))
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		out = append(out, byName[name])

		next := dependents[name]
		sort.Strings(next)
		for _, dep := range next {
			indegree[dep]--
			if indegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	if len(out) != len(mods) {
		var stuck []string
		for name, deg := range indegree {
			if deg > 0 {
				stuck = append(stuck, name)
			}
		}
		sort.Strings(stuck)
		return nil, fmt.Errorf("module: dependency cycle among %v", stuck)
	}
	return out, nil
}
