package platform

import (
	"fmt"
	"io/fs"
	"sort"
)

// multiFS merges several embedded filesystems into one view.
//
// It exists because core modules share the default schema but each ships its own
// embedded SQL directory. The migration runner sees a single source per schema,
// so schema_migrations stays one consistent history, while the files still come
// from the module that owns them.
//
// Overlapping root files are a hard error: two files with the same name in the
// same schema would be ordered arbitrarily and one would never apply.
type multiFS []fs.FS

func (m multiFS) Open(name string) (fs.File, error) {
	for _, f := range m {
		file, err := f.Open(name)
		if err == nil {
			return file, nil
		}
	}
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}

func (m multiFS) ReadDir(name string) ([]fs.DirEntry, error) {
	seen := map[string]fs.DirEntry{}
	for _, f := range m {
		entries, err := fs.ReadDir(f, name)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if prev, ok := seen[e.Name()]; ok {
				return nil, fmt.Errorf(
					"two modules ship a migration named %q in the same schema; "+
						"rename one so the order is unambiguous (previous: %s)",
					e.Name(), describe(prev))
			}
			seen[e.Name()] = e
		}
	}
	if len(seen) == 0 {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrNotExist}
	}

	out := make([]fs.DirEntry, 0, len(seen))
	for _, e := range seen {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out, nil
}

func describe(e fs.DirEntry) string {
	return e.Name()
}
