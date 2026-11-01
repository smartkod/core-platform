package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"time"

	"core-platform/contract/core"
	"core-platform/platform/uuid"
)

// Files implements core.Files.
type Files struct {
	store core.Storage
	// meta is the interface, not the concrete repository, so the service can be
	// tested against an in-memory store and a different implementation can be
	// substituted without touching this file.
	meta  core.FileStore
	valid *Validator
	now   func() time.Time
}

// NewFiles wires the service.
func NewFiles(store core.Storage, meta core.FileStore, valid *Validator) *Files {
	return &Files{
		store: store,
		meta:  meta,
		valid: valid,
		now:   func() time.Time { return time.Now().UTC() },
	}
}

// Upload validates, stores and records a file.
//
// Order matters: the content is validated before anything is written, and the
// metadata row is created only after the object exists. A crash between the two
// therefore leaves an orphaned object (harmless, swept by a janitor) rather than
// a row pointing at bytes that were never written.
func (f *Files) Upload(ctx context.Context, in core.UploadInput) (*core.FileObject, error) {
	// Inspect returns a copy whose Body has been rewound. Using the original
	// would hand storage an already-consumed reader and store a zero-byte file.
	rewound, contentType, err := f.valid.Inspect(in)
	if err != nil {
		return nil, err
	}

	id := uuid.New()
	// The key is derived from the id, never from the client's filename. A
	// user-supplied path is how path traversal and cross-user overwrites happen;
	// the original name is kept for display only.
	key := keyFor(rewound.OwnerID, id)

	counting := &hashReader{r: rewound.Body}
	written, err := f.store.Put(ctx, key, counting, rewound.Size, contentType)
	if err != nil {
		return nil, err
	}

	// The declared size is advisory. What was actually written is the truth, and
	// a mismatch means the client lied or the stream broke.
	if written != rewound.Size {
		_ = f.store.Delete(ctx, key)
		return nil, fmt.Errorf("%w: expected %d bytes, stored %d",
			core.ErrValidation, rewound.Size, written)
	}

	obj := &core.FileObject{
		ID:           id,
		OwnerID:      rewound.OwnerID,
		Key:          key,
		OriginalName: sanitizeFilename(rewound.Filename),
		ContentType:  contentType,
		Size:         written,
		Checksum:     counting.sumString(),
		CreatedAt:    f.now(),
	}
	if err := f.meta.Create(ctx, obj); err != nil {
		// Roll back the object so a failed insert does not leak storage.
		_ = f.store.Delete(ctx, key)
		return nil, err
	}

	return obj, nil
}

// Get returns metadata, scoped to the owner.
func (f *Files) Get(ctx context.Context, id, ownerID string) (*core.FileObject, error) {
	return f.meta.ByID(ctx, id, ownerID)
}

// Download opens the bytes after the ownership check.
func (f *Files) Download(ctx context.Context, id, ownerID string) (io.ReadCloser, *core.FileObject, error) {
	obj, err := f.meta.ByID(ctx, id, ownerID)
	if err != nil {
		return nil, nil, err
	}

	rc, err := f.store.Get(ctx, obj.Key)
	if err != nil {
		// The row exists but the object does not: a data-integrity problem,
		// reported as a server fault rather than "not found", because telling the
		// user "not found" would send them hunting for a file they can see in
		// their list.
		return nil, nil, fmt.Errorf("%w: object %s is missing from storage",
			core.ErrProviderFailure, obj.ID)
	}
	return rc, obj, nil
}

// Delete removes the metadata first, then the bytes.
//
// Metadata first is deliberate: if the storage delete fails, the object is
// orphaned and a janitor can sweep it, but the user never sees a file they
// deleted. The reverse order would leave a visible row whose download 500s.
func (f *Files) Delete(ctx context.Context, id, ownerID string) error {
	obj, err := f.meta.ByID(ctx, id, ownerID)
	if err != nil {
		return err
	}
	if err := f.meta.Delete(ctx, id, ownerID); err != nil {
		return err
	}
	_ = f.store.Delete(ctx, obj.Key)
	return nil
}

// List returns one owner's files.
func (f *Files) List(ctx context.Context, ownerID string, limit, offset int) ([]core.FileObject, int64, error) {
	return f.meta.List(ctx, ownerID, limit, offset)
}

// SignedURL mints a short-lived download link.
func (f *Files) SignedURL(ctx context.Context, id, ownerID string, ttl time.Duration) (string, error) {
	obj, err := f.meta.ByID(ctx, id, ownerID)
	if err != nil {
		return "", err
	}
	return f.store.SignedURL(ctx, obj.Key, ttl)
}

// keyFor builds an object key.
//
// The owner is a path segment so an operator can find a user's objects, and a
// GDPR delete can remove a whole prefix at once. The id guarantees uniqueness, so
// two uploads of the same filename never collide.
func keyFor(ownerID, id string) string {
	return ownerID + "/" + id
}

// hashReader computes a checksum while the bytes stream to storage, so a large
// upload is never held in memory to be hashed.
type hashReader struct {
	r   io.Reader
	h   hash.Hash
	sum string
}

func (h *hashReader) Read(p []byte) (int, error) {
	if h.h == nil {
		h.h = sha256.New()
	}
	n, err := h.r.Read(p)
	if n > 0 {
		_, _ = h.h.Write(p[:n])
	}
	return n, err
}

func (d *hashReader) sumString() string {
	if d.h == nil {
		return ""
	}
	if d.sum == "" {
		d.sum = hex.EncodeToString(d.h.Sum(nil))
	}
	return d.sum
}

// Compile-time check.
var _ core.Files = (*Files)(nil)
