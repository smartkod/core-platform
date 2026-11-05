package security_test

import (
	"net/http"
	"strings"
	"testing"
)

// TestCSRFNegativeCases checks the browser-side rules on a real instance.
//
// The unit tests prove the middleware's logic; these prove the wiring: that the
// check is installed on organization and catalogue routes too, not only on the
// identity group where it originally lived.
func TestCSRFNegativeCases(t *testing.T) {
	w := setup(t)

	t.Run("missing header on a cookie-authenticated write", func(t *testing.T) {
		// Log in in a browser-shaped session so the client holds cookies.
		b := browserClient(t, w.base, w.alice.email)
		status, _, body := b.do(http.MethodPatch, "/api/v1/organizations/"+w.orgA,
			map[string]string{"description": "no csrf"}, map[string]string{
				"Origin": "http://localhost:3000",
			})
		if isSuccess(status) {
			t.Fatalf("a cookie write without X-CSRF-Token succeeded (status %d)", status)
		}
		denied(t, "missing csrf", status, body)
	})

	t.Run("wrong header value", func(t *testing.T) {
		b := browserClient(t, w.base, w.alice.email)
		status, _, body := b.do(http.MethodPatch, "/api/v1/organizations/"+w.orgA,
			map[string]string{"description": "wrong csrf"}, map[string]string{
				"Origin":       "http://localhost:3000",
				"X-CSRF-Token": "definitely-not-the-token",
			})
		if isSuccess(status) {
			t.Fatalf("a wrong CSRF token succeeded (status %d)", status)
		}
		denied(t, "wrong csrf", status, body)
	})

	t.Run("token from another session", func(t *testing.T) {
		victim := browserClient(t, w.base, w.alice.email)
		attacker := browserClient(t, w.base, w.bob.email)

		// The attacker's valid token, sent with the victim's access cookie.
		status, _, body := victim.do(http.MethodPatch, "/api/v1/organizations/"+w.orgA,
			map[string]string{"description": "foreign csrf"}, map[string]string{
				"Origin":       "http://localhost:3000",
				"X-CSRF-Token": attacker.csrfToken(),
			})
		if isSuccess(status) {
			t.Fatalf("a CSRF token from another session succeeded (status %d)", status)
		}
		denied(t, "foreign csrf", status, body)
	})

	t.Run("valid token succeeds", func(t *testing.T) {
		b := browserClient(t, w.base, w.alice.email)
		status, _, body := b.do(http.MethodPatch, "/api/v1/organizations/"+w.orgA,
			map[string]string{"description": "with csrf"}, map[string]string{
				"Origin":       "http://localhost:3000",
				"X-CSRF-Token": b.csrfToken(),
			})
		mustStatus(t, "valid csrf", status, http.StatusOK, body)
	})

	t.Run("bearer is exempt by design", func(t *testing.T) {
		status, _, body := w.alice.do(http.MethodPatch, "/api/v1/organizations/"+w.orgA,
			map[string]string{"description": "bearer"}, nil)
		mustStatus(t, "bearer write", status, http.StatusOK, body)
	})
}

// TestCORSNegativeCases checks that the allow-list actually refuses rather than
// merely not advertising itself.
func TestCORSNegativeCases(t *testing.T) {
	w := setup(t)

	preflight := func(origin string) *http.Response {
		req, err := http.NewRequest(http.MethodOptions,
			w.base+"/api/v1/auth/login", strings.NewReader(""))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Origin", origin)
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "content-type,x-csrf-token,x-client-type")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}

	t.Run("allowed origin", func(t *testing.T) {
		res := preflight("http://localhost:3000")
		defer func() { _ = res.Body.Close() }()
		if res.StatusCode != http.StatusNoContent {
			t.Fatalf("preflight status = %d, want 204", res.StatusCode)
		}
		if got := res.Header.Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
			t.Errorf("allow-origin = %q", got)
		}
		allowed := strings.ToLower(res.Header.Get("Access-Control-Allow-Headers"))
		for _, want := range []string{"x-csrf-token", "x-client-type", "content-type"} {
			if !strings.Contains(allowed, want) {
				t.Errorf("allow-headers %q is missing %q", allowed, want)
			}
		}
	})

	t.Run("unknown origin", func(t *testing.T) {
		res := preflight("http://evil.example")
		defer func() { _ = res.Body.Close() }()
		if got := res.Header.Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("allow-origin = %q for an origin that is not on the list", got)
		}
	})

	t.Run("unknown origin simple request", func(t *testing.T) {
		// A simple POST bypasses preflight but the browser still withholds the
		// response; the server must not echo the origin either.
		req, err := http.NewRequest(http.MethodPost, w.base+"/api/v1/auth/login",
			strings.NewReader(`{"email":"x@y.z","password":"nope-nope-nope"}`))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://evil.example")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = res.Body.Close() }()
		if got := res.Header.Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("allow-origin = %q for an unlisted origin", got)
		}
	})
}

// TestSecurityHeaders records the headers the API sends.
//
// The policy itself is unit-tested in platform/httpx; this proves the running
// instance actually installs it, which a unit test cannot show.
func TestSecurityHeaders(t *testing.T) {
	base := baseURL(t)

	res, err := http.Get(base + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()

	if got := res.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if res.Header.Get("X-Request-Id") == "" {
		t.Error("X-Request-ID is missing")
	}
	if got := res.Header.Get("Referrer-Policy"); got != "no-referrer" {
		t.Errorf("Referrer-Policy = %q, want no-referrer", got)
	}
	if csp := res.Header.Get("Content-Security-Policy"); !strings.Contains(csp, "default-src 'none'") {
		t.Errorf("Content-Security-Policy = %q", csp)
	}
	if got := res.Header.Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("X-Frame-Options = %q, want DENY", got)
	}

	// HSTS is production-only, so it must be absent here: this suite runs
	// against an HTTP development instance.
	if got := res.Header.Get("Strict-Transport-Security"); got != "" {
		t.Errorf("development sent Strict-Transport-Security = %q", got)
	}
}
