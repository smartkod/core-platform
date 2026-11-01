package service_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"core-platform/contract/core"
	identitymigrations "core-platform/mod/identity/migrations"
	"core-platform/mod/notifications/migrations"
	"core-platform/mod/notifications/repository"
	"core-platform/mod/notifications/service"
	"core-platform/platform/migrate"
)

// testDB prepares a database with both the identity and notification schemas,
// because notifications reference users.
func testDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set; skipping database-backed notification tests")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                 gormlogger.Default.LogMode(gormlogger.Silent),
		SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	for _, src := range []core.MigrationSource{
		identitymigrations.Source(),
		migrations.Source(),
	} {
		if err := migrate.RunSource(context.Background(), db, src); err != nil {
			t.Fatalf("migrate: %v", err)
		}
	}

	// notifications and notification_preferences are only written here, so they
	// can be wiped safely. users is shared with the identity test binaries,
	// which run in parallel over the same TEST_DATABASE_DSN; truncating it
	// would delete rows they are using mid-test, so it is only ever added to.
	if err := db.Exec("TRUNCATE notifications, notification_preferences CASCADE").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return db
}

// seedSeq makes every seeded user unique within this process: two calls in the
// same clock tick would otherwise produce the same suffix.
var seedSeq atomic.Int64

// seedUser inserts a user row and returns its id. Notifications reference it.
// The row is created rather than looked up and never collides with the users
// other test binaries create, so users is left for the identity tests.
func seedUser(t *testing.T, db *gorm.DB) string {
	t.Helper()

	n := seedSeq.Add(1)
	id := fmt.Sprintf("0199a000-0000-7000-8000-%08x%04d", time.Now().Unix(), n)
	email := fmt.Sprintf("seed.%d.%d@example.com", os.Getpid(), n)
	err := db.Exec(`INSERT INTO users (id, email, password_hash, created_at, updated_at)
	                VALUES (?, ?, 'x', now(), now())`, id, email).Error
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return id
}

// recordingMailer captures what would have been sent.
type recordingMailer struct {
	mu   sync.Mutex
	sent []struct{ To, Subject, Body string }
	err  error
}

func (m *recordingMailer) Send(_ context.Context, to, subject, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.sent = append(m.sent, struct{ To, Subject, Body string }{to, subject, body})
	return nil
}

func (m *recordingMailer) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sent)
}

func newService(t *testing.T, db *gorm.DB, mailer core.Mailer) *service.Notifications {
	t.Helper()

	renderer, err := service.NewRenderer(fixtures(), "templates")
	if err != nil {
		t.Fatalf("renderer: %v", err)
	}

	cfg := service.DefaultConfig()
	cfg.BaseDelay = time.Millisecond
	cfg.MaxDelay = 5 * time.Millisecond
	return service.New(repository.NewNotifications(db), renderer, mailer, cfg)
}

func TestSendQueuesNotification(t *testing.T) {
	db := testDB(t)
	userID := seedUser(t, db)
	mailer := &recordingMailer{}
	svc := newService(t, db, mailer)
	ctx := context.Background()

	n, err := svc.Send(ctx, core.SendInput{
		UserID:   userID,
		Channel:  core.ChannelEmail,
		Template: "welcome",
		Data:     map[string]any{"Name": "Ada", "AppName": "Demo"},
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	if n.Status != core.StatusPending {
		t.Errorf("status = %q, want pending (an e-mail must be delivered later)", n.Status)
	}
	if mailer.count() != 0 {
		t.Error("Send must queue, not deliver; the worker does that")
	}

	// The rendered body is stored, not re-rendered at delivery time.
	if n.Subject == "" || n.Body == "" {
		t.Fatal("the rendered subject and body must be stored with the row")
	}
}

func TestSendInAppIsImmediatelySent(t *testing.T) {
	db := testDB(t)
	userID := seedUser(t, db)
	svc := newService(t, db, &recordingMailer{})
	ctx := context.Background()

	n, err := svc.SendRaw(ctx, core.SendRawInput{
		UserID:  userID,
		Channel: core.ChannelInApp,
		Subject: "Merhaba",
		Body:    "Bir şey oldu",
	})
	if err != nil {
		t.Fatalf("SendRaw: %v", err)
	}

	// An in-app message has no transport; it is "sent" the moment it is stored.
	if n.Status != core.StatusSent {
		t.Errorf("status = %q, want sent", n.Status)
	}
}

func TestSendRejectsUnknownTemplate(t *testing.T) {
	db := testDB(t)
	userID := seedUser(t, db)
	svc := newService(t, db, &recordingMailer{})

	_, err := svc.Send(context.Background(), core.SendInput{
		UserID: userID, Channel: core.ChannelEmail, Template: "nope",
	})
	if err == nil {
		t.Fatal("an unknown template must be rejected, not queued empty")
	}
}

func TestDeliverPendingSendsAndMarksSent(t *testing.T) {
	db := testDB(t)
	userID := seedUser(t, db)
	mailer := &recordingMailer{}
	svc := newService(t, db, mailer)
	ctx := context.Background()

	_, err := svc.SendRaw(ctx, core.SendRawInput{
		UserID:  userID,
		Channel: core.ChannelEmail,
		Subject: "Hello",
		Body:    "World",
	})
	if err != nil {
		t.Fatalf("SendRaw: %v", err)
	}

	sent, failed, err := svc.DeliverPending(ctx, 10)
	if err != nil {
		t.Fatalf("DeliverPending: %v", err)
	}
	if sent != 0 {
		t.Errorf("sent = %d, want 0: the row has no recipient address yet", sent)
	}
	_ = failed

	// With a recipient the message goes out.
	_, err = svc.SendRaw(ctx, core.SendRawInput{
		UserID:  userID,
		Channel: core.ChannelEmail,
		Subject: "Hello",
		Body:    "World",
		Payload: map[string]string{"to": "ada@example.com"},
	})
	if err != nil {
		t.Fatalf("SendRaw: %v", err)
	}

	sent, failed, err = svc.DeliverPending(ctx, 10)
	if err != nil {
		t.Fatalf("DeliverPending: %v", err)
	}
	if sent != 1 || failed != 0 {
		t.Fatalf("sent=%d failed=%d, want 1/0", sent, failed)
	}
	if mailer.count() != 1 {
		t.Fatalf("mailer received %d messages, want 1", mailer.count())
	}
}

// TestDeliveryFailureSchedulesRetry is the core promise of the outbox: a
// transport failure delays a message rather than losing it.
func TestDeliveryFailureSchedulesRetry(t *testing.T) {
	db := testDB(t)
	userID := seedUser(t, db)
	mailer := &recordingMailer{err: errors.New("smtp is down")}
	svc := newService(t, db, mailer)
	ctx := context.Background()

	if _, err := svc.SendRaw(ctx, core.SendRawInput{
		UserID: userID, Channel: core.ChannelEmail,
		Subject: "Hello", Body: "World",
		Payload: map[string]string{"to": "ada@example.com"},
	}); err != nil {
		t.Fatalf("SendRaw: %v", err)
	}

	_, failed, err := svc.DeliverPending(ctx, 10)
	if err != nil {
		t.Fatalf("DeliverPending: %v", err)
	}
	if failed != 1 {
		t.Fatalf("failed = %d, want 1", failed)
	}

	var row struct {
		Status    string
		Attempts  int
		LastError string
	}
	if err := db.Raw("SELECT status, attempts, last_error FROM notifications LIMIT 1").
		Scan(&row).Error; err != nil {
		t.Fatal(err)
	}

	if row.Status != "pending" {
		t.Errorf("status = %q, want pending (it must be retried, not dropped)", row.Status)
	}
	if row.Attempts != 1 {
		t.Errorf("attempts = %d, want 1", row.Attempts)
	}
	if row.LastError == "" {
		t.Error("last_error must record why the attempt failed")
	}
}

// TestRetriesAreExhausted verifies that a permanently broken message eventually
// stops being retried and is kept as failed rather than deleted.
func TestRetriesAreExhausted(t *testing.T) {
	db := testDB(t)
	userID := seedUser(t, db)
	mailer := &recordingMailer{err: errors.New("smtp is down")}

	renderer, err := service.NewRenderer(fixtures(), "templates")
	if err != nil {
		t.Fatal(err)
	}
	cfg := service.Config{
		MaxAttempts: 2,
		BaseDelay:   time.Millisecond,
		MaxDelay:    2 * time.Millisecond,
	}
	svc := service.New(repository.NewNotifications(db), renderer, mailer, cfg)
	ctx := context.Background()

	if _, err := svc.SendRaw(ctx, core.SendRawInput{
		UserID: userID, Channel: core.ChannelEmail,
		Subject: "Hello", Body: "World",
		Payload: map[string]string{"to": "ada@example.com"},
	}); err != nil {
		t.Fatal(err)
	}

	// Two passes: the first schedules a retry, the second exhausts the budget.
	for i := 0; i < 2; i++ {
		time.Sleep(5 * time.Millisecond)
		if _, _, err := svc.DeliverPending(ctx, 10); err != nil {
			t.Fatalf("pass %d: %v", i+1, err)
		}
	}

	var status string
	if err := db.Raw("SELECT status FROM notifications LIMIT 1").Scan(&status).Error; err != nil {
		t.Fatal(err)
	}
	if status != "failed" {
		t.Errorf("status = %q, want failed after exhausting retries", status)
	}

	var count int64
	db.Raw("SELECT count(*) FROM notifications").Scan(&count)
	if count != 1 {
		t.Error("a failed notification must be kept, not deleted")
	}
}

func TestInboxAndReadState(t *testing.T) {
	db := testDB(t)
	userID := seedUser(t, db)
	svc := newService(t, db, &recordingMailer{})
	ctx := context.Background()

	var ids []string
	for i := 0; i < 3; i++ {
		n, err := svc.SendRaw(ctx, core.SendRawInput{
			UserID: userID, Channel: core.ChannelInApp,
			Subject: "Message", Body: "Body",
		})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, n.ID)
	}

	items, total, err := svc.List(ctx, userID, false, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(items) != 3 {
		t.Fatalf("list = %d items, total %d; want 3/3", len(items), total)
	}

	if unread, err := svc.UnreadCount(ctx, userID); err != nil || unread != 3 {
		t.Fatalf("unread = %d (%v), want 3", unread, err)
	}

	if err := svc.MarkRead(ctx, userID, ids[0]); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	if unread, _ := svc.UnreadCount(ctx, userID); unread != 2 {
		t.Errorf("unread after marking one read = %d, want 2", unread)
	}

	if n, err := svc.MarkAllRead(ctx, userID); err != nil || n != 2 {
		t.Errorf("MarkAllRead = %d (%v), want 2", n, err)
	}
	if unread, _ := svc.UnreadCount(ctx, userID); unread != 0 {
		t.Errorf("unread after mark-all = %d, want 0", unread)
	}
}

// TestMarkReadIsScopedToOwner prevents one user from clearing another's badge by
// guessing an id.
func TestMarkReadIsScopedToOwner(t *testing.T) {
	db := testDB(t)
	ownerID := seedUser(t, db)
	otherID := seedUser(t, db)
	svc := newService(t, db, &recordingMailer{})
	ctx := context.Background()

	n, err := svc.SendRaw(ctx, core.SendRawInput{
		UserID: ownerID, Channel: core.ChannelInApp,
		Subject: "Message", Body: "Body",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := svc.MarkRead(ctx, otherID, n.ID); err == nil {
		t.Fatal("a different user must not be able to mark this notification read")
	}
	if unread, _ := svc.UnreadCount(ctx, ownerID); unread != 1 {
		t.Errorf("owner's unread = %d, want 1", unread)
	}
}

func TestListIsScopedToOwner(t *testing.T) {
	db := testDB(t)
	ownerID := seedUser(t, db)
	otherID := seedUser(t, db)
	svc := newService(t, db, &recordingMailer{})
	ctx := context.Background()

	if _, err := svc.SendRaw(ctx, core.SendRawInput{
		UserID: ownerID, Channel: core.ChannelInApp,
		Subject: "Private", Body: "Body",
	}); err != nil {
		t.Fatal(err)
	}

	items, total, err := svc.List(ctx, otherID, false, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || len(items) != 0 {
		t.Fatalf("another user sees %d notifications; it must be 0", len(items))
	}
}

func TestSendRejectsMissingSubject(t *testing.T) {
	db := testDB(t)
	userID := seedUser(t, db)
	svc := newService(t, db, &recordingMailer{})

	_, err := svc.SendRaw(context.Background(), core.SendRawInput{
		UserID: userID, Channel: core.ChannelEmail, Body: "no subject",
	})
	if err == nil {
		t.Fatal("an e-mail without a subject must be rejected")
	}
}

func TestSendRejectsUnknownChannel(t *testing.T) {
	db := testDB(t)
	userID := seedUser(t, db)
	svc := newService(t, db, &recordingMailer{})

	_, err := svc.SendRaw(context.Background(), core.SendRawInput{
		UserID: userID, Channel: "carrier_pigeon", Subject: "s", Body: "b",
	})
	if err == nil {
		t.Fatal("an unknown channel must be rejected")
	}
}
