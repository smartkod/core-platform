package core

import (
	"context"
	"time"
)

// NotificationChannel is a delivery transport.
type NotificationChannel string

// Supported channels.
const (
	// ChannelEmail is implemented in V1.
	ChannelEmail NotificationChannel = "email"
	// ChannelPush is reserved; nothing sends it yet.
	ChannelPush NotificationChannel = "push"
	// ChannelInApp is stored and read by the client, never delivered.
	ChannelInApp NotificationChannel = "in_app"
)

// Notification is one message queued for a user.
//
// It is stored before it is sent. That is the whole point of the outbox: a mail
// server that is down delays a message, it does not lose it.
type Notification struct {
	ID      string
	UserID  string
	Channel NotificationChannel
	// Template is the code of the template used, e.g. "welcome".
	Template string
	// Subject and Body are the rendered result, stored so a retry sends exactly
	// what was originally intended even if the template changes later.
	Subject string
	Body    string
	// Payload carries structured data for channels that need it (deep links).
	Payload map[string]string
	// Status is pending, sent, failed or skipped.
	Status NotificationStatus
	// Attempts counts delivery tries.
	Attempts int
	// LastError explains the most recent failure.
	LastError string
	// NextAttemptAt is when the worker may try again.
	NextAttemptAt time.Time
	ReadAt        *time.Time
	CreatedAt     time.Time
	SentAt        *time.Time
}

// NotificationStatus is the delivery state.
type NotificationStatus string

// Delivery states.
const (
	// StatusPending means queued or awaiting a retry.
	StatusPending NotificationStatus = "pending"
	// StatusSent means the transport accepted it. Acceptance is not proof of
	// delivery: SMTP queues, so this is the strongest signal available.
	StatusSent NotificationStatus = "sent"
	// StatusFailed means retries are exhausted.
	StatusFailed NotificationStatus = "failed"
	// StatusSkipped means the user disabled the channel; the row is kept so
	// support can answer "did we try to contact them".
	StatusSkipped NotificationStatus = "skipped"
)

// Notifications is the service surface other modules use.
type Notifications interface {
	// Send queues a notification rendered from a template.
	Send(ctx context.Context, in SendInput) (*Notification, error)
	// SendRaw queues a notification with an already-rendered body.
	SendRaw(ctx context.Context, in SendRawInput) (*Notification, error)
	// List returns a user's in-app notifications, newest first.
	List(ctx context.Context, userID string, unreadOnly bool, limit, offset int) ([]Notification, int64, error)
	// UnreadCount returns how many unread in-app notifications a user has.
	UnreadCount(ctx context.Context, userID string) (int64, error)
	// MarkRead marks one notification read, scoped to the owner.
	MarkRead(ctx context.Context, userID, id string) error
	// MarkAllRead marks every unread notification read.
	MarkAllRead(ctx context.Context, userID string) (int64, error)
}

// SendInput renders a template and queues the result.
type SendInput struct {
	UserID   string
	Channel  NotificationChannel
	Template string
	Data     map[string]any
}

// SendRawInput queues an already-rendered message.
type SendRawInput struct {
	UserID  string
	Channel NotificationChannel
	Subject string
	Body    string
	Payload map[string]string
}

// NotificationPreferences is a user's channel opt-in state.
//
// Absence of a row means "use the default", not "disabled": a new user must
// receive transactional mail without having to visit a settings page first.
type NotificationPreferences struct {
	UserID string
	// Channels maps a channel to whether it is enabled.
	Channels map[NotificationChannel]bool
	// UpdatedAt is informational.
	UpdatedAt time.Time
}

// PreferenceStore is the preference side of notifications.
type PreferenceStore interface {
	Get(ctx context.Context, userID string) (*NotificationPreferences, error)
	Save(ctx context.Context, p *NotificationPreferences) error
}

// Mailer sends plain-text or simple HTML e-mail.
//
// It is deliberately not templated here: the template engine lives in the
// notifications module, and a mailer that also renders would make retries depend
// on template state that has since changed.
type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// NotificationProvider is implemented by the notifications module.
//
// It exists so other modules can queue a message through Deps without importing
// the notifications package.
type NotificationProvider interface {
	Notifications() Notifications
}
