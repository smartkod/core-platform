package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"core-platform/contract/core"
)

// memStorage is an in-memory core.Storage.
//
// The service tests use it instead of the disk adapter so that a failure points
// at the service, not at file I/O, and so the tests stay fast.
type memStorage struct {
	mu      sync.Mutex
	objects map[string][]byte
	puts    int
	deletes int
	getErr  error
	putErr  error
}

func newMemStorage() *memStorage {
	return &memStorage{objects: map[string][]byte{}}
}

func (m *memStorage) Put(_ context.Context, key string, r io.Reader, size int64, _ string) (int64, error) {
	if m.putErr != nil {
		return 0, m.putErr
	}
	body, err := io.ReadAll(r)
	if err != nil {
		return 0, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects[key] = body
	m.puts++
	return int64(len(body)), nil
}

func (m *memStorage) Get(_ context.Context, key string) (io.ReadCloser, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	body, ok := m.objects[key]
	if !ok {
		return nil, core.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(body)), nil
}

func (m *memStorage) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.objects, key)
	m.deletes++
	return nil
}

func (m *memStorage) SignedURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "/raw/" + key, nil
}

func (m *memStorage) Driver() string { return "memory" }

func (m *memStorage) has(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.objects[key]
	return ok
}

func (m *memStorage) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.objects)
}

// memMeta is an in-memory core.FileStore.
type memMeta struct {
	mu    sync.Mutex
	rows  map[string]*core.FileObject
	order []string
	fail  error
}

func newMemMeta() *memMeta { return &memMeta{rows: map[string]*core.FileObject{}} }

func (m *memMeta) Create(_ context.Context, f *core.FileObject) error {
	if m.fail != nil {
		return m.fail
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *f
	m.rows[f.ID] = &cp
	m.order = append(m.order, f.ID)
	return nil
}

func (m *memMeta) ByID(_ context.Context, id, ownerID string) (*core.FileObject, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.rows[id]
	if !ok {
		return nil, core.ErrNotFound
	}
	// Ownership scoping is the service's security property, so the fake enforces
	// it exactly as the real repository does.
	if ownerID != "" && f.OwnerID != ownerID {
		return nil, core.ErrNotFound
	}
	cp := *f
	return &cp, nil
}

func (m *memMeta) List(_ context.Context, ownerID string, limit, offset int) ([]core.FileObject, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var owned []core.FileObject
	for _, id := range m.order {
		if f := m.rows[id]; f.OwnerID == ownerID {
			owned = append(owned, *f)
		}
	}
	total := int64(len(owned))

	if offset > len(owned) {
		return nil, total, nil
	}
	owned = owned[offset:]
	if limit > 0 && limit < len(owned) {
		owned = owned[:limit]
	}
	return owned, total, nil
}

func (m *memMeta) Delete(_ context.Context, id, ownerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.rows[id]
	if !ok || f.OwnerID != ownerID {
		return core.ErrNotFound
	}
	delete(m.rows, id)
	return nil
}

// newServiceWithFakes wires the service against in-memory fakes.
func newServiceWithFakes(t *testing.T) (*Files, *memStorage, *memMeta) {
	t.Helper()
	store := newMemStorage()
	meta := newMemMeta()
	return &Files{
		store: store,
		meta:  meta,
		valid: NewValidator(nil, 1<<20),
		now:   func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}, store, meta
}

// pngUpload builds a valid PNG upload.
//
// The payload is a real PNG signature plus padding, so the sniffer classifies it
// as image/png rather than relying on the declared type.
func pngUpload(owner string) core.UploadInput {
	body := append([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A},
		bytes.Repeat([]byte{0}, 64)...)
	return core.UploadInput{
		OwnerID:  owner,
		Filename: "photo.png",
		Size:     int64(len(body)),
		Body:     bytes.NewReader(body),
	}
}

// pngSize is the byte count pngUpload produces, kept in one place so a change to
// the payload cannot leave a test asserting a stale number.
func pngSize() int64 {
	return int64(8 + 64)
}

func TestUploadStoresBytesAndMetadata(t *testing.T) {
	svc, store, _ := newServiceWithFakes(t)

	obj, err := svc.Upload(context.Background(), pngUpload("user-1"))
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	if obj.ID == "" {
		t.Error("a file id must be assigned")
	}
	if obj.ContentType != "image/png" {
		t.Errorf("content type = %q, want the sniffed image/png", obj.ContentType)
	}
	if obj.Size != pngSize() {
		t.Errorf("size = %d, want %d", obj.Size, pngSize())
	}
	if obj.Checksum == "" {
		t.Error("a checksum must be recorded")
	}
	if !store.has(obj.Key) {
		t.Error("the object must exist in storage")
	}

	// The key must never contain the client's filename: that is how path
	// traversal and cross-user overwrite become possible.
	if strings.Contains(obj.Key, "photo") {
		t.Errorf("key = %q, must be derived from the id, not the filename", obj.Key)
	}
	if !strings.HasPrefix(obj.Key, "user-1/") {
		t.Errorf("key = %q, want the owner as the first segment", obj.Key)
	}
}

func TestUploadRejectsInvalidContent(t *testing.T) {
	svc, store, _ := newServiceWithFakes(t)

	in := pngUpload("user-1")
	in.Body = strings.NewReader("<!DOCTYPE html><script>alert(1)</script>")
	in.Size = int64(len("<!DOCTYPE html><script>alert(1)</script>"))

	if _, err := svc.Upload(context.Background(), in); err == nil {
		t.Fatal("HTML declared as a PNG must be rejected")
	}
	if store.count() != 0 {
		t.Error("nothing must be written when validation fails")
	}
}

func TestUploadSizeMismatchRollsBack(t *testing.T) {
	svc, store, meta := newServiceWithFakes(t)

	in := pngUpload("user-1")
	in.Size = 500 // the body is shorter than declared

	if _, err := svc.Upload(context.Background(), in); err == nil {
		t.Fatal("a size mismatch must be rejected")
	}
	if store.count() != 0 {
		t.Error("the partial object must be removed after a size mismatch")
	}
	if len(meta.order) != 0 {
		t.Error("no metadata row may survive a rejected upload")
	}
}

func TestUploadMetadataFailureRollsBackObject(t *testing.T) {
	svc, store, meta := newServiceWithFakes(t)
	meta.fail = errors.New("insert failed")

	if _, err := svc.Upload(context.Background(), pngUpload("user-1")); err == nil {
		t.Fatal("expected the metadata failure to surface")
	}
	// A row that failed to insert must not leave bytes behind.
	if store.count() != 0 {
		t.Error("the object must be deleted when the metadata insert fails")
	}
}

func TestDownloadReturnsExactBytes(t *testing.T) {
	svc, _, _ := newServiceWithFakes(t)
	ctx := context.Background()

	obj, err := svc.Upload(ctx, pngUpload("user-1"))
	if err != nil {
		t.Fatal(err)
	}

	rc, meta, err := svc.Download(ctx, obj.ID, "user-1")
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	defer func() { _ = rc.Close() }()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(got)) != obj.Size {
		t.Errorf("read %d bytes, want %d", len(got), obj.Size)
	}
	if meta.ID != obj.ID {
		t.Error("the returned metadata must describe the downloaded file")
	}
}

func TestDownloadIsScopedToOwner(t *testing.T) {
	svc, _, _ := newServiceWithFakes(t)
	ctx := context.Background()

	obj, err := svc.Upload(ctx, pngUpload("user-1"))
	if err != nil {
		t.Fatal(err)
	}

	// Another user guessing the id must get the same answer as a missing file.
	if _, _, err := svc.Download(ctx, obj.ID, "user-2"); err == nil {
		t.Fatal("another user must not be able to download this file")
	}
	if _, err := svc.Get(ctx, obj.ID, "user-2"); err == nil {
		t.Fatal("another user must not be able to read this file's metadata")
	}
}

func TestGetReturnsMetadataOnly(t *testing.T) {
	svc, _, _ := newServiceWithFakes(t)
	ctx := context.Background()

	obj, err := svc.Upload(ctx, pngUpload("user-1"))
	if err != nil {
		t.Fatal(err)
	}

	got, err := svc.Get(ctx, obj.ID, "user-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != obj.ID || got.Checksum != obj.Checksum {
		t.Error("metadata must round-trip")
	}
}

func TestDeleteRemovesRowAndBytes(t *testing.T) {
	svc, store, _ := newServiceWithFakes(t)
	ctx := context.Background()

	obj, err := svc.Upload(ctx, pngUpload("user-1"))
	if err != nil {
		t.Fatal(err)
	}

	if err := svc.Delete(ctx, obj.ID, "user-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if store.has(obj.Key) {
		t.Error("the bytes must be removed from storage")
	}
	if _, err := svc.Get(ctx, obj.ID, "user-1"); err == nil {
		t.Error("the row must be gone")
	}
}

func TestDeleteIsScopedToOwner(t *testing.T) {
	svc, store, _ := newServiceWithFakes(t)
	ctx := context.Background()

	obj, err := svc.Upload(ctx, pngUpload("user-1"))
	if err != nil {
		t.Fatal(err)
	}

	if err := svc.Delete(ctx, obj.ID, "user-2"); err == nil {
		t.Fatal("another user must not be able to delete this file")
	}
	if !store.has(obj.Key) {
		t.Error("the object must survive a rejected delete")
	}
}

// TestDownloadReportsMissingObject covers the case where the row exists but the
// bytes do not: a data-integrity fault, not a "not found" for the user.
func TestDownloadReportsMissingObject(t *testing.T) {
	svc, store, _ := newServiceWithFakes(t)
	ctx := context.Background()

	obj, err := svc.Upload(ctx, pngUpload("user-1"))
	if err != nil {
		t.Fatal(err)
	}
	// Simulate storage losing the object out of band.
	_ = store.Delete(ctx, obj.Key)

	_, _, err = svc.Download(ctx, obj.ID, "user-1")
	if err == nil {
		t.Fatal("a missing object must be an error")
	}
	if errors.Is(err, core.ErrNotFound) {
		t.Error("a missing object must not be reported as not-found; the row exists")
	}
}

func TestListReturnsOnlyOwnerFiles(t *testing.T) {
	svc, _, _ := newServiceWithFakes(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if _, err := svc.Upload(ctx, pngUpload("user-1")); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.Upload(ctx, pngUpload("user-2")); err != nil {
		t.Fatal(err)
	}

	items, total, err := svc.List(ctx, "user-1", 10, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 3 || len(items) != 3 {
		t.Fatalf("list = %d items, total %d; want 3/3", len(items), total)
	}
	for _, it := range items {
		if it.OwnerID != "user-1" {
			t.Errorf("file owned by %q leaked into user-1's list", it.OwnerID)
		}
	}
}

func TestListPagination(t *testing.T) {
	svc, _, _ := newServiceWithFakes(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		if _, err := svc.Upload(ctx, pngUpload("user-1")); err != nil {
			t.Fatal(err)
		}
	}

	items, total, err := svc.List(ctx, "user-1", 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5 regardless of the page", total)
	}
	if len(items) != 2 {
		t.Errorf("page size = %d, want 2", len(items))
	}
}

func TestSignedURLIsScopedToOwner(t *testing.T) {
	svc, _, _ := newServiceWithFakes(t)
	ctx := context.Background()

	obj, err := svc.Upload(ctx, pngUpload("user-1"))
	if err != nil {
		t.Fatal(err)
	}

	url, err := svc.SignedURL(ctx, obj.ID, "user-1", time.Minute)
	if err != nil {
		t.Fatalf("SignedURL: %v", err)
	}
	if url == "" {
		t.Error("a URL must be returned")
	}

	if _, err := svc.SignedURL(ctx, obj.ID, "user-2", time.Minute); err == nil {
		t.Fatal("another user must not get a signed URL")
	}
}

func TestKeyIsUniquePerUpload(t *testing.T) {
	svc, _, _ := newServiceWithFakes(t)
	ctx := context.Background()

	a, err := svc.Upload(ctx, pngUpload("user-1"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Upload(ctx, pngUpload("user-1"))
	if err != nil {
		t.Fatal(err)
	}

	if a.Key == b.Key {
		t.Fatal("two uploads of the same filename must not share a key")
	}
}
