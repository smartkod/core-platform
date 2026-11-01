package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"core-platform/contract/core"
	"core-platform/mod/identity/api"
	"core-platform/mod/identity/migrations"
	"core-platform/mod/identity/repository"
	"core-platform/mod/identity/service"
	"core-platform/platform/httpx"
	"core-platform/platform/migrate"
)

// These tests exercise the real HTTP stack against a real database. The unit
// tests cover transport helpers; this file covers what a client actually sees:
// status codes, envelopes, cookies and ownership scoping.

const (
	testAccessSecret  = "0123456789abcdef0123456789abcdef"
	testRefreshSecret = "fedcba9876543210fedcba9876543210"
	testPassword      = "correct horse battery"
)

// harness is a fully wired identity server.
type harness struct {
	echo  *echo.Echo
	db    *gorm.DB
	users *repository.Users
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set; skipping HTTP tests that need a database")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                 gormlogger.Default.LogMode(gormlogger.Silent),
		SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := migrate.RunSource(context.Background(), db, migrations.Source()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// No TRUNCATE: go test runs package binaries in parallel, and the service
	// tests share this database. Truncating would pull rows out from under the
	// other binary (deadlocks, vanished FK targets). Tests instead scope their
	// rows with addresses unique to this process, issued by h.email.

	tokens, err := service.NewTokens(service.TokenConfig{
		AccessSecret:  testAccessSecret,
		RefreshSecret: testRefreshSecret,
		AccessTTL:     15 * time.Minute,
		Issuer:        "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	users := repository.NewUsers(db)
	tokensRepo := repository.NewTokens(db)
	auth := service.NewAuth(users, tokensRepo, tokens, service.NewPassword(10),
		service.NopMailer(), service.NewAuthConfig())

	handler := api.NewHandler(auth, users, api.HandlerConfig{
		CookieSecure: false,
		AccessTTL:    15 * time.Minute,
	})
	identity := api.NewIdentity(tokens, users)

	// A limiter that never refuses: rate limiting has its own tests, and a low
	// limit here would make unrelated assertions fail intermittently.
	limiter := bypassLimiter{}
	limiters := api.NewRateLimiters(limiter, limiter)

	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = httpx.ErrorHandler(discardLogger{})
	e.Use(httpx.RequestID())

	// Group middleware only sticks to routes registered after the Use call, so
	// CSRFProtect comes first — exactly as module.go wires production.
	g := e.Group("/api/v1")
	g.Use(api.CSRFProtect())
	handler.Register(g, identity, limiters)

	return &harness{echo: e, db: db, users: users}
}

// bypassLimiter never refuses and implements the Refund contract.
type bypassLimiter struct{}

func (bypassLimiter) Allow(string) bool { return true }
func (bypassLimiter) Refund(string)     {}

type discardLogger struct{}

func (discardLogger) Info(string, ...any)     {}
func (discardLogger) Warn(string, ...any)     {}
func (discardLogger) Error(string, ...any)    {}
func (discardLogger) With(...any) core.Logger { return discardLogger{} }

// emailSeq counts addresses issued by this binary. It is shared with
// transport_test.go, which lives in the same package.
var emailSeq atomic.Int64

// email builds an address unique to this process and call: two test binaries
// sharing TEST_DATABASE_DSN never collide on the users table.
func (h *harness) email(base string) string {
	return fmt.Sprintf("%s.%d.%d@example.com", base, os.Getpid(), emailSeq.Add(1))
}

// do performs one request and returns the response.
func (h *harness) do(t *testing.T, method, path string, body any, headers map[string]string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()

	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	rec := httptest.NewRecorder()
	h.echo.ServeHTTP(rec, req)

	out := map[string]any{}
	if rec.Body.Len() > 0 {
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
	}
	return rec, out
}

// register creates an account and returns the login payload.
func (h *harness) register(t *testing.T, email string) {
	t.Helper()
	rec, doc := h.do(t, http.MethodPost, "/api/v1/auth/register",
		map[string]string{"email": email, "password": testPassword, "name": "Test"}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status = %d, body %v", rec.Code, doc)
	}
}

// login returns the access and refresh tokens.
func (h *harness) login(t *testing.T, email string) (access, refresh string) {
	t.Helper()
	rec, doc := h.do(t, http.MethodPost, "/api/v1/auth/login",
		map[string]string{"email": email, "password": testPassword}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d, body %v", rec.Code, doc)
	}

	data, _ := doc["data"].(map[string]any)
	access, _ = data["access_token"].(string)
	refresh, _ = data["refresh_token"].(string)
	if access == "" || refresh == "" {
		t.Fatalf("login did not return both tokens: %v", doc)
	}
	return access, refresh
}

func TestRegisterReturnsCreatedWithNoPassword(t *testing.T) {
	h := newHarness(t)
	email := h.email("new")

	rec, doc := h.do(t, http.MethodPost, "/api/v1/auth/register",
		map[string]string{"email": email, "password": testPassword, "name": "Ada"}, nil)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}

	raw := rec.Body.String()
	if strings.Contains(raw, "password") {
		t.Errorf("the response must never mention a password: %s", raw)
	}
	if strings.Contains(raw, testPassword) {
		t.Error("the response leaked the plain password")
	}

	data, _ := doc["data"].(map[string]any)
	if data["email"] != email {
		t.Errorf("email = %v, want the registered address %q", data["email"], email)
	}
}

func TestRegisterWithWeakPasswordIs422(t *testing.T) {
	h := newHarness(t)

	rec, doc := h.do(t, http.MethodPost, "/api/v1/auth/register",
		map[string]string{"email": h.email("weak"), "password": "123"}, nil)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	if env, _ := doc["error"].(map[string]any); env["code"] != "VALIDATION_ERROR" {
		t.Errorf("code = %v, want VALIDATION_ERROR", env["code"])
	}
}

// TestDuplicateRegistrationDoesNotRevealExistence is an enumeration defence: the
// response must be identical whether or not the address exists.
func TestDuplicateRegistrationDoesNotRevealExistence(t *testing.T) {
	h := newHarness(t)
	email := h.email("taken")
	h.register(t, email)

	rec, doc := h.do(t, http.MethodPost, "/api/v1/auth/register",
		map[string]string{"email": email, "password": testPassword}, nil)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", rec.Code)
	}
	data, _ := doc["data"].(map[string]any)
	msg, _ := data["message"].(string)
	if strings.Contains(strings.ToLower(msg), "already") || strings.Contains(strings.ToLower(msg), "exists") {
		t.Errorf("message %q reveals that the account exists", msg)
	}
}

func TestLoginWrongPasswordIs401(t *testing.T) {
	h := newHarness(t)
	email := h.email("user")
	h.register(t, email)

	rec, doc := h.do(t, http.MethodPost, "/api/v1/auth/login",
		map[string]string{"email": email, "password": "wrong password"}, nil)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if env, _ := doc["error"].(map[string]any); env["code"] != "UNAUTHORIZED" {
		t.Errorf("code = %v, want UNAUTHORIZED", env["code"])
	}
}

func TestMeRequiresToken(t *testing.T) {
	h := newHarness(t)

	rec, _ := h.do(t, http.MethodGet, "/api/v1/me", nil, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestMeWithBearerToken(t *testing.T) {
	h := newHarness(t)
	email := h.email("user")
	h.register(t, email)
	access, _ := h.login(t, email)

	rec, doc := h.do(t, http.MethodGet, "/api/v1/me", nil,
		map[string]string{echo.HeaderAuthorization: "Bearer " + access})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %v", rec.Code, doc)
	}
	data, _ := doc["data"].(map[string]any)
	if data["email"] != email {
		t.Errorf("email = %v, want %q", data["email"], email)
	}
}

func TestMeWithGarbageTokenIs401(t *testing.T) {
	h := newHarness(t)

	rec, _ := h.do(t, http.MethodGet, "/api/v1/me", nil,
		map[string]string{echo.HeaderAuthorization: "Bearer not-a-token"})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestRefreshRotatesAndInvalidatesOldToken(t *testing.T) {
	h := newHarness(t)
	email := h.email("user")
	h.register(t, email)
	_, refresh := h.login(t, email)

	rec, doc := h.do(t, http.MethodPost, "/api/v1/auth/refresh",
		map[string]string{"refresh_token": refresh}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("refresh status = %d: %v", rec.Code, doc)
	}
	data, _ := doc["data"].(map[string]any)
	rotated, _ := data["refresh_token"].(string)
	if rotated == "" || rotated == refresh {
		t.Fatal("refresh must rotate the token")
	}

	// Replaying the old token is a theft signal.
	rec, _ = h.do(t, http.MethodPost, "/api/v1/auth/refresh",
		map[string]string{"refresh_token": refresh}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("replayed refresh status = %d, want 401", rec.Code)
	}

	// And the whole family must be dead, including the token just issued.
	rec, _ = h.do(t, http.MethodPost, "/api/v1/auth/refresh",
		map[string]string{"refresh_token": rotated}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("family token status = %d, want 401 after reuse detection", rec.Code)
	}
}

func TestLogoutRevokesRefreshToken(t *testing.T) {
	h := newHarness(t)
	email := h.email("user")
	h.register(t, email)
	_, refresh := h.login(t, email)

	rec, _ := h.do(t, http.MethodPost, "/api/v1/auth/logout",
		map[string]string{"refresh_token": refresh}, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d, want 204", rec.Code)
	}

	rec, _ = h.do(t, http.MethodPost, "/api/v1/auth/refresh",
		map[string]string{"refresh_token": refresh}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("refresh after logout = %d, want 401", rec.Code)
	}

	// Logout is idempotent.
	rec, _ = h.do(t, http.MethodPost, "/api/v1/auth/logout",
		map[string]string{"refresh_token": refresh}, nil)
	if rec.Code != http.StatusNoContent {
		t.Errorf("second logout = %d, want 204", rec.Code)
	}
}

func TestLogoutAllInvalidatesAccessToken(t *testing.T) {
	h := newHarness(t)
	email := h.email("user")
	h.register(t, email)
	access, _ := h.login(t, email)

	rec, _ := h.do(t, http.MethodPost, "/api/v1/me/logout-all", nil,
		map[string]string{echo.HeaderAuthorization: "Bearer " + access})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("logout-all status = %d, want 204", rec.Code)
	}

	// The access token is still cryptographically valid but must be rejected,
	// because token_version was bumped.
	rec, _ = h.do(t, http.MethodGet, "/api/v1/me", nil,
		map[string]string{echo.HeaderAuthorization: "Bearer " + access})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("access after logout-all = %d, want 401", rec.Code)
	}
}

func TestAdminEndpointRequiresAdmin(t *testing.T) {
	h := newHarness(t)
	email := h.email("user")
	h.register(t, email)
	access, _ := h.login(t, email)

	rec, _ := h.do(t, http.MethodGet, "/api/v1/admin/users", nil,
		map[string]string{echo.HeaderAuthorization: "Bearer " + access})
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

func TestAdminEndpointWithoutTokenIs401(t *testing.T) {
	h := newHarness(t)

	rec, _ := h.do(t, http.MethodGet, "/api/v1/admin/users", nil, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestUpdateMeChangesName(t *testing.T) {
	h := newHarness(t)
	email := h.email("user")
	h.register(t, email)
	access, _ := h.login(t, email)
	auth := map[string]string{echo.HeaderAuthorization: "Bearer " + access}

	rec, doc := h.do(t, http.MethodPatch, "/api/v1/me",
		map[string]string{"name": "Yeni İsim"}, auth)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch status = %d: %v", rec.Code, doc)
	}
	data, _ := doc["data"].(map[string]any)
	if data["name"] != "Yeni İsim" {
		t.Errorf("name = %v, want the updated value", data["name"])
	}

	// The change must persist.
	rec, doc = h.do(t, http.MethodGet, "/api/v1/me", nil, auth)
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d", rec.Code)
	}
	data, _ = doc["data"].(map[string]any)
	if data["name"] != "Yeni İsim" {
		t.Errorf("name after re-read = %v", data["name"])
	}
}

func TestDeleteMeRemovesAccess(t *testing.T) {
	h := newHarness(t)
	email := h.email("user")
	h.register(t, email)
	access, _ := h.login(t, email)
	auth := map[string]string{echo.HeaderAuthorization: "Bearer " + access}

	rec, _ := h.do(t, http.MethodDelete, "/api/v1/me", nil, auth)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", rec.Code)
	}

	// The account is gone, so login must fail the same way a wrong password does.
	rec, _ = h.do(t, http.MethodPost, "/api/v1/auth/login",
		map[string]string{"email": email, "password": testPassword}, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("login after delete = %d, want 401", rec.Code)
	}
}

func TestForgotPasswordDoesNotRevealAccounts(t *testing.T) {
	h := newHarness(t)
	knownEmail := h.email("known")
	h.register(t, knownEmail)

	known, _ := h.do(t, http.MethodPost, "/api/v1/auth/forgot-password",
		map[string]string{"email": knownEmail}, nil)
	unknown, _ := h.do(t, http.MethodPost, "/api/v1/auth/forgot-password",
		map[string]string{"email": h.email("nobody")}, nil)

	if known.Code != unknown.Code {
		t.Errorf("known=%d unknown=%d; the responses must be indistinguishable",
			known.Code, unknown.Code)
	}
	if known.Code != http.StatusAccepted {
		t.Errorf("status = %d, want 202", known.Code)
	}
}

func TestResetWithInvalidTokenIs422(t *testing.T) {
	h := newHarness(t)

	rec, _ := h.do(t, http.MethodPost, "/api/v1/auth/reset-password",
		map[string]string{"token": "bogus", "password": "another good password"}, nil)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
}

// TestWebClientGetsHttpOnlyCookies covers the browser transport: the refresh
// token must live in a cookie that JavaScript cannot read.
func TestWebClientGetsHttpOnlyCookies(t *testing.T) {
	h := newHarness(t)
	email := h.email("web")
	h.register(t, email)

	rec, doc := h.do(t, http.MethodPost, "/api/v1/auth/login",
		map[string]string{"email": email, "password": testPassword},
		map[string]string{"X-Client-Type": "web"})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %v", rec.Code, doc)
	}

	// The refresh token must not appear in the JSON body for a web client.
	if raw := rec.Body.String(); strings.Contains(raw, "refresh_token") {
		t.Error("a web login must not return the refresh token in the body")
	}

	var refreshCookie, csrfCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		switch c.Name {
		case api.RefreshCookieName:
			refreshCookie = c
		case api.CSRFCookieName:
			csrfCookie = c
		}
	}

	if refreshCookie == nil {
		t.Fatal("the refresh cookie was not set")
	}
	if !refreshCookie.HttpOnly {
		t.Error("the refresh cookie must be HttpOnly; scripts must not read it")
	}
	if csrfCookie == nil {
		t.Fatal("the CSRF cookie was not set")
	}
	if csrfCookie.HttpOnly {
		t.Error("the CSRF cookie must be readable so the client can echo it")
	}
}

func TestCookieAuthRequiresCSRFOnWrite(t *testing.T) {
	h := newHarness(t)
	email := h.email("csrftest")
	h.register(t, email)

	rec, _ := h.do(t, http.MethodPost, "/api/v1/auth/login",
		map[string]string{"email": email, "password": testPassword},
		map[string]string{"X-Client-Type": "web"})
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d", rec.Code)
	}

	var access, csrf string
	for _, c := range rec.Result().Cookies() {
		switch c.Name {
		case api.AccessCookieName:
			access = c.Value
		case api.CSRFCookieName:
			csrf = c.Value
		}
	}
	if access == "" || csrf == "" {
		t.Fatal("the login response did not set the expected cookies")
	}

	// Cookie-authenticated write without the double-submit header: rejected.
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/me", bytes.NewReader([]byte(`{"name":"x"}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.AddCookie(&http.Cookie{Name: api.AccessCookieName, Value: access})
	req.AddCookie(&http.Cookie{Name: api.CSRFCookieName, Value: csrf})
	w := httptest.NewRecorder()
	h.echo.ServeHTTP(w, req)
	if w.Code == http.StatusOK {
		t.Error("a cookie write without the CSRF header must be rejected")
	}

	// With the header it succeeds.
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/me", bytes.NewReader([]byte(`{"name":"x"}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(api.CSRFHeaderName, csrf)
	req.AddCookie(&http.Cookie{Name: api.AccessCookieName, Value: access})
	req.AddCookie(&http.Cookie{Name: api.CSRFCookieName, Value: csrf})
	w = httptest.NewRecorder()
	h.echo.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 with a matching CSRF header", w.Code)
	}
}

// TestBearerWinsOverCookie documents the mixed-mode rule: when both credentials
// arrive, the header decides, so a stale cookie cannot hijack a request.
func TestBearerWinsOverCookie(t *testing.T) {
	h := newHarness(t)

	h.register(t, h.email("a"))
	emailB := h.email("b")
	h.register(t, emailB)
	accessB, _ := h.login(t, emailB)

	// A cookie for a different user must be ignored when a Bearer is present.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+accessB)
	req.AddCookie(&http.Cookie{Name: api.AccessCookieName, Value: "not-a-valid-token"})

	w := httptest.NewRecorder()
	h.echo.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: the Bearer token must win", w.Code)
	}
	var doc map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &doc)
	data, _ := doc["data"].(map[string]any)
	if data["email"] != emailB {
		t.Errorf("email = %v, want the Bearer user (%q)", data["email"], emailB)
	}
}

func TestListUsersPagination(t *testing.T) {
	h := newHarness(t)
	email := h.email("one")
	h.register(t, email)

	// Promote the account to admin directly: the API has no role endpoint, which
	// is deliberate (an application decides who is an admin).
	if err := h.db.Exec("UPDATE users SET role = 'admin' WHERE email = ?", email).Error; err != nil {
		t.Fatal(err)
	}
	access, _ := h.login(t, email)

	rec, doc := h.do(t, http.MethodGet, "/api/v1/admin/users?limit=10&offset=0", nil,
		map[string]string{echo.HeaderAuthorization: "Bearer " + access})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %v", rec.Code, doc)
	}

	data, _ := doc["data"].(map[string]any)
	for _, key := range []string{"items", "total", "limit", "offset"} {
		if _, ok := data[key]; !ok {
			t.Errorf("response is missing %q; list endpoints must report it", key)
		}
	}
	if total, _ := data["total"].(float64); total < 1 {
		t.Errorf("total = %v, want at least 1", data["total"])
	}
}
