// Package repository is the only layer that touches the notification tables.
package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"core-platform/contract/core"
	"core-platform/mod/notifications/model"
)

// Notifications is the outbox store.
type Notifications struct{ db *gorm.DB }

// NewNotifications builds the store.
func NewNotifications(db *gorm.DB) *Notifications { return &Notifications{db: db} }

// Create inserts a notification.
func (r *Notifications) Create(ctx context.Context, n *model.Notification) error {
	return r.db.WithContext(ctx).Create(n).Error
}

// Due returns pending notifications whose next attempt is in the past.
//
// The FOR UPDATE SKIP LOCKED clause is what makes multiple workers safe: each
// one claims a different set of rows instead of all of them trying to send the
// same message. Without it, two instances would double-send every mail.
func (r *Notifications) Due(ctx context.Context, limit int) ([]model.Notification, error) {
	var out []model.Notification
	err := r.db.WithContext(ctx).
		Raw(`SELECT * FROM notifications
		     WHERE status = ? AND next_attempt_at <= now()
		     ORDER BY next_attempt_at
		     LIMIT ?
		     FOR UPDATE SKIP LOCKED`, string(core.StatusPending), limit).
		Scan(&out).Error
	return out, err
}

// DueForUpdate is Due run inside a transaction held by the caller.
func (r *Notifications) DueForUpdate(ctx context.Context, tx *gorm.DB, limit int) ([]model.Notification, error) {
	var out []model.Notification
	err := tx.WithContext(ctx).
		Raw(`SELECT * FROM notifications
		     WHERE status = ? AND next_attempt_at <= now()
		     ORDER BY next_attempt_at
		     LIMIT ?
		     FOR UPDATE SKIP LOCKED`, string(core.StatusPending), limit).
		Scan(&out).Error
	return out, err
}

// MarkSent records successful delivery.
func (r *Notifications) MarkSent(ctx context.Context, id string) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&model.Notification{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     string(core.StatusSent),
			"sent_at":    now,
			"last_error": "",
			"attempts":   gorm.Expr("attempts + 1"),
		}).Error
}

// MarkFailed records a failed attempt and schedules the next one, or gives up.
//
// Giving up is a status change rather than a deletion: support needs to know a
// message was attempted and abandoned.
func (r *Notifications) MarkFailed(ctx context.Context, id, reason string, nextAt time.Time, giveUp bool) error {
	updates := map[string]any{
		"last_error":      truncate(reason, 500),
		"attempts":        gorm.Expr("attempts + 1"),
		"next_attempt_at": nextAt,
	}
	if giveUp {
		updates["status"] = string(core.StatusFailed)
	}
	return r.db.WithContext(ctx).Model(&model.Notification{}).
		Where("id = ?", id).Updates(updates).Error
}

// MarkSkipped records that the user disabled the channel.
func (r *Notifications) MarkSkipped(ctx context.Context, id, reason string) error {
	return r.db.WithContext(ctx).Model(&model.Notification{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     string(core.StatusSkipped),
			"last_error": truncate(reason, 500),
		}).Error
}

// ByID returns a notification scoped to its owner.
func (r *Notifications) ByID(ctx context.Context, id, userID string) (*model.Notification, error) {
	var n model.Notification
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&n).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, core.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &n, nil
}

// Inbox returns a user's in-app notifications.
func (r *Notifications) Inbox(ctx context.Context, userID string, unreadOnly bool, limit, offset int) ([]model.Notification, int64, error) {
	base := r.db.WithContext(ctx).Model(&model.Notification{}).
		Where("user_id = ? AND channel = ?", userID, string(core.ChannelInApp))
	if unreadOnly {
		base = base.Where("read_at IS NULL")
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []model.Notification
	err := base.Order("created_at DESC").Limit(limit).Offset(offset).Find(&out).Error
	return out, total, err
}

// UnreadCount returns the unread in-app badge count.
func (r *Notifications) UnreadCount(ctx context.Context, userID string) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.Notification{}).
		Where("user_id = ? AND channel = ? AND read_at IS NULL",
			userID, string(core.ChannelInApp)).
		Count(&n).Error
	return n, err
}

// MarkRead marks one notification read, scoped to the owner.
func (r *Notifications) MarkRead(ctx context.Context, userID, id string) error {
	now := time.Now().UTC()
	res := r.db.WithContext(ctx).Model(&model.Notification{}).
		Where("id = ? AND user_id = ? AND read_at IS NULL", id, userID).
		Update("read_at", now)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		// Either it does not belong to this user or it was already read. Both are
		// reported as not found so the response cannot be used to probe ids.
		return core.ErrNotFound
	}
	return nil
}

// MarkAllRead marks every unread notification read and returns the count.
func (r *Notifications) MarkAllRead(ctx context.Context, userID string) (int64, error) {
	now := time.Now().UTC()
	res := r.db.WithContext(ctx).Model(&model.Notification{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Update("read_at", now)
	return res.RowsAffected, res.Error
}

// Preferences returns a user's channel settings, or nil when unset.
func (r *Notifications) Preferences(ctx context.Context, userID string) (*model.Preference, error) {
	var p model.Preference
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// SavePreferences upserts a user's channel settings.
func (r *Notifications) SavePreferences(ctx context.Context, p *model.Preference) error {
	p.UpdatedAt = time.Now().UTC()
	return r.db.WithContext(ctx).Save(p).Error
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
