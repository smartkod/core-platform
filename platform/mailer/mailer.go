// Package mailer implements core.Mailer.
//
// Two drivers exist: SMTP for real delivery and a no-op that records nothing,
// which is the correct default for a development environment. A third, LogMailer,
// prints messages to the log so a developer can click a reset link without an
// SMTP server.
package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"

	"core-platform/contract/core"
)

// SMTPConfig configures real delivery.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	// From is the envelope and header sender, e.g. "no-reply@example.com".
	From string
	// FromName is the display name.
	FromName string
	// StartTLS upgrades the connection. Disable only for a local relay.
	StartTLS bool
	// Timeout bounds the whole conversation.
	Timeout time.Duration
}

// SMTP sends mail over SMTP.
type SMTP struct{ cfg SMTPConfig }

// NewSMTP builds an SMTP mailer.
func NewSMTP(cfg SMTPConfig) (*SMTP, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("mailer: SMTP host is required")
	}
	if cfg.Port == 0 {
		cfg.Port = 587
	}
	if cfg.From == "" {
		return nil, fmt.Errorf("mailer: a sender address is required")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 15 * time.Second
	}
	return &SMTP{cfg: cfg}, nil
}

// Send delivers one message.
func (s *SMTP) Send(ctx context.Context, to, subject, body string) error {
	if to == "" {
		return fmt.Errorf("mailer: recipient is required")
	}

	from := s.cfg.From
	var fromHeader string
	if s.cfg.FromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", mime.QEncoding.Encode("utf-8", s.cfg.FromName), from)
	} else {
		fromHeader = from
	}

	msg := strings.Join([]string{
		"From: " + fromHeader,
		"To: " + to,
		"Subject: " + mime.QEncoding.Encode("utf-8", subject),
		"MIME-Version: 1.0",
		`Content-Type: text/plain; charset="utf-8"`,
		"",
		body,
	}, "\r\n")

	addr := net.JoinHostPort(s.cfg.Host, fmt.Sprint(s.cfg.Port))

	// net/smtp has no context support, so the deadline is enforced through the
	// dialer. Without it a hung server would block a worker slot forever.
	dialer := &net.Dialer{Timeout: s.cfg.Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("mailer: dial %s: %w", addr, err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(s.cfg.Timeout))
	}

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("mailer: client: %w", err)
	}
	defer func() { _ = client.Close() }()

	if s.cfg.StartTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{
				ServerName: s.cfg.Host,
				MinVersion: tls.VersionTLS12,
			}); err != nil {
				return fmt.Errorf("mailer: starttls: %w", err)
			}
		}
	}

	if s.cfg.Username != "" {
		auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("mailer: auth: %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("mailer: mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("mailer: rcpt: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("mailer: data: %w", err)
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		_ = w.Close()
		return fmt.Errorf("mailer: write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("mailer: close: %w", err)
	}
	return client.Quit()
}

// Logger is the logging subset LogMailer needs.
type Logger interface {
	Info(msg string, kv ...any)
}

// LogMailer writes messages to the log instead of sending them.
//
// It exists so a developer can complete a password reset locally: the link is
// printed, nothing leaves the machine, and no SMTP account is needed.
type LogMailer struct{ log Logger }

// NewLogMailer builds the logging mailer.
func NewLogMailer(l Logger) *LogMailer { return &LogMailer{log: l} }

// Send logs the message.
func (m *LogMailer) Send(ctx context.Context, to, subject, body string) error {
	if m.log != nil {
		m.log.Info("outbound mail (not sent)",
			"to", to, "subject", subject, "body", body)
	}
	return nil
}

// Noop discards messages. It is used when no transport is configured, so the
// queue still records intent without pretending delivery happened.
type Noop struct{}

// Send discards the message.
func (Noop) Send(context.Context, string, string, string) error { return nil }

// Compile-time checks.
var (
	_ core.Mailer = (*SMTP)(nil)
	_ core.Mailer = (*LogMailer)(nil)
	_ core.Mailer = Noop{}
)
