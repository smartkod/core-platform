// Package repository is the only layer that touches the files table.
//
// It implements core.FileStore and owns the mapping between the storage type and
// the public contract type, so no other package has to know about either.
package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"core-platform/contract/core"
	"core-platform/mod/files/model"
)

// Files is the file metadata store.
type Files struct{ db *gorm.DB }

// NewFiles builds the store.
func NewFiles(db *gorm.DB) *Files { return &Files{db: db} }

// Create inserts a file row.
func (r *Files) Create(ctx context.Context, f *core.FileObject) error {
	return r.db.WithContext(ctx).Create(fromContract(f)).Error
}

// ByID returns a live file.
//
// When ownerID is non-empty the query is scoped to that owner, so a caller who
// guesses an identifier gets NotFound rather than someone else's record. An
// empty ownerID is the admin path and must be used deliberately.
func (r *Files) ByID(ctx context.Context, id, ownerID string) (*core.FileObject, error) {
	q := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id)
	if ownerID != "" {
		q = q.Where("owner_id = ?", ownerID)
	}

	var f model.File
	err := q.First(&f).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, core.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return toContract(&f), nil
}

// List returns one owner's live files, newest first.
func (r *Files) List(ctx context.Context, ownerID string, limit, offset int) ([]core.FileObject, int64, error) {
	base := r.db.WithContext(ctx).Model(&model.File{}).
		Where("owner_id = ? AND deleted_at IS NULL", ownerID)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []model.File
	if err := base.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	out := make([]core.FileObject, 0, len(rows))
	for i := range rows {
		out = append(out, *toContract(&rows[i]))
	}
	return out, total, nil
}

// Delete soft-deletes the row, scoped to the owner.
func (r *Files) Delete(ctx context.Context, id, ownerID string) error {
	now := time.Now().UTC()
	res := r.db.WithContext(ctx).Model(&model.File{}).
		Where("id = ? AND owner_id = ? AND deleted_at IS NULL", id, ownerID).
		Update("deleted_at", now)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return core.ErrNotFound
	}
	return nil
}

// HardDelete removes the row entirely. Used by the retention janitor after the
// object has been removed from storage.
func (r *Files) HardDelete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.File{}).Error
}

// toContract converts a storage row to the public type.
//
// The object key is included because the storage adapter needs it; it is the API
// layer's job to keep it out of a JSON response.
func toContract(f *model.File) *core.FileObject {
	return &core.FileObject{
		ID:           f.ID,
		OwnerID:      f.OwnerID,
		Key:          f.ObjectKey,
		OriginalName: f.OriginalName,
		ContentType:  f.ContentType,
		Size:         f.SizeBytes,
		Checksum:     f.Checksum,
		CreatedAt:    f.CreatedAt,
	}
}

// fromContract converts the public type to a storage row.
func fromContract(f *core.FileObject) *model.File {
	return &model.File{
		ID:           f.ID,
		OwnerID:      f.OwnerID,
		ObjectKey:    f.Key,
		OriginalName: f.OriginalName,
		ContentType:  f.ContentType,
		SizeBytes:    f.Size,
		Checksum:     f.Checksum,
		CreatedAt:    f.CreatedAt,
	}
}

// Compile-time check that the store satisfies the contract.
var _ core.FileStore = (*Files)(nil)
