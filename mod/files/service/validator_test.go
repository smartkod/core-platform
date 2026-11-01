package service

import (
	"bytes"
	"strings"
	"testing"

	"core-platform/contract/core"
)

// Minimal byte prefixes for the formats the validator must recognise. They are
// hand-built so the tests describe exactly what is being sniffed.
var (
	pngMagic  = []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
	jpegMagic = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F'}
	pdfMagic  = []byte("%PDF-1.7\n")
)

func upload(name, declared string, body []byte) core.UploadInput {
	return core.UploadInput{
		OwnerID:     "user-1",
		Filename:    name,
		ContentType: declared,
		Size:        int64(len(body)),
		Body:        bytes.NewReader(body),
	}
}

func TestValidatorAcceptsAllowedTypes(t *testing.T) {
	v := NewValidator(nil, 1<<20)

	cases := []struct {
		name string
		body []byte
		want string
	}{
		{"a.png", append(pngMagic, bytes.Repeat([]byte{0}, 64)...), "image/png"},
		{"a.jpg", append(jpegMagic, bytes.Repeat([]byte{0}, 64)...), "image/jpeg"},
		{"a.pdf", append(pdfMagic, bytes.Repeat([]byte{'x'}, 64)...), "application/pdf"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := v.Check(upload(tc.name, "", tc.body))
			if err != nil {
				t.Fatalf("rejected: %v", err)
			}
			if got != tc.want {
				t.Errorf("type = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestValidatorAcceptsTextWithCharsetParameter covers the most common real
// upload: a browser sends "text/plain; charset=utf-8", and a naive comparison
// against the sniffed "text/plain" would reject every text file.
func TestValidatorAcceptsTextWithCharsetParameter(t *testing.T) {
	v := NewValidator(nil, 1<<20)
	body := []byte(strings.Repeat("hello world. ", 40))

	got, err := v.Check(upload("a.txt", "text/plain; charset=utf-8", body))
	if err != nil {
		t.Fatalf("rejected: %v", err)
	}
	if !strings.HasPrefix(got, "text/plain") {
		t.Errorf("type = %q, want a text/plain variant", got)
	}
}

// TestValidatorRejectsSVG covers the highest-value rule in this file: an SVG is
// an active document, so storing one is stored XSS with extra steps.
func TestValidatorRejectsSVG(t *testing.T) {
	v := NewValidator(nil, 1<<20)
	svg := []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)

	if _, err := v.Check(upload("x.svg", "image/svg+xml", svg)); err == nil {
		t.Fatal("SVG must be rejected")
	}
}

func TestValidatorRejectsSVGAddedToAllowList(t *testing.T) {
	// Even an operator who adds image/svg+xml must not get it through: the check
	// is not an allow-list entry, it is an unconditional refusal.
	v := NewValidator([]string{"image/svg+xml", "image/png"}, 1<<20)
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`)

	if _, err := v.Check(upload("x.svg", "image/svg+xml", svg)); err == nil {
		t.Fatal("SVG must be rejected even when explicitly allow-listed")
	}
}

// TestValidatorRejectsPolyglot covers a client lying about the type: HTML
// declared as a PNG.
func TestValidatorRejectsPolyglot(t *testing.T) {
	v := NewValidator(nil, 1<<20)
	html := []byte("<!DOCTYPE html><html><body><script>alert(1)</script></body></html>")

	if _, err := v.Check(upload("innocent.png", "image/png", html)); err == nil {
		t.Fatal("content that does not match its declared type must be rejected")
	}
}

func TestValidatorRejectsOversize(t *testing.T) {
	v := NewValidator(nil, 100)
	body := bytes.Repeat([]byte("a"), 200)

	if _, err := v.Check(upload("big.txt", "text/plain", body)); err == nil {
		t.Fatal("an oversized upload must be rejected")
	}
}

func TestValidatorRejectsEmpty(t *testing.T) {
	v := NewValidator(nil, 100)
	if _, err := v.Check(upload("empty.txt", "text/plain", nil)); err == nil {
		t.Fatal("an empty upload must be rejected")
	}
}

func TestValidatorRejectsMissingOwner(t *testing.T) {
	v := NewValidator(nil, 100)
	in := upload("a.txt", "text/plain", []byte("hello"))
	in.OwnerID = ""

	if _, err := v.Check(in); err == nil {
		t.Fatal("an upload without an owner must be rejected")
	}
}

// TestValidatorRejectsDangerousFilename covers names that break a header or a
// display, not path traversal: a path component in the name is stripped by
// sanitizeFilename and the object key is built from an id, so "../evil.txt"
// becomes the harmless "evil.txt". What must never pass is a name that could
// split a Content-Disposition header.
func TestValidatorRejectsDangerousFilename(t *testing.T) {
	v := NewValidator(nil, 1<<20)

	for _, name := range []string{"a\r\nb.txt", `quote".txt`, "nul\x00.txt"} {
		t.Run(name, func(t *testing.T) {
			body := []byte(strings.Repeat("hello ", 40))
			if _, err := v.Check(upload(name, "text/plain", body)); err == nil {
				t.Fatalf("filename %q must be rejected", name)
			}
		})
	}
}

// TestValidatorAcceptsPathInFilename documents that a path component is stripped
// rather than rejected, because browsers on some platforms send the full path.
func TestValidatorAcceptsPathInFilename(t *testing.T) {
	v := NewValidator(nil, 1<<20)
	body := []byte(strings.Repeat("hello ", 40))

	if _, err := v.Check(upload("../evil.txt", "text/plain", body)); err != nil {
		t.Fatalf("a path component should be stripped, not rejected: %v", err)
	}
}

// TestInspectRewindsBody guards the streaming contract: the validator peeks at
// the first bytes, and the storage adapter must still receive the whole object.
// Inspect is the entry point the service uses for exactly this reason — Check
// cannot rewind, because it hands back only the type and the caller's own
// reader would already be consumed.
func TestInspectRewindsBody(t *testing.T) {
	v := NewValidator(nil, 1<<20)
	body := append(pngMagic, bytes.Repeat([]byte{7}, 128)...)
	in := upload("a.png", "", body)

	rewound, ctype, err := v.Inspect(in)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if ctype != "image/png" {
		t.Errorf("type = %q, want image/png", ctype)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(rewound.Body); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if buf.Len() != len(body) {
		t.Fatalf("body after Inspect has %d bytes, want %d (it was not rewound)", buf.Len(), len(body))
	}
	if !bytes.Equal(buf.Bytes(), body) {
		t.Fatal("body contents changed during validation")
	}
}

func TestSanitizeFilenameStripsPaths(t *testing.T) {
	cases := map[string]string{
		"../../etc/passwd":  "passwd",
		`C:\Users\me\a.txt`: "a.txt",
		"dir/sub/file.pdf":  "file.pdf",
		"plain.txt":         "plain.txt",
	}
	for in, want := range cases {
		if got := sanitizeFilename(in); got != want {
			t.Errorf("sanitize(%q) = %q, want %q", in, got, want)
		}
	}
}
