package service_test

import "testing/fstest"

// fixtures mirrors the module's template set in memory, so the database-backed
// tests do not read from the embed directory.
func fixtures() fstest.MapFS {
	return fstest.MapFS{
		"templates/welcome.subject.tmpl": {
			Data: []byte(`Welcome to {{.AppName}}`),
		},
		"templates/welcome.body.txt.tmpl": {
			Data: []byte("Hi {{.Name}},\n"),
		},
	}
}
