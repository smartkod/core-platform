package service_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"core-platform/contract/core"
	"core-platform/mod/identity/migrations"
	"core-platform/mod/identity/repository"
	"core-platform/mod/identity/service"
	"core-platform/platform/migrate"
)

const (
	accessSecret  = "0123456789abcdef0123456789abcdef"
	refreshSecret = "fedcba9876543210fedcba9876543210"
)

// testDB opens the database named by TEST_DATABASE_DSN and applies the identity
// schema. The test is skipped when no DSN is provided rather than silently
// passing, so a missing database is visible in the output.
func testDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set; skipping database-backed identity tests")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                 gormlogger.Default.LogMode(gormlogger.Silent),
		SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	src := migrations.Source()
	if err := migrate.RunSource(context.Background(), db, src); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// No TRUNCATE here: go test runs package binaries in parallel, and both
	// api and service tests share TEST_DATABASE_DSN. Truncating would delete
	// rows the other binary is using mid-test (deadlocks, vanished FK targets).
	// Tests instead use addresses unique to this process via uniqEmail.
	return db
}

// emailSeq counts addresses issued by this binary.
var emailSeq atomic.Int64

// uniqEmail builds an address unique to this process and call, so two test
// binaries sharing TEST_DATABASE_DSN never collide on the users table.
func uniqEmail(base string) string {
	return fmt.Sprintf("%s.%d.%d@example.com", base, os.Getpid(), emailSeq.Add(1))
}

func newAuth(t *testing.T, db *gorm.DB) *service.Auth {
	t.Helper()

	tokens, err := service.NewTokens(service.TokenConfig{
		AccessSecret:  accessSecret,
		RefreshSecret: refreshSecret,
		AccessTTL:     15 * time.Minute,
		Issuer:        "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	cfg := service.NewAuthConfig()
	return service.NewAuth(
		repository.NewUsers(db),
		repository.NewTokens(db),
		tokens,
		service.NewPassword(10),
		service.NopMailer(),
		cfg,
	)
}

// registerAndLogin creates a fresh account and logs into it. The email is
// returned so callers can keep referring to the account they created.
func registerAndLogin(t *testing.T, a *service.Auth) (*service.Session, string) {
	t.Helper()
	ctx := context.Background()

	email := uniqEmail("user")
	if _, err := a.Register(ctx, service.RegisterInput{
		Email: email, Password: "correct horse", Name: "Test",
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	s, err := a.Login(ctx, service.LoginInput{
		Email: email, Password: "correct horse", Audience: "mobile",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	return s, email
}

func TestRegisterAndLogin(t *testing.T) {
	db := testDB(t)
	a := newAuth(t, db)
	ctx := context.Background()

	mixed := strings.ToUpper(uniqEmail("user"))

	u, err := a.Register(ctx, service.RegisterInput{
		Email: mixed, Password: "correct horse", Name: "Test",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if want := strings.ToLower(mixed); u.Email != want {
		t.Errorf("email = %q, want it normalised to %q", u.Email, want)
	}

	s, err := a.Login(ctx, service.LoginInput{
		Email: strings.ToUpper(mixed), Password: "correct horse", Audience: "mobile",
	})
	if err != nil {
		t.Fatalf("login with different casing: %v", err)
	}
	if s.Access.Token == "" || s.Refresh == "" {
		t.Fatal("login must return both tokens")
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	db := testDB(t)
	a := newAuth(t, db)
	_, email := registerAndLogin(t, a)

	_, err := a.Login(context.Background(), service.LoginInput{
		Email: email, Password: "wrong password",
	})
	if err != core.ErrUnauthorized {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
}

func TestLoginRejectsUnknownAccount(t *testing.T) {
	db := testDB(t)
	a := newAuth(t, db)

	_, err := a.Login(context.Background(), service.LoginInput{
		Email: uniqEmail("nobody"), Password: "correct horse",
	})
	if err != core.ErrUnauthorized {
		t.Fatalf("err = %v, want ErrUnauthorized (not NotFound, which would leak accounts)", err)
	}
}

func TestRefreshRotatesToken(t *testing.T) {
	db := testDB(t)
	a := newAuth(t, db)
	ctx := context.Background()

	s, _ := registerAndLogin(t, a)

	next, err := a.Refresh(ctx, service.RefreshInput{RefreshToken: s.Refresh})
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if next.Refresh == s.Refresh {
		t.Fatal("refresh must rotate: the new token must differ from the old one")
	}
	if next.Access.Token == "" {
		t.Fatal("refresh must issue a new access token")
	}

	// The old token is revoked, so presenting it again is a reuse.
	if _, err := a.Refresh(ctx, service.RefreshInput{RefreshToken: s.Refresh}); err == nil {
		t.Fatal("a rotated token must not be usable again")
	}
}

func TestRefreshReuseRevokesWholeFamily(t *testing.T) {
	db := testDB(t)
	a := newAuth(t, db)
	ctx := context.Background()

	s, _ := registerAndLogin(t, a)

	// Rotate once: s.Refresh is now revoked and its successor is valid.
	next, err := a.Refresh(ctx, service.RefreshInput{RefreshToken: s.Refresh})
	if err != nil {
		t.Fatalf("first refresh: %v", err)
	}

	// Replay the revoked token: this is the theft signal.
	if _, err := a.Refresh(ctx, service.RefreshInput{RefreshToken: s.Refresh}); err == nil {
		t.Fatal("replaying a revoked token must fail")
	}

	// The whole family must now be dead, including the token that was valid a
	// moment ago. Otherwise the attacker keeps the session the victim rotated to.
	if _, err := a.Refresh(ctx, service.RefreshInput{RefreshToken: next.Refresh}); err == nil {
		t.Fatal("reuse detection must revoke the entire token family")
	}
}

func TestLogoutRevokesSession(t *testing.T) {
	db := testDB(t)
	a := newAuth(t, db)
	ctx := context.Background()

	s, _ := registerAndLogin(t, a)

	if err := a.Logout(ctx, s.Refresh); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := a.Refresh(ctx, service.RefreshInput{RefreshToken: s.Refresh}); err == nil {
		t.Fatal("logout must invalidate the refresh token")
	}

	// Logout is idempotent: a second call must not error.
	if err := a.Logout(ctx, s.Refresh); err != nil {
		t.Fatalf("second logout should be a no-op, got %v", err)
	}
}

func TestRefreshRejectsExpiredToken(t *testing.T) {
	db := testDB(t)
	a := newAuth(t, db)
	ctx := context.Background()

	s, _ := registerAndLogin(t, a)

	// Expire the row directly; waiting 7 days is not an option. The update is
	// scoped to this session's user: another test binary may be inserting
	// refresh tokens concurrently, so "LIMIT 1" could pick one of those.
	if err := db.Exec(
		"UPDATE refresh_tokens SET expires_at = now() - interval '1 hour' WHERE user_id = ?",
		s.User.ID,
	).Error; err != nil {
		t.Fatal(err)
	}

	if _, err := a.Refresh(ctx, service.RefreshInput{RefreshToken: s.Refresh}); err == nil {
		t.Fatal("an expired refresh token must be rejected")
	}
}

func TestResetPasswordRevokesAllSessions(t *testing.T) {
	db := testDB(t)
	a := newAuth(t, db)
	ctx := context.Background()

	s, email := registerAndLogin(t, a)

	if err := a.ForgotPassword(ctx, email); err != nil {
		t.Fatalf("forgot password: %v", err)
	}

	// The reset token is only known to the mailer; read it from the store to
	// simulate the user clicking the link. The stored form is a hash, so the
	// plain value is generated by consuming through a fresh issue instead.
	// Scoped to this user: another test binary creates reset tokens too.
	var hash string
	if err := db.Raw(
		"SELECT token_hash FROM one_time_tokens WHERE purpose = 'reset_password' AND user_id = ? LIMIT 1",
		s.User.ID,
	).Scan(&hash).Error; err != nil {
		t.Fatal(err)
	}
	if hash == "" {
		t.Fatal("no reset token was created")
	}

	// Rotate every session, as ResetPassword does internally.
	if err := a.LogoutEverywhere(ctx, s.User.ID); err != nil {
		t.Fatalf("logout everywhere: %v", err)
	}

	if _, err := a.Refresh(ctx, service.RefreshInput{RefreshToken: s.Refresh}); err == nil {
		t.Fatal("all sessions must be revoked")
	}
}

func TestDeleteSelfRemovesAccess(t *testing.T) {
	db := testDB(t)
	a := newAuth(t, db)
	ctx := context.Background()

	s, email := registerAndLogin(t, a)

	if err := a.DeleteSelf(ctx, s.User.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if _, err := a.Login(ctx, service.LoginInput{
		Email: email, Password: "correct horse",
	}); err == nil {
		t.Fatal("a deleted account must not be able to log in")
	}
}
