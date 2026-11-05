package security_test

import (
	"net/http"
	"strings"
	"testing"
)

// TestMassAssignment proves that fields the caller is not allowed to set are
// ignored rather than bound.
//
// The DTO is the boundary: a struct with only the editable fields cannot be
// tricked into writing `role` or `owner_id`, even though the JSON contains
// them. This test asserts the boundary holds at the API, not just in the type.
func TestMassAssignment(t *testing.T) {
	w := setup(t)

	t.Run("registration cannot pick a role", func(t *testing.T) {
		c := newClient(t, w.base)
		email := "mass-" + unique(t) + "@audit.example"
		status, data, raw := c.do(http.MethodPost, "/api/v1/auth/register", map[string]any{
			"email": email, "password": password(), "name": "Mass",
			"role": "admin", "status": "active", "email_verified": true, "token_version": 99,
		}, nil)
		mustStatus(t, "register with extra fields", status, http.StatusCreated, raw)

		var created struct {
			Role           string `json:"role"`
			Status         string `json:"status"`
			EmailVerified  bool   `json:"email_verified"`
			OrganizationID string `json:"organization_id"`
		}
		decodeInto(t, data, &created)
		if created.Role != "user" {
			t.Errorf("role = %q, want user: a caller must not choose its own role", created.Role)
		}
		if created.EmailVerified {
			t.Error("email_verified was set from the request body")
		}
	})

	t.Run("profile update cannot change role", func(t *testing.T) {
		status, data, raw := w.uma.do(http.MethodPatch, "/api/v1/me", map[string]any{
			"name": "Renamed", "role": "admin", "status": "active", "id": w.alice.userID(),
		}, nil)
		mustStatus(t, "patch me", status, http.StatusOK, raw)

		var updated struct {
			ID   string `json:"id"`
			Role string `json:"role"`
			Name string `json:"name"`
		}
		decodeInto(t, data, &updated)
		if updated.Role != "user" {
			t.Errorf("role = %q after a profile update", updated.Role)
		}
		if updated.ID != w.uma.userID() {
			t.Errorf("id changed to %q: the body overrode the session", updated.ID)
		}
		if updated.Name != "Renamed" {
			t.Errorf("name = %q, want the allowed field to have been written", updated.Name)
		}
	})

	t.Run("organization update cannot change status or owner", func(t *testing.T) {
		status, data, raw := w.alice.do(http.MethodPatch, "/api/v1/organizations/"+w.orgA, map[string]any{
			"name": "Audit A", "status": "active", "owner_id": w.uma.userID(),
			"verification_level": "verified",
		}, nil)
		mustStatus(t, "patch organization", status, http.StatusOK, raw)

		var org struct {
			Status            string `json:"status"`
			OwnerID           string `json:"owner_id"`
			VerificationLevel string `json:"verification_level"`
		}
		decodeInto(t, data, &org)
		if org.OwnerID != w.alice.userID() {
			t.Errorf("owner_id = %q: the body transferred ownership", org.OwnerID)
		}
		if org.Status != "draft" {
			t.Errorf("status = %q: the body published the organization", org.Status)
		}
		if org.VerificationLevel != "none" {
			t.Errorf("verification_level = %q: the body granted verification", org.VerificationLevel)
		}
	})

	t.Run("offering creation uses the path tenant", func(t *testing.T) {
		status, data, raw := w.alice.do(http.MethodPost,
			"/api/v1/organizations/"+w.orgA+"/offerings", map[string]any{
				"name": "mass", "slug": "mass-" + unique(t),
				"taxonomy_id":     w.alice.firstTaxonomyID(),
				"organization_id": w.orgB,
				"status":          "published",
			}, nil)
		mustStatus(t, "create offering", status, http.StatusCreated, raw)

		var o struct {
			OrganizationID string `json:"organization_id"`
			Status         string `json:"status"`
		}
		decodeInto(t, data, &o)
		if o.OrganizationID != w.orgA {
			t.Errorf("offering landed in %q, want the path's tenant %q", o.OrganizationID, w.orgA)
		}
		if o.Status != "draft" {
			t.Errorf("status = %q: a body must not publish on create", o.Status)
		}
	})

	t.Run("notification preferences ignore user_id", func(t *testing.T) {
		status, data, raw := w.uma.do(http.MethodPut, "/api/v1/notifications/preferences", map[string]any{
			"email": false, "in_app": false, "push": false,
			"user_id": w.alice.userID(),
		}, nil)
		mustStatus(t, "put preferences", status, http.StatusOK, raw)

		var prefs struct {
			Channels map[string]bool `json:"channels"`
		}
		decodeInto(t, data, &prefs)

		// Alice's preferences must be untouched.
		var alicePrefs struct {
			Channels map[string]bool `json:"channels"`
		}
		w.alice.decode("/api/v1/notifications/preferences", &alicePrefs)
		if !alicePrefs.Channels["email"] {
			t.Error("another user's preferences were changed through a foreign user_id")
		}
	})
}

// TestInjectionPayloads sends SQL and template-shaped values through the
// filters. The assertion is not "the database was attacked"; it is that the API
// answers normally instead of leaking an error or a stack trace.
func TestInjectionPayloads(t *testing.T) {
	w := setup(t)

	payloads := []string{
		"' OR 1=1 --",
		"'; DROP TABLE users; --",
		"\" OR \"\"=\"",
		"1' UNION SELECT null,null,null --",
		"${7*7}",
		"{{7*7}}",
		"%27%20OR%201%3D1--",
		"\\x00",
		"admin'--",
	}

	for _, payload := range payloads {
		payload := payload
		t.Run(shortLabel(payload), func(t *testing.T) {
			// Query filters.
			for _, path := range []string{
				"/api/v1/offerings?q=" + urlQuery(payload),
				"/api/v1/organizations?q=" + urlQuery(payload),
				"/api/v1/taxonomy?parent_id=" + urlQuery(payload),
			} {
				status, _, body := w.alice.do(http.MethodGet, path, nil, nil)
				if status >= 500 {
					t.Errorf("GET %s returned %d (body: %s)", path, status, truncate(body))
				}
				assertNoStacktrace(t, body)
			}

			// Path parameters.
			status, _, body := w.alice.do(http.MethodGet,
				"/api/v1/organizations/"+urlPath(payload), nil, nil)
			if status >= 500 {
				t.Errorf("path injection returned %d (body: %s)", status, truncate(body))
			}
			assertNoStacktrace(t, body)

			// Body fields.
			status, _, body = w.alice.do(http.MethodPost,
				"/api/v1/organizations/"+w.orgA+"/offerings", map[string]any{
					"name": payload, "slug": "inj-" + unique(t), "taxonomy_id": w.alice.firstTaxonomyID(),
				}, nil)
			if status >= 500 {
				t.Errorf("body injection returned %d (body: %s)", status, truncate(body))
			}
			assertNoStacktrace(t, body)
		})
	}
}

// TestXSSIsStoredAsText confirms the API does not transform user input into
// markup. The UI renders with text nodes, so the value must come back verbatim
// and unescaped; the safety lives in the renderer, not in the storage layer.
func TestXSSIsStoredAsText(t *testing.T) {
	w := setup(t)
	payload := `<img src=x onerror="alert(1)">`

	status, data, raw := w.alice.do(http.MethodPost, "/api/v1/organizations/"+w.orgA+"/branches",
		map[string]any{"name": payload, "type": "physical"}, nil)
	mustStatus(t, "create branch with xss payload", status, http.StatusCreated, raw)

	var branch struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	decodeInto(t, data, &branch)
	if branch.Name != payload {
		t.Errorf("stored name = %q, want the exact input %q", branch.Name, payload)
	}

	// And it survives a round trip unchanged (no double encoding).
	var list []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	w.alice.decode("/api/v1/organizations/"+w.orgA+"/branches", &list)
	found := false
	for _, b := range list {
		if b.ID == branch.ID {
			found = true
			if b.Name != payload {
				t.Errorf("read-back name = %q", b.Name)
			}
		}
	}
	if !found {
		t.Error("the created branch is missing from the list")
	}
}

// TestBodyLimitsAndTypes covers the shape checks: oversized bodies, wrong
// types, and values outside the allowed range.
func TestBodyLimitsAndTypes(t *testing.T) {
	w := setup(t)

	t.Run("wrong types are a validation error", func(t *testing.T) {
		status, _, body := w.alice.do(http.MethodPost,
			"/api/v1/organizations/"+w.orgA+"/offerings", map[string]any{
				"name": 123, "slug": []string{"not", "a", "string"},
			}, nil)
		if status != http.StatusUnprocessableEntity && status != http.StatusBadRequest {
			t.Errorf("wrong types: status %d, want 422 or 400 (body: %s)", status, truncate(body))
		}
	})

	t.Run("null body", func(t *testing.T) {
		status, _, body := w.alice.do(http.MethodPatch, "/api/v1/organizations/"+w.orgA, nil, nil)
		if status >= 500 {
			t.Errorf("null body returned %d (body: %s)", status, truncate(body))
		}
	})

	t.Run("oversized body is rejected before it is parsed", func(t *testing.T) {
		// Covered in detail by TestOversizedBodyIsRejected, which speaks raw
		// HTTP so a stalled upload cannot look like a pass.
		t.Skip("see TestOversizedBodyIsRejected")
	})
	t.Run("very long name is rejected or truncated", func(t *testing.T) {
		long := strings.Repeat("x", 10000)
		status, data, body := w.alice.do(http.MethodPost,
			"/api/v1/organizations/"+w.orgA+"/branches",
			map[string]any{"name": long, "type": "physical"}, nil)
		if isSuccess(status) {
			var b struct {
				Name string `json:"name"`
			}
			decodeInto(t, data, &b)
			if len(b.Name) >= len(long) {
				t.Errorf("a %d character name was stored without a bound", len(b.Name))
			}
		} else if status >= 500 {
			t.Errorf("long name returned %d (body: %s)", status, truncate(body))
		}
	})
}

// TestUnknownFieldsAreIgnored makes sure unexpected JSON keys do not produce a
// 500 or a hint about the schema.
func TestUnknownFieldsAreIgnored(t *testing.T) {
	w := setup(t)
	status, _, body := w.alice.do(http.MethodPatch, "/api/v1/organizations/"+w.orgA, map[string]any{
		"name":            "Audit A",
		"totally_unknown": true,
		"nested":          map[string]any{"deep": []int{1, 2, 3}},
		"__proto__":       map[string]any{"polluted": true},
		"constructor":     "x",
	}, nil)
	mustStatus(t, "patch with unknown fields", status, http.StatusOK, body)
}

// ------------------------------------------------------------------- helpers

func assertNoStacktrace(t *testing.T, body string) {
	t.Helper()
	for _, marker := range []string{"goroutine ", "panic:", "runtime error", ".go:", "sql: ", "pq: "} {
		if strings.Contains(body, marker) {
			t.Errorf("response leaks internal detail %q: %s", marker, truncate(body))
		}
	}
}

func shortLabel(s string) string {
	out := strings.Map(func(r rune) rune {
		if r < 32 || r > 126 {
			return '_'
		}
		return r
	}, s)
	if len(out) > 24 {
		out = out[:24]
	}
	return out
}

func urlQuery(s string) string {
	var b strings.Builder
	for _, r := range []byte(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteByte(r)
		default:
			b.WriteString("%")
			const hex = "0123456789ABCDEF"
			b.WriteByte(hex[r>>4])
			b.WriteByte(hex[r&0x0f])
		}
	}
	return b.String()
}

func urlPath(s string) string { return urlQuery(s) }
