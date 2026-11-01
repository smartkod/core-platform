// Package service renders and queues notifications.
package service

import (
	"bytes"
	"fmt"
	htmltemplate "html/template"
	"io/fs"
	"path"
	"sort"
	"strings"
	texttemplate "text/template"
)

// Renderer turns a template code plus data into a subject and a body.
//
// Two engines are used deliberately: text/template for the plain-text body,
// where escaping would be wrong, and html/template for the HTML body, where
// escaping is the whole point. Rendering the HTML body with text/template would
// let any caller-supplied value inject markup into a mail.
type Renderer struct {
	subjects map[string]*texttemplate.Template
	textBody map[string]*texttemplate.Template
	htmlBody map[string]*htmltemplate.Template
}

// MissingTemplateError names the template that does not exist, so a typo in a
// Send call is obvious instead of producing an empty message.
type MissingTemplateError struct{ Code string }

func (e *MissingTemplateError) Error() string {
	return fmt.Sprintf("notifications: no template named %q", e.Code)
}

// Rendered is the output of one template.
type Rendered struct {
	Subject string
	Text    string
	HTML    string
}

// NewRenderer loads templates from a filesystem.
//
// Taking fs.FS rather than embed.FS keeps the renderer testable with an
// in-memory fixture, and would let a future release load templates from disk
// without changing this signature.
//
// Expected layout under dir, one trio per code:
//
//	<code>.subject.tmpl
//	<code>.body.txt.tmpl
//	<code>.body.html.tmpl   (optional)
func NewRenderer(fsys fs.FS, dir string) (*Renderer, error) {
	r := &Renderer{
		subjects: map[string]*texttemplate.Template{},
		textBody: map[string]*texttemplate.Template{},
		htmlBody: map[string]*htmltemplate.Template{},
	}

	entries, err := fs.ReadDir(fsys, dirOrRoot(dir))
	if err != nil {
		return nil, fmt.Errorf("notifications: read templates: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		full := path.Join(dir, name)
		body, err := fs.ReadFile(fsys, full)
		if err != nil {
			return nil, fmt.Errorf("notifications: read %s: %w", full, err)
		}

		switch {
		case strings.HasSuffix(name, ".subject.tmpl"):
			code := strings.TrimSuffix(name, ".subject.tmpl")
			tpl, err := texttemplate.New(code).Parse(string(body))
			if err != nil {
				return nil, fmt.Errorf("notifications: parse subject %s: %w", name, err)
			}
			r.subjects[code] = tpl

		case strings.HasSuffix(name, ".body.txt.tmpl"):
			code := strings.TrimSuffix(name, ".body.txt.tmpl")
			tpl, err := texttemplate.New(code).Parse(string(body))
			if err != nil {
				return nil, fmt.Errorf("notifications: parse text body %s: %w", name, err)
			}
			r.textBody[code] = tpl

		case strings.HasSuffix(name, ".body.html.tmpl"):
			code := strings.TrimSuffix(name, ".body.html.tmpl")
			tpl, err := htmltemplate.New(code).Parse(string(body))
			if err != nil {
				return nil, fmt.Errorf("notifications: parse html body %s: %w", name, err)
			}
			r.htmlBody[code] = tpl
		}
	}

	// A body without a subject can never be sent: the mailer requires one.
	for code := range r.textBody {
		if _, ok := r.subjects[code]; !ok {
			return nil, fmt.Errorf(
				"notifications: template %q has a body but no subject file", code)
		}
	}

	return r, nil
}

func dirOrRoot(dir string) string {
	if dir == "" {
		return "."
	}
	return dir
}

// Has reports whether a template code is known.
func (r *Renderer) Has(code string) bool {
	_, ok := r.subjects[code]
	return ok
}

// Codes lists the registered template codes, sorted.
func (r *Renderer) Codes() []string {
	out := make([]string, 0, len(r.subjects))
	for k := range r.subjects {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Render executes a template with data.
//
// missingkey=error is set so a typo in a template variable fails loudly instead
// of rendering "<no value>" into a customer's e-mail.
func (r *Renderer) Render(code string, data map[string]any) (Rendered, error) {
	subj, ok := r.subjects[code]
	if !ok {
		return Rendered{}, &MissingTemplateError{Code: code}
	}

	var out Rendered
	var buf bytes.Buffer

	if err := subj.Option("missingkey=error").Execute(&buf, data); err != nil {
		return Rendered{}, fmt.Errorf("notifications: render subject %q: %w", code, err)
	}
	out.Subject = strings.TrimSpace(buf.String())

	buf.Reset()
	if tpl := r.textBody[code]; tpl != nil {
		if err := tpl.Option("missingkey=error").Execute(&buf, data); err != nil {
			return Rendered{}, fmt.Errorf("notifications: render body %q: %w", code, err)
		}
		out.Text = strings.TrimSpace(buf.String())
	}

	buf.Reset()
	if tpl := r.htmlBody[code]; tpl != nil {
		if err := tpl.Option("missingkey=error").Execute(&buf, data); err != nil {
			return Rendered{}, fmt.Errorf("notifications: render html %q: %w", code, err)
		}
		out.HTML = strings.TrimSpace(buf.String())
	}

	return out, nil
}
