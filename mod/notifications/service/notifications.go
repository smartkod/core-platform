// Package service queues and delivers notifications.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"core-platform/contract/core"
	"core-platform/mod/notifications/model"
	"core-platform/mod/notifications/repository"
	"core-platform/platform/uuid"
)

// Notifications implements core.Notifications.
type Notifications struct {
	repo     *repository.Notifications
	renderer *Renderer
	mailer   core.Mailer
	now      func() time.Time

	// maxAttempts bounds retries before a message is abandoned.
	maxAttempts int
	// baseDelay is the first retry delay; each further attempt doubles it.
	baseDelay time.Duration
	// maxDelay caps the exponential growth.
	maxDelay time.Duration
}

// Config is the delivery policy.
type Config struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

// DefaultConfig returns the values described in PRD.md §6.5.
func DefaultConfig() Config {
	return Config{
		MaxAttempts: 5,
		BaseDelay:   30 * time.Second,
		MaxDelay:    4 * time.Hour,
	}
}

// New builds the service.
func New(repo *repository.Notifications, renderer *Renderer, mailer core.Mailer, cfg Config) *Notifications {
	if cfg.MaxAttempts <= 0 {
		cfg = DefaultConfig()
	}
	return &Notifications{
		repo:        repo,
		renderer:    renderer,
		mailer:      mailer,
		maxAttempts: cfg.MaxAttempts,
		baseDelay:   cfg.BaseDelay,
		maxDelay:    cfg.MaxDelay,
		now:         func() time.Time { return time.Now().UTC() },
	}
}

// Send renders a template and queues the result.
//
// Rendering happens now, not at delivery time: a retry must send exactly what
// was originally intended even if the template changed in between.
func (n *Notifications) Send(ctx context.Context, in core.SendInput) (*core.Notification, error) {
	if in.UserID == "" {
		return nil, fmt.Errorf("%w: missing user", core.ErrValidation)
	}

	rendered, err := n.renderer.Render(in.Template, in.Data)
	if err != nil {
		return nil, err
	}

	body := rendered.Text
	if body == "" {
		body = rendered.HTML
	}
	if body == "" {
		return nil, fmt.Errorf("%w: template %q rendered an empty body", core.ErrValidation, in.Template)
	}

	return n.queue(ctx, core.SendRawInput{
		UserID:  in.UserID,
		Channel: in.Channel,
		Subject: rendered.Subject,
		Body:    body,
	})
}

// SendRaw queues an already-rendered message.
func (n *Notifications) SendRaw(ctx context.Context, in core.SendRawInput) (*core.Notification, error) {
	if in.UserID == "" {
		return nil, fmt.Errorf("%w: missing user", core.ErrValidation)
	}
	if strings.TrimSpace(in.Subject) == "" {
		return nil, fmt.Errorf("%w: missing subject", core.ErrValidation)
	}
	// A body is required for e-mail but not for an in-app notification that only
	// carries a payload; the check is therefore per channel.
	if in.Channel == core.ChannelEmail && strings.TrimSpace(in.Body) == "" {
		return nil, fmt.Errorf("%w: missing body", core.ErrValidation)
	}
	return n.queue(ctx, in)
}

func (n *Notifications) queue(ctx context.Context, in core.SendRawInput) (*core.Notification, error) {
	channel := in.Channel
	if channel == "" {
		channel = core.ChannelEmail
	}
	switch channel {
	case core.ChannelEmail, core.ChannelInApp, core.ChannelPush:
	default:
		return nil, fmt.Errorf("%w: unknown channel %q", core.ErrValidation, channel)
	}

	payload := "{}"
	if len(in.Payload) > 0 {
		b, err := json.Marshal(in.Payload)
		if err != nil {
			return nil, fmt.Errorf("%w: payload is not serialisable", core.ErrValidation)
		}
		payload = string(b)
	}

	now := n.now()
	status := core.StatusPending
	if channel == core.ChannelInApp {
		// In-app messages need no worker: the client reads them from the inbox.
		status = core.StatusSent
	}

	rec := &model.Notification{
		ID:            uuid.New(),
		UserID:        in.UserID,
		Channel:       string(channel),
		Subject:       in.Subject,
		Body:          in.Body,
		Payload:       payload,
		Status:        string(status),
		NextAttemptAt: now,
		CreatedAt:     now,
	}
	if status == core.StatusSent {
		sent := now
		rec.SentAt = &sent
	}

	if err := n.repo.Create(ctx, rec); err != nil {
		return nil, err
	}
	return toContract(rec), nil
}

// DeliverPending sends due messages. It is called by the worker loop.
//
// Each message is marked before the next is attempted, and a failure schedules a
// retry with exponential backoff. Abandoning after maxAttempts is a status
// change, never a deletion: support must be able to see that a message was
// attempted and dropped.
func (n *Notifications) DeliverPending(ctx context.Context, limit int) (sent, failed int, err error) {
	due, err := n.repo.Due(ctx, limit)
	if err != nil {
		return 0, 0, err
	}

	for _, rec := range due {
		if rec.Channel != string(core.ChannelEmail) {
			// Only e-mail has a transport in V1. Push rows stay pending so a later
			// release can deliver them rather than silently dropping them.
			continue
		}

		ok, skip, sendErr := n.deliver(ctx, rec)
		switch {
		case skip:
			_ = n.repo.MarkSkipped(ctx, rec.ID, skipReason(sendErr))
		case ok:
			_ = n.repo.MarkSent(ctx, rec.ID)
			sent++
		default:
			attempts := rec.Attempts + 1
			giveUp := attempts >= n.maxAttempts
			next := n.now().Add(n.backoff(attempts))
			_ = n.repo.MarkFailed(ctx, rec.ID, errText(sendErr), next, giveUp)
			failed++
		}
	}
	return sent, failed, nil
}

// deliver sends one message and reports whether it should be skipped instead.
func (n *Notifications) deliver(ctx context.Context, rec model.Notification) (ok, skip bool, err error) {
	if n.mailer == nil {
		return false, true, fmt.Errorf("no mail transport configured")
	}

	// The recipient address is resolved by the caller and stored in the payload,
	// because the notifications module does not depend on user lookup: keeping it
	// out means this package works even if the identity schema changes.
	to := rec.PayloadMap()["to"]
	if to == "" {
		return false, true, fmt.Errorf("notification has no recipient address")
	}

	if err := n.mailer.Send(ctx, to, rec.Subject, rec.Body); err != nil {
		return false, false, err
	}
	return true, false, nil
}

// backoff returns the delay before the given attempt.
func (n *Notifications) backoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	d := n.baseDelay
	for i := 1; i < attempt; i++ {
		d *= 2
		if d >= n.maxDelay {
			return n.maxDelay
		}
	}
	return d
}

// List returns a user's in-app notifications.
func (n *Notifications) List(ctx context.Context, userID string, unreadOnly bool, limit, offset int) ([]core.Notification, int64, error) {
	recs, total, err := n.repo.Inbox(ctx, userID, unreadOnly, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out := make([]core.Notification, 0, len(recs))
	for i := range recs {
		out = append(out, *toContract(&recs[i]))
	}
	return out, total, nil
}

// UnreadCount returns the badge count.
func (n *Notifications) UnreadCount(ctx context.Context, userID string) (int64, error) {
	return n.repo.UnreadCount(ctx, userID)
}

// MarkRead marks one notification read.
func (n *Notifications) MarkRead(ctx context.Context, userID, id string) error {
	return n.repo.MarkRead(ctx, userID, id)
}

// MarkAllRead marks every unread notification read.
func (n *Notifications) MarkAllRead(ctx context.Context, userID string) (int64, error) {
	return n.repo.MarkAllRead(ctx, userID)
}

func toContract(m *model.Notification) *core.Notification {
	return &core.Notification{
		ID:            m.ID,
		UserID:        m.UserID,
		Channel:       core.NotificationChannel(m.Channel),
		Template:      m.Template,
		Subject:       m.Subject,
		Body:          m.Body,
		Payload:       m.PayloadMap(),
		Status:        core.NotificationStatus(m.Status),
		Attempts:      m.Attempts,
		LastError:     m.LastError,
		NextAttemptAt: m.NextAttemptAt,
		ReadAt:        m.ReadAt,
		CreatedAt:     m.CreatedAt,
		SentAt:        m.SentAt,
	}
}

func skipReason(err error) string {
	if err == nil {
		return "skipped"
	}
	return err.Error()
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
