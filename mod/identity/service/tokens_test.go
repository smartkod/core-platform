package service

import (
	"testing"
	"time"
)

const (
	testAccessSecret  = "0123456789abcdef0123456789abcdef"
	testRefreshSecret = "fedcba9876543210fedcba9876543210"
)

func tokenService(t *testing.T) *Tokens {
	t.Helper()
	tok, err := NewTokens(TokenConfig{
		AccessSecret:  testAccessSecret,
		RefreshSecret: testRefreshSecret,
		AccessTTL:     15 * time.Minute,
		Issuer:        "test",
	})
	if err != nil {
		t.Fatalf("NewTokens: %v", err)
	}
	return tok
}

func TestTokensRejectIdenticalSecrets(t *testing.T) {
	_, err := NewTokens(TokenConfig{
		AccessSecret:  testAccessSecret,
		RefreshSecret: testAccessSecret,
	})
	if err == nil {
		t.Fatal("identical secrets must be rejected: one leaked key would mint both token types")
	}
}

func TestTokensRequireSecrets(t *testing.T) {
	if _, err := NewTokens(TokenConfig{RefreshSecret: testRefreshSecret}); err == nil {
		t.Fatal("missing access secret must be rejected")
	}
	if _, err := NewTokens(TokenConfig{AccessSecret: testAccessSecret}); err == nil {
		t.Fatal("missing refresh secret must be rejected")
	}
}

func TestAccessTokenRoundTrip(t *testing.T) {
	tok := tokenService(t)

	issued, err := tok.IssueAccess("user-1", "a@b.c", "admin", []string{"read"}, 3, "sess-1")
	if err != nil {
		t.Fatalf("IssueAccess: %v", err)
	}

	claims, err := tok.ParseAccess(issued.Token)
	if err != nil {
		t.Fatalf("ParseAccess: %v", err)
	}
	if claims.Subject != "user-1" {
		t.Errorf("subject = %q, want user-1", claims.Subject)
	}
	if claims.Role != "admin" {
		t.Errorf("role = %q, want admin", claims.Role)
	}
	if claims.Ver != 3 {
		t.Errorf("ver = %d, want 3", claims.Ver)
	}
	if claims.SID != "sess-1" {
		t.Errorf("sid = %q, want sess-1", claims.SID)
	}
}

func TestAccessTokenRejectedWithWrongSecret(t *testing.T) {
	tok := tokenService(t)
	issued, err := tok.IssueAccess("user-1", "a@b.c", "user", nil, 0, "s")
	if err != nil {
		t.Fatal(err)
	}

	other, err := NewTokens(TokenConfig{
		AccessSecret:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RefreshSecret: testRefreshSecret,
		Issuer:        "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := other.ParseAccess(issued.Token); err == nil {
		t.Fatal("a token signed with a different secret must not parse")
	}
}

func TestAccessTokenRejectedWithWrongIssuer(t *testing.T) {
	tok := tokenService(t)
	issued, err := tok.IssueAccess("user-1", "a@b.c", "user", nil, 0, "s")
	if err != nil {
		t.Fatal(err)
	}

	other, err := NewTokens(TokenConfig{
		AccessSecret:  testAccessSecret,
		RefreshSecret: testRefreshSecret,
		Issuer:        "someone-else",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := other.ParseAccess(issued.Token); err == nil {
		t.Fatal("a token from another issuer must not parse")
	}
}

func TestExpiredAccessTokenRejected(t *testing.T) {
	tok, err := NewTokens(TokenConfig{
		AccessSecret:  testAccessSecret,
		RefreshSecret: testRefreshSecret,
		AccessTTL:     time.Millisecond,
		Issuer:        "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Issue in the past so the token is already expired without sleeping.
	tok.now = func() time.Time { return time.Now().UTC().Add(-time.Hour) }
	issued, err := tok.IssueAccess("user-1", "a@b.c", "user", nil, 0, "s")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := tok.ParseAccess(issued.Token); err == nil {
		t.Fatal("an expired token must not parse")
	}
}

func TestRefreshTokenIsOpaqueAndHashed(t *testing.T) {
	tok := tokenService(t)

	v, err := tok.NewRefresh(time.Hour, false)
	if err != nil {
		t.Fatal(err)
	}
	if v.Plain == "" || v.Hash == "" {
		t.Fatal("refresh value must have both plain and hash forms")
	}
	if v.Plain == v.Hash {
		t.Fatal("stored form must differ from the value given to the client")
	}
	if HashToken(v.Plain) != v.Hash {
		t.Fatal("hash must be reproducible from the plain value")
	}

	// Two tokens must never collide.
	other, err := tok.NewRefresh(time.Hour, false)
	if err != nil {
		t.Fatal(err)
	}
	if other.Plain == v.Plain {
		t.Fatal("refresh tokens must be unique")
	}
}

func TestRememberMeExtendsRefreshTTL(t *testing.T) {
	tok := tokenService(t)

	short, err := tok.NewRefresh(time.Hour, false)
	if err != nil {
		t.Fatal(err)
	}
	long, err := tok.NewRefresh(time.Hour, true)
	if err != nil {
		t.Fatal(err)
	}
	if !long.ExpiresAt.After(short.ExpiresAt) {
		t.Fatalf("remember-me expiry %v must be after %v", long.ExpiresAt, short.ExpiresAt)
	}
}
