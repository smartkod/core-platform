// Package storage provides the object-storage adapters behind core.Storage.
//
// Two drivers exist: local disk for development and single-node deploys, and
// S3-compatible object storage for anything that must survive a container
// being replaced. Application code depends on core.Storage, never on either
// package, so switching is a configuration change.
package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"core-platform/contract/core"
)

// Local stores objects as files under a root directory.
//
// It is a legitimate production option for a single node, and it is what makes
// a fresh clone runnable without any cloud account.
type Local struct {
	root string
	// publicURL is prepended to signed paths. Empty means the adapter returns a
	// relative path and the HTTP layer serves the bytes itself.
	publicURL string
}

// NewLocal builds a local-disk store rooted at dir.
func NewLocal(dir, publicURL string) (*Local, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, fmt.Errorf("storage: local root directory is required")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("storage: resolve root: %w", err)
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("storage: create root %s: %w", abs, err)
	}
	return &Local{root: abs, publicURL: strings.TrimRight(publicURL, "/")}, nil
}

// Driver implements core.Storage.
func (l *Local) Driver() string { return "local" }

// Put writes the object, creating parent directories as needed.
//
// The write goes to a temporary file first and is then renamed. A crash
// mid-write therefore leaves no half-written object that a later read would
// serve as if it were complete.
func (l *Local) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) (int64, error) {
	path, err := l.resolve(key)
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return 0, fmt.Errorf("storage: mkdir: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".upload-*")
	if err != nil {
		return 0, fmt.Errorf("storage: create temp: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		// Harmless when the rename succeeded: the path no longer exists.
		_ = os.Remove(tmpName)
	}()

	written, err := io.Copy(tmp, r)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return 0, fmt.Errorf("storage: write: %w", err)
	}
	if size > 0 && written != size {
		return 0, fmt.Errorf("storage: wrote %d bytes but %d were declared", written, size)
	}

	if err := os.Rename(tmpName, path); err != nil {
		return 0, fmt.Errorf("storage: commit: %w", err)
	}
	return written, nil
}

// Get opens the object for reading.
func (l *Local) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	path, err := l.resolve(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, core.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("storage: open: %w", err)
	}
	return f, nil
}

// Delete removes the object. A missing object is not an error.
func (l *Local) Delete(ctx context.Context, key string) error {
	path, err := l.resolve(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("storage: delete: %w", err)
	}
	return nil
}

// SignedURL returns a path for the HTTP layer to serve.
//
// A local disk cannot mint a real pre-signed URL, so this returns a route the
// application serves after its own authorization check. Returning a bare path
// pretends to be a signature and would be a security hole if anything trusted
// it, which is why the HTTP layer never treats this URL as proof of anything.
func (l *Local) SignedURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if _, err := l.resolve(key); err != nil {
		return "", err
	}
	// The key is opaque and already random (UUID-based), so it leaks nothing.
	path := "/api/v1/files/raw/" + key
	if l.publicURL != "" {
		return l.publicURL + path, nil
	}
	return path, nil
}

// resolve turns a key into an absolute path and refuses anything that would
// escape the root.
//
// This is the single most important function in the package: a key containing
// ".." or an absolute path must never address a file outside the storage root,
// and a path traversal here would be a full filesystem read for any
// authenticated user.
//
// The key is normalised to forward slashes before any check, because on Windows
// "/absolute" becomes `\absolute` after FromSlash and would slip past
// filepath.IsAbs, which only recognises drive-letter and UNC forms.
func (l *Local) resolve(key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("storage: empty key")
	}

	// A key is always a slash-separated, relative path by contract.
	if strings.HasPrefix(key, "/") || strings.HasPrefix(key, "\\") {
		return "", fmt.Errorf("storage: key must be relative")
	}
	if filepath.IsAbs(key) || filepath.VolumeName(key) != "" {
		return "", fmt.Errorf("storage: key must be relative")
	}

	clean := path.Clean(key)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("storage: key escapes the storage root")
	}

	full := filepath.Join(l.root, filepath.FromSlash(clean))
	rel, err := filepath.Rel(l.root, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("storage: key escapes the storage root")
	}
	return full, nil
}
