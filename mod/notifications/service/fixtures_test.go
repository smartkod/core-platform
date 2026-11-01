package service

import (
	"testing/fstest"
)

// testTemplates mirrors the module's real template set, built in memory so the
// tests do not read from the module's embed directory and cannot be broken by a
// copy edit.
func testTemplates() fstest.MapFS {
	return fstest.MapFS{
		"templates/welcome.subject.tmpl": {
			Data: []byte(`Welcome to {{.AppName}}`),
		},
		"templates/welcome.body.txt.tmpl": {
			Data: []byte("Hi {{.Name}},\n\nYour account is ready.\n"),
		},
		"templates/welcome.body.html.tmpl": {
			Data: []byte(`<p>Hi {{.Name}},</p><p>Your account is ready.</p>`),
		},
		"templates/password_reset.subject.tmpl": {
			Data: []byte(`Reset your {{.AppName}} password`),
		},
		"templates/password_reset.body.txt.tmpl": {
			Data: []byte("Open {{.ResetURL}} within {{.ExpiresIn}}.\n"),
		},
	}
}
