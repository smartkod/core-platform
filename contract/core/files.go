package core

import (
	"context"
	"io"
	"time"
)

// Storage is the object-storage contract other modules and applications use.
//
// It is deliberately narrow: put bytes, get bytes, delete, and mint a temporary
// URL. Anything provider-specific (multipart upload, lifecycle rules, bucket
// policies) stays in the adapter, so switching from local disk to S3 never
// touches application code.
type Storage interface {
	// Put stores an object under key and returns the number of bytes written.
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) (int64, error)
	// Get opens an object for reading. The caller must close it.
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete removes an object. Deleting a missing key is not an error.
	Delete(ctx context.Context, key string) error
	// SignedURL returns a time-limited URL for direct download.
	SignedURL(ctx context.Context, key string, ttl time.Duration) (string, error)
	// Driver reports which backend is in use, for logs and diagnostics.
	Driver() string
}

// FileObject is one stored file's metadata.
//
// The metadata is authoritative and the bytes are not: an object can be missing
// from storage (deleted out of band) while the row survives, which is why the
// service verifies existence rather than trusting the database.
type FileObject struct {
	ID           string
	OwnerID      string
	Key          string
	OriginalName string
	ContentType  string
	Size         int64
	Checksum     string
	CreatedAt    time.Time
}

// FileStore is the metadata side of file storage.
type FileStore interface {
	// Create records a file.
	Create(ctx context.Context, f *FileObject) error
	// ByID returns a file. Implementations must scope by owner when ownerID is
	// non-empty, so a caller cannot read another user's record by guessing an id.
	ByID(ctx context.Context, id, ownerID string) (*FileObject, error)
	// List returns one owner's files, newest first.
	List(ctx context.Context, ownerID string, limit, offset int) ([]FileObject, int64, error)
	// Delete removes the metadata row.
	Delete(ctx context.Context, id, ownerID string) error
}

// Files is the service surface exposed to other modules.
//
// Upload and Delete take the owner explicitly rather than reading it from the
// request: a module calling this from a background job has no HTTP context, and
// an owner taken from context is easy to forget to set.
type Files interface {
	// Upload validates and stores a file for ownerID.
	Upload(ctx context.Context, in UploadInput) (*FileObject, error)
	// Get returns a file's metadata, scoped to ownerID ("" means any owner, for
	// admin paths only).
	Get(ctx context.Context, id, ownerID string) (*FileObject, error)
	// Download opens a file's bytes after the same ownership check.
	Download(ctx context.Context, id, ownerID string) (io.ReadCloser, *FileObject, error)
	// Delete removes the bytes and the row.
	Delete(ctx context.Context, id, ownerID string) error
	// List returns one owner's files.
	List(ctx context.Context, ownerID string, limit, offset int) ([]FileObject, int64, error)
	// SignedURL mints a short-lived download link.
	SignedURL(ctx context.Context, id, ownerID string, ttl time.Duration) (string, error)
}

// UploadInput is one upload request.
type UploadInput struct {
	OwnerID     string
	Filename    string
	ContentType string
	Size        int64
	Body        io.Reader
	// MaxSize overrides the configured limit when non-zero. Used by callers that
	// need a tighter bound than the global setting (an avatar vs. a video).
	MaxSize int64
}
