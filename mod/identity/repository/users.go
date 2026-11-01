// Package repository is the only layer that touches the database.
//
// Every query is scoped by user where the caller is a user, so an authorization
// mistake cannot silently become a data leak.
package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"core-platform/contract/core"
	"core-platform/mod/identity/model"
)

// Users is the account store.
type Users struct{ db *gorm.DB }

// NewUsers builds the account store.
func NewUsers(db *gorm.DB) *Users { return &Users{db: db} }

// Create inserts a user.
func (r *Users) Create(ctx context.Context, u *model.User) error {
	return r.db.WithContext(ctx).Create(u).Error
}

// ByEmail finds a live account by e-mail. citext makes this case-insensitive.
func (r *Users) ByEmail(ctx context.Context, email string) (*model.User, error) {
	var u model.User
	err := r.db.WithContext(ctx).
		Where("email = ? AND deleted_at IS NULL", email).
		First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, core.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// ByID finds a live account.
func (r *Users) ByID(ctx context.Context, id string) (*model.User, error) {
	var u model.User
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, core.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// Update writes a user row. Callers pass a fully loaded model.
func (r *Users) Update(ctx context.Context, u *model.User) error {
	u.UpdatedAt = time.Now().UTC()
	return r.db.WithContext(ctx).Save(u).Error
}

// BumpTokenVersion invalidates every access token ever issued for the user.
//
// This is the one lever that makes "log out everywhere" and "password changed"
// actually mean something, because access tokens cannot be revoked one by one.
func (r *Users) BumpTokenVersion(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", userID).
		UpdateColumn("token_version", gorm.Expr("token_version + 1")).Error
}

// SoftDelete marks the account deleted and bumps the token version so open
// sessions stop working immediately.
func (r *Users) SoftDelete(ctx context.Context, userID string) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{
			"deleted_at":    now,
			"status":        model.StatusDeleted,
			"token_version": gorm.Expr("token_version + 1"),
		}).Error
}

// List returns a page of accounts for the admin endpoint.
func (r *Users) List(ctx context.Context, limit, offset int) ([]model.User, int64, error) {
	var out []model.User
	var total int64

	base := r.db.WithContext(ctx).Model(&model.User{}).Where("deleted_at IS NULL")
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := base.Order("created_at DESC").Limit(limit).Offset(offset).Find(&out).Error
	return out, total, err
}
