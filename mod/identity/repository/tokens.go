package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"core-platform/mod/identity/model"
)

// Tokens stores refresh tokens and one-time tokens.
type Tokens struct{ db *gorm.DB }

// NewTokens builds the token store.
func NewTokens(db *gorm.DB) *Tokens { return &Tokens{db: db} }

// Create persists a refresh token row.
func (r *Tokens) Create(ctx context.Context, t *model.RefreshToken) error {
	return r.db.WithContext(ctx).Create(t).Error
}

// ByHash finds a refresh token by its hash.
//
// A missing row is reported as (nil, nil): reuse detection needs to treat
// "unknown token" and "already revoked token" the same way, and neither is an
// internal error.
func (r *Tokens) ByHash(ctx context.Context, hash string) (*model.RefreshToken, error) {
	var t model.RefreshToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// Revoke marks one token revoked and records its successor.
func (r *Tokens) Revoke(ctx context.Context, id, replacedBy string) error {
	now := time.Now().UTC()
	updates := map[string]any{"revoked_at": now}
	if replacedBy != "" {
		updates["replaced_by"] = replacedBy
	}
	return r.db.WithContext(ctx).Model(&model.RefreshToken{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Updates(updates).Error
}

// RevokeFamily revokes every token in a family.
//
// This is what a detected reuse triggers: the stolen token and every token
// derived from the same login stop working together (L1.md 3.1).
func (r *Tokens) RevokeFamily(ctx context.Context, familyID string) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&model.RefreshToken{}).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Update("revoked_at", now).Error
}

// RevokeAllForUser revokes every outstanding session for a user.
func (r *Tokens) RevokeAllForUser(ctx context.Context, userID string) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&model.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error
}

// CreateOneTime persists a verification or reset token.
func (r *Tokens) CreateOneTime(ctx context.Context, t *model.OneTimeToken) error {
	return r.db.WithContext(ctx).Create(t).Error
}

// ConsumeOneTime marks a one-time token used and returns it.
//
// The conditional UPDATE is deliberate: two concurrent requests carrying the
// same reset link must not both succeed.
func (r *Tokens) ConsumeOneTime(ctx context.Context, hash, purpose string) (*model.OneTimeToken, error) {
	now := time.Now().UTC()

	var t model.OneTimeToken
	res := r.db.WithContext(ctx).Model(&model.OneTimeToken{}).
		Where("token_hash = ? AND purpose = ? AND used_at IS NULL AND expires_at > ?", hash, purpose, now).
		Update("used_at", now)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, nil
	}

	if err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// PurgeExpired removes expired token rows. Called by the module's janitor.
func (r *Tokens) PurgeExpired(ctx context.Context) error {
	now := time.Now().UTC()
	if err := r.db.WithContext(ctx).
		Where("expires_at < ?", now).Delete(&model.RefreshToken{}).Error; err != nil {
		return err
	}
	return r.db.WithContext(ctx).
		Where("expires_at < ?", now).Delete(&model.OneTimeToken{}).Error
}
