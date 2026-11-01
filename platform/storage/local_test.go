package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newLocal(t *testing.T) *Local {
	t.Helper()
	s, err := NewLocal(t.TempDir(), "")
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}
	return s
}

func TestLocalPutGetDelete(t *testing.T) {
	s := newLocal(t)
	ctx := context.Background()

	body := []byte("hello world")
	n, err := s.Put(ctx, "user-1/obj-1", bytes.NewReader(body), int64(len(body)), "text/plain")
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if n != int64(len(body)) {
		t.Fatalf("wrote %d, want %d", n, len(body))
	}

	rc, err := s.Get(ctx, "user-1/obj-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	got, _ := io.ReadAll(rc)
	_ = rc.Close()
	if string(got) != string(body) {
		t.Fatalf("read %q, want %q", got, body)
	}

	if err := s.Delete(ctx, "user-1/obj-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(ctx, "user-1/obj-1"); err == nil {
		t.Fatal("Get after Delete must fail")
	}
}

func TestLocalDeleteMissingIsNotAnError(t *testing.T) {
	s := newLocal(t)
	if err := s.Delete(context.Background(), "never/existed"); err != nil {
		t.Fatalf("deleting a missing object must be idempotent, got %v", err)
	}
}

func TestLocalPutSizeMismatch(t *testing.T) {
	s := newLocal(t)
	body := []byte("short")

	// The caller promised more bytes than it sent. Accepting this would store a
	// truncated object under a database row that claims otherwise.
	_, err := s.Put(context.Background(), "a/b", bytes.NewReader(body), 100, "text/plain")
	if err == nil {
		t.Fatal("a size mismatch must be rejected")
	}
}

// TestLocalRejectsPathTraversal is the most important test in this package. A
// key is derived from internal ids today, but the storage adapter must not be
// the thing that makes a traversal exploitable if that ever changes.
func TestLocalRejectsPathTraversal(t *testing.T) {
	s := newLocal(t)
	ctx := context.Background()

	cases := []string{
		"../escape",
		"../../etc/passwd",
		"a/../../escape",
		"/absolute/path",
		`..\windows\system32`,
		"..",
		".",
		"",
	}

	for _, key := range cases {
		t.Run(key, func(t *testing.T) {
			if _, err := s.Put(ctx, key, strings.NewReader("x"), 1, "text/plain"); err == nil {
				t.Fatalf("key %q must be rejected", key)
			}
			if _, err := s.Get(ctx, key); err == nil {
				t.Fatalf("Get with key %q must be rejected", key)
			}
			if err := s.Delete(ctx, key); err == nil {
				t.Fatalf("Delete with key %q must be rejected", key)
			}
		})
	}
}

func TestLocalSignedURLDoesNotEscapeRoot(t *testing.T) {
	s := newLocal(t)
	if _, err := s.SignedURL(context.Background(), "../secret", 0); err == nil {
		t.Fatal("SignedURL must refuse a traversal key")
	}
}

// TestLocalNoTempFileLeftBehind verifies the write-then-rename strategy does not
// accumulate partial files.
func TestLocalNoTempFileLeftBehind(t *testing.T) {
	root := t.TempDir()
	s, err := NewLocal(root, "")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := s.Put(context.Background(), "u/o", strings.NewReader("data"), 4, "text/plain"); err != nil {
		t.Fatalf("Put: %v", err)
	}

	var temps []string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err == nil && strings.HasPrefix(filepath.Base(path), ".upload-") {
			temps = append(temps, path)
		}
		return nil
	})
	if len(temps) > 0 {
		t.Fatalf("temporary files left behind: %v", temps)
	}
}

func TestLocalDriver(t *testing.T) {
	if got := newLocal(t).Driver(); got != "local" {
		t.Errorf("Driver() = %q, want local", got)
	}
}
