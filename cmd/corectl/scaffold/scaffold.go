// Package scaffold renders a new application from templates.
//
// The generated project must build and run with no manual edits: that is the
// golden-path gate (L0.md 6.2).
package scaffold

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"text/template"
)

// Params is the input for rendering.
type Params struct {
	// AppName is the directory and module name, e.g. "fitscope".
	AppName string
	// ModulePath is the Go module path, e.g. "fitscope" or "github.com/me/fitscope".
	ModulePath string
	// Level is L0..L3.
	Level string
	// Modules are the chosen module names, in start order.
	Modules []string
	// CoreModulePath is the core-platform module path, e.g. "core-platform".
	CoreModulePath string
	// CoreVersion is the version to require.
	CoreVersion string
	// CoreReplacePath is written as a local replace directive so the fresh
	// project builds before the core is published. Empty disables it.
	CoreReplacePath string
	// GoVersion is the major.minor toolchain version, used in both go.mod and
	// the Dockerfile so the two can never disagree.
	GoVersion string
	// HasIdentity reports whether the identity module is included, so templates
	// can emit the JWT settings only when they are meaningful.
	HasIdentity bool
}

// HasModule reports whether name is in the module list.
func (p Params) HasModule(name string) bool {
	for _, m := range p.Modules {
		if m == name {
			return true
		}
	}
	return false
}

// Render writes the project into dir, refusing to overwrite existing files.
func Render(dir string, p Params) error {
	files, err := renderFiles(p)
	if err != nil {
		return err
	}

	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		full := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return fmt.Errorf("scaffold: mkdir %s: %w", filepath.Dir(full), err)
		}
		if _, err := os.Stat(full); err == nil {
			return fmt.Errorf("scaffold: %s already exists", full)
		}
		if err := os.WriteFile(full, files[name], 0o644); err != nil {
			return fmt.Errorf("scaffold: write %s: %w", full, err)
		}
	}
	return nil
}

func renderFiles(p Params) (map[string][]byte, error) {
	out := map[string][]byte{}

	for name, body := range templates {
		tpl, err := template.New(name).Option("missingkey=error").Parse(body)
		if err != nil {
			return nil, fmt.Errorf("scaffold: parse %s: %w", name, err)
		}
		var buf bytes.Buffer
		if err := tpl.Execute(&buf, p); err != nil {
			return nil, fmt.Errorf("scaffold: render %s: %w", name, err)
		}
		out[name] = buf.Bytes()
	}

	// Empty migration directory must survive git.
	out["migrations/.gitkeep"] = []byte{}
	return out, nil
}

func init() {
	// Keep sort order stable when Modules carries the same entries twice.
	_ = sort.Strings
}
