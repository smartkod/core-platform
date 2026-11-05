package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"core-platform/contract/core"
	"core-platform/mod/identity/api"
	"core-platform/platform/httpx"
)

// The middleware tests below exercise the transport rules without a database.
// Authentication itself is covered by the service tests; what matters here is
// which credential wins and when CSRF applies.

const testCSRFSecret = "test-csrf-secret-0123456789abcdef"

func csrfEcho() *echo.Echo {
	e := echo.New()
	e.Use(api.CSRFProtect([]byte(testCSRFSecret)))
	e.POST("/write", func(c echo.Context) error { return c.NoContent(http.StatusNoContent) })
	e.GET("/read", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	return e
}

func TestCSRFIgnoresBearerRequests(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/write", strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer some-token")
	rec := httptest.NewRecorder()
	csrfEcho().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204: a Bearer client must not need a CSRF token", rec.Code)
	}
}

func TestCSRFRejectsCookieRequestWithoutToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/write", strings.NewReader("{}"))
	req.AddCookie(&http.Cookie{Name: api.AccessCookieName, Value: "session-a"})
	req.AddCookie(&http.Cookie{Name: api.CSRFCookieName, Value: api.IssueCSRFToken([]byte(testCSRFSecret), "session-a")})
	// No X-CSRF-Token header: this is the cross-site form post case.
	rec := httptest.NewRecorder()
	csrfEcho().ServeHTTP(rec, req)

	// The echo error handler is not installed here, so the sentinel surfaces as
	// a 500. What matters is that the handler did not run.
	if rec.Code == http.StatusNoContent {
		t.Fatal("a cookie-authenticated write without a CSRF header must be rejected")
	}
}

func TestCSRFPassesAnonymousRequests(t *testing.T) {
	// A fresh client (login, register) has no access cookie, so there is no
	// credential for a cross-site form post to hijack and no token to require.
	req := httptest.NewRequest(http.MethodPost, "/write", strings.NewReader("{}"))
	rec := httptest.NewRecorder()
	csrfEcho().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204: an anonymous POST must not need a CSRF token", rec.Code)
	}

	// A stale CSRF cookie alone is not a credential either. Only the access
	// cookie marks a request as cookie-authenticated.
	req = httptest.NewRequest(http.MethodPost, "/write", strings.NewReader("{}"))
	req.AddCookie(&http.Cookie{Name: api.CSRFCookieName, Value: "expected"})
	rec = httptest.NewRecorder()
	csrfEcho().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204: a CSRF cookie without an access cookie is anonymous", rec.Code)
	}
}

func TestCSRFAcceptsMatchingToken(t *testing.T) {
	const session = "session-a"
	token := api.IssueCSRFToken([]byte(testCSRFSecret), session)

	req := httptest.NewRequest(http.MethodPost, "/write", strings.NewReader("{}"))
	req.AddCookie(&http.Cookie{Name: api.AccessCookieName, Value: session})
	req.AddCookie(&http.Cookie{Name: api.CSRFCookieName, Value: token})
	req.Header.Set(api.CSRFHeaderName, token)
	rec := httptest.NewRecorder()
	csrfEcho().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}

func TestCSRFRejectsMismatchedToken(t *testing.T) {
	const session = "session-a"
	token := api.IssueCSRFToken([]byte(testCSRFSecret), session)

	req := httptest.NewRequest(http.MethodPost, "/write", strings.NewReader("{}"))
	req.AddCookie(&http.Cookie{Name: api.AccessCookieName, Value: session})
	req.AddCookie(&http.Cookie{Name: api.CSRFCookieName, Value: token})
	req.Header.Set(api.CSRFHeaderName, "something-else")
	rec := httptest.NewRecorder()
	csrfEcho().ServeHTTP(rec, req)

	if rec.Code == http.StatusNoContent {
		t.Fatal("a mismatched CSRF header must be rejected")
	}
}

// A token issued for session A must not authorize a write performed with
// session B's access cookie. This is the case a plain random double-submit
// cookie cannot detect, and the reason the token is signed over the session.
func TestCSRFRejectsTokenFromAnotherSession(t *testing.T) {
	foreign := api.IssueCSRFToken([]byte(testCSRFSecret), "session-b")

	req := httptest.NewRequest(http.MethodPost, "/write", strings.NewReader("{}"))
	req.AddCookie(&http.Cookie{Name: api.AccessCookieName, Value: "session-a"})
	req.AddCookie(&http.Cookie{Name: api.CSRFCookieName, Value: foreign})
	req.Header.Set(api.CSRFHeaderName, foreign)
	rec := httptest.NewRecorder()
	csrfEcho().ServeHTTP(rec, req)

	if rec.Code == http.StatusNoContent {
		t.Fatal("a CSRF token bound to another session must be rejected")
	}
}

// An unsigned or tampered value must not pass even when the header and cookie
// match, because both could have been planted by a sibling host.
func TestCSRFRejectsUnsignedToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/write", strings.NewReader("{}"))
	req.AddCookie(&http.Cookie{Name: api.AccessCookieName, Value: "session-a"})
	req.AddCookie(&http.Cookie{Name: api.CSRFCookieName, Value: "planted-token"})
	req.Header.Set(api.CSRFHeaderName, "planted-token")
	rec := httptest.NewRecorder()
	csrfEcho().ServeHTTP(rec, req)

	if rec.Code == http.StatusNoContent {
		t.Fatal("an unsigned CSRF token must be rejected")
	}
}

// A wrong signing key must not be accepted: that would mean the browser and
// the server disagree, which is a deployment bug worth failing loudly on.
func TestCSRFRejectsTokenFromAnotherKey(t *testing.T) {
	token := httpx.IssueCSRFToken([]byte("a-completely-different-key"), "session-a")

	req := httptest.NewRequest(http.MethodPost, "/write", strings.NewReader("{}"))
	req.AddCookie(&http.Cookie{Name: api.AccessCookieName, Value: "session-a"})
	req.AddCookie(&http.Cookie{Name: api.CSRFCookieName, Value: token})
	req.Header.Set(api.CSRFHeaderName, token)
	rec := httptest.NewRecorder()
	csrfEcho().ServeHTTP(rec, req)

	if rec.Code == http.StatusNoContent {
		t.Fatal("a CSRF token signed with another key must be rejected")
	}
}

func TestCSRFAllowsSafeMethodsFromCookies(t *testing.T) {
	e := csrfEcho()

	req := httptest.NewRequest(http.MethodGet, "/read", nil)
	req.AddCookie(&http.Cookie{Name: api.RefreshCookieName, Value: "r"})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: GET must not require a CSRF token", rec.Code)
	}
}

// TestUserFromReturnsNilOutsideMiddleware documents the accessor's contract so
// a handler can rely on a nil check instead of a panic.
func TestUserFromReturnsNilOutsideMiddleware(t *testing.T) {
	e := echo.New()
	e.GET("/x", func(c echo.Context) error {
		if core.UserFrom(c) != nil {
			t.Error("expected nil principal on an unauthenticated request")
		}
		return c.NoContent(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
}
