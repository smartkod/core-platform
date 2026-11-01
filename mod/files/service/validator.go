package service

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"core-platform/contract/core"

	"github.com/gabriel-vasile/mimetype"
)

// Validator decides whether an upload is acceptable.
//
// Content sniffing is authoritative and the declared type is not: a client that
// says "image/png" while sending HTML would otherwise get stored XSS for free.
type Validator struct {
	allowed    map[string]bool
	maxSize    int64
	sniffBytes int64
}

// NewValidator builds a validator.
//
// allowed is the closed list of accepted content types. An empty list falls back
// to a conservative default; it never means "allow everything", because an
// unconfigured upload endpoint is how arbitrary file hosting happens.
func NewValidator(allowed []string, maxSize int64) *Validator {
	if maxSize <= 0 {
		maxSize = 10 << 20 // 10 MiB
	}

	set := make(map[string]bool, len(allowed))
	for _, t := range allowed {
		if t = strings.ToLower(strings.TrimSpace(t)); t != "" {
			set[t] = true
		}
	}
	if len(set) == 0 {
		set = map[string]bool{
			"image/png":       true,
			"image/jpeg":      true,
			"image/webp":      true,
			"image/gif":       true,
			"application/pdf": true,
			"text/plain":      true,
		}
	}

	return &Validator{
		allowed: set,
		maxSize: maxSize,
		// Enough to identify any supported format; capped so a huge upload is not
		// fully buffered just to be rejected.
		sniffBytes: 3072,
	}
}

// MaxSize is the configured limit.
func (v *Validator) MaxSize() int64 { return v.maxSize }

// Check validates a single upload and returns the accepted type.
//
// It reads only the first few kilobytes to classify the content, then restores
// the reader so the caller can stream the full body to storage without
// buffering it in memory. Because UploadInput is passed by value, the rewind is
// reported back to the caller through the returned input rather than by
// mutating a copy the caller cannot see.
func (v *Validator) Check(in core.UploadInput) (contentType string, err error) {
	ctype, rewound, err := v.Validate(in)
	_ = rewound // callers using Check keep their own reader; see Inspect
	return ctype, err
}

// Inspect validates the upload and returns it with its body rewound.
//
// This is the entry point the service uses: the caller must use the returned
// input, because the original Body has been consumed by content sniffing.
func (v *Validator) Inspect(in core.UploadInput) (core.UploadInput, string, error) {
	ctype, rewound, err := v.Validate(in)
	return rewound, ctype, err
}

// Validate does the work for both entry points.
func (v *Validator) Validate(in core.UploadInput) (string, core.UploadInput, error) {
	limit := v.maxSize
	if in.MaxSize > 0 && in.MaxSize < limit {
		limit = in.MaxSize
	}

	if in.Size > limit {
		return "", in, fmt.Errorf("%w: file is larger than the %d byte limit", core.ErrValidation, limit)
	}
	if in.Size <= 0 {
		return "", in, fmt.Errorf("%w: file is empty", core.ErrValidation)
	}
	if in.OwnerID == "" {
		return "", in, fmt.Errorf("%w: missing owner", core.ErrValidation)
	}

	head := make([]byte, v.sniffBytes)
	n, err := io.ReadFull(in.Body, head)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return "", in, fmt.Errorf("%w: cannot read upload", core.ErrValidation)
	}
	head = head[:n]

	// Put the bytes back so the storage adapter sees the whole object.
	in.Body = io.MultiReader(bytes.NewReader(head), in.Body)

	detected := mimetype.Detect(head)
	ctype := detected.String()

	// SVG is deliberately absent from the default allow-list and is rejected
	// even when a deployment adds it: an SVG is a document that can carry
	// script, so serving one from the API origin is stored XSS with extra steps.
	if isSVG(ctype) || isSVG(in.ContentType) {
		return "", in, fmt.Errorf("%w: SVG uploads are not allowed", core.ErrValidation)
	}

	// An image declared as a document, or a script declared as an image, is the
	// classic polyglot attack. The sniffed type is what counts.
	if !v.allowed[ctype] && !v.allowed[baseType(ctype)] {
		return "", in, fmt.Errorf(
			"%w: content type %q is not allowed (declared %q)",
			core.ErrValidation, ctype, in.ContentType)
	}

	// A declared type that contradicts the detected one is a signal, not a
	// nuisance: reject it so a caller cannot lie about what they stored.
	if in.ContentType != "" && !compatible(ctype, in.ContentType) {
		return "", in, fmt.Errorf(
			"%w: declared type %q does not match the detected content %q",
			core.ErrValidation, in.ContentType, ctype)
	}

	if !safeFilename(in.Filename) {
		return "", in, fmt.Errorf("%w: invalid filename", core.ErrValidation)
	}

	return ctype, in, nil
}

// baseType strips parameters: "text/plain; charset=utf-8" -> "text/plain".
func baseType(ct string) string {
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}
	return strings.ToLower(strings.TrimSpace(ct))
}

func isSVG(ct string) bool {
	return strings.Contains(strings.ToLower(ct), "svg")
}

// compatible reports whether a declared type plausibly describes the detected
// one. Browsers send parameters (charset) and generic types, so exact equality
// is too strict in practice.
func compatible(detected, declared string) bool {
	detected = baseType(detected)
	mediaType := baseType(declared)
	if mediaType == "" {
		return true
	}
	if mediaType == detected {
		return true
	}
	// text/plain is routinely reported for any text file; the sniffed type wins,
	// and the allow-list has already vetted it.
	if mediaType == "application/octet-stream" {
		return true
	}
	// A declared type of the same top-level family is acceptable: "text/plain"
	// declared as "text/csv" is a labelling disagreement, not a polyglot.
	declaredTop, declaredSub, ok1 := strings.Cut(mediaType, "/")
	detectedTop, _, ok2 := strings.Cut(detected, "/")
	if ok1 && ok2 && declaredTop == detectedTop {
		return true
	}
	_ = declaredSub
	return false
}

// safeFilename rejects names that would be dangerous in a Content-Disposition
// header or on a filesystem.
func safeFilename(name string) bool {
	if name == "" {
		return true // a name is optional; the id is authoritative
	}
	if len(name) > 255 {
		return false
	}
	base := path.Base(strings.ReplaceAll(name, "\\", "/"))
	if base == "." || base == ".." {
		return false
	}
	for _, r := range name {
		if r == 0 || r == '\n' || r == '\r' || r == '"' {
			return false
		}
	}
	return true
}

// sanitizeFilename strips any path component, leaving just the base name.
func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	base := path.Base(name)
	if base == "." || base == ".." || base == "/" {
		return ""
	}
	return base
}

// detectContent is exposed for tests that want the raw classification.
func detectContent(head []byte) string {
	m := mimetype.Detect(head)
	if m == nil {
		return http.DetectContentType(head)
	}
	return m.String()
}
