package service

import (
	"strings"
	"testing"
)

// The renderer is tested in isolation because template mistakes are otherwise
// invisible: a bad template produces an empty subject and the mail silently
// arrives with no title.

func newTestRenderer(t *testing.T) *Renderer {
	t.Helper()

	// The real templates live in the module's embed directory; the test builds
	// its own so it does not depend on copy changes.
	r, err := NewRenderer(testTemplates(), "templates")
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}
	return r
}

func TestRendererRendersSubjectAndBody(t *testing.T) {
	r := newTestRenderer(t)

	out, err := r.Render("welcome", map[string]any{"Name": "Ada", "AppName": "Demo"})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if !strings.Contains(out.Subject, "Demo") {
		t.Errorf("subject = %q, want it to contain the app name", out.Subject)
	}
	if !strings.Contains(out.Text, "Ada") {
		t.Errorf("text body = %q, want it to contain the name", out.Text)
	}
}

func TestRendererEscapesHTML(t *testing.T) {
	r := newTestRenderer(t)

	// A name is user-supplied. In the HTML body it must be escaped; in the text
	// body it must not be touched.
	out, err := r.Render("welcome", map[string]any{
		"Name":    `<script>alert(1)</script>`,
		"AppName": "Demo",
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if strings.Contains(out.HTML, "<script>") {
		t.Error("HTML body must escape user input; a script tag survived")
	}
	if !strings.Contains(out.HTML, "&lt;script&gt;") {
		t.Errorf("HTML body should contain the escaped form, got %q", out.HTML)
	}
	if !strings.Contains(out.Text, "<script>") {
		t.Error("text body must not HTML-escape its content")
	}
}

func TestRendererMissingKeyIsAnError(t *testing.T) {
	r := newTestRenderer(t)

	// The template references .Name; omitting it must fail rather than render
	// "<no value>" into a customer's inbox.
	if _, err := r.Render("welcome", map[string]any{"AppName": "Demo"}); err == nil {
		t.Fatal("a missing template variable must be an error")
	}
}

func TestRendererUnknownTemplate(t *testing.T) {
	r := newTestRenderer(t)

	_, err := r.Render("no_such_template", nil)
	if err == nil {
		t.Fatal("an unknown template code must be an error")
	}
	if _, ok := err.(*MissingTemplateError); !ok {
		t.Errorf("err = %T, want *MissingTemplateError", err)
	}
}

func TestRendererListsCodes(t *testing.T) {
	r := newTestRenderer(t)

	codes := r.Codes()
	if len(codes) == 0 {
		t.Fatal("no templates were loaded")
	}
	for _, want := range []string{"welcome", "password_reset"} {
		if !r.Has(want) {
			t.Errorf("template %q was not loaded; got %v", want, codes)
		}
	}
}
