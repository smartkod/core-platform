package security_test

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestRoleMatrix is the authorization contract per role.
//
// The rows are what the permission seed in 0004_authz_init declares; the test
// proves the running server enforces them, not just the UI.
func TestRoleMatrix(t *testing.T) {
	w := setup(t)

	type caseRow struct {
		name   string
		method string
		path   func() string
		body   func() any
		allow  map[string]bool // role -> allowed
	}

	all := map[string]bool{"owner": true, "manager": true, "staff": true, "member": true}
	ownerOnly := map[string]bool{"owner": true, "manager": false, "staff": false, "member": false}
	managers := map[string]bool{"owner": true, "manager": true, "staff": false, "member": false}
	catalogWriters := map[string]bool{"owner": true, "manager": true, "staff": true, "member": false}
	catalogPublishers := map[string]bool{"owner": true, "manager": true, "staff": false, "member": false}

	target := func() string { return w.alice.createOffering(w.orgA, "matrix-"+unique(t), "matrix-"+unique(t)) }

	rows := []caseRow{
		{
			name: "read organization", method: http.MethodGet,
			path:  func() string { return "/api/v1/organizations/" + w.orgA },
			allow: all,
		},
		{
			name: "edit organization", method: http.MethodPatch,
			path:  func() string { return "/api/v1/organizations/" + w.orgA },
			body:  func() any { return map[string]string{"description": "matrix"} },
			allow: managers,
		},
		{
			name: "list branches", method: http.MethodGet,
			path:  func() string { return "/api/v1/organizations/" + w.orgA + "/branches" },
			allow: all,
		},
		{
			name: "create branch", method: http.MethodPost,
			path: func() string { return "/api/v1/organizations/" + w.orgA + "/branches" },
			body: func() any {
				return map[string]any{"name": "matrix-" + unique(t), "type": "physical"}
			},
			allow: managers,
		},
		{
			name: "list members", method: http.MethodGet,
			path:  func() string { return "/api/v1/organizations/" + w.orgA + "/members" },
			allow: all,
		},
		{
			name: "invite member", method: http.MethodPost,
			path: func() string { return "/api/v1/organizations/" + w.orgA + "/members" },
			body: func() any {
				return map[string]string{
					"email": "invitee-" + unique(t) + "@audit.example", "role_key": "member",
				}
			},
			allow: ownerOnly,
		},
		{
			name: "create offering", method: http.MethodPost,
			path: func() string { return "/api/v1/organizations/" + w.orgA + "/offerings" },
			body: func() any {
				return map[string]any{
					"name": "matrix", "slug": "matrix-" + unique(t),
					"taxonomy_id": w.alice.firstTaxonomyID(),
				}
			},
			allow: catalogWriters,
		},
		{
			name: "publish offering", method: http.MethodPost,
			path: func() string {
				return "/api/v1/organizations/" + w.orgA + "/offerings/" + target() + "/status"
			},
			body:  func() any { return map[string]string{"status": "published"} },
			allow: catalogPublishers,
		},
	}

	actors := map[string]*client{
		"owner":   w.alice,
		"manager": w.mia,
		"staff":   w.sam,
		"member":  w.uma,
	}

	for _, row := range rows {
		row := row
		for role, expect := range row.allow {
			role, expect := role, expect
			t.Run(row.name+"/"+role, func(t *testing.T) {
				var body any
				if row.body != nil {
					body = row.body()
				}
				status, _, raw := actors[role].do(row.method, row.path(), body, nil)
				if expect && !isSuccess(status) {
					t.Errorf("status %d, want 2xx (body: %s)", status, truncate(raw))
				}
				if !expect && isSuccess(status) {
					t.Errorf("status %d, want a denial (body: %s)", status, truncate(raw))
				}
				if !expect && status != http.StatusForbidden && status != http.StatusNotFound {
					t.Errorf("denial status %d, want 403 or 404 (body: %s)", status, truncate(raw))
				}
			})
		}
	}
}

// TestOwnerOnlyActions checks the two operations whose permission key is not
// enough on its own: ownership transfer and removal of the owner.
func TestOwnerOnlyActions(t *testing.T) {
	w := setup(t)

	// A manager holds `organization.manage` but must not seize ownership.
	status, _, body := w.mia.do(http.MethodPost, "/api/v1/organizations/"+w.orgA+"/transfer-ownership",
		map[string]string{"new_owner_id": w.mia.userID()}, nil)
	mustStatus(t, "manager transfer ownership to self", status, http.StatusForbidden, body)

	// The owner must not have changed.
	var org struct {
		OwnerID string `json:"owner_id"`
	}
	w.alice.decode("/api/v1/organizations/"+w.orgA, &org)
	if org.OwnerID != w.alice.userID() {
		t.Fatalf("owner changed to %s after a denied transfer", org.OwnerID)
	}

	// A manager must not remove the owner.
	status, _, body = w.mia.do(http.MethodDelete,
		"/api/v1/organizations/"+w.orgA+"/members/"+w.ownerMembershipID(t), nil, nil)
	if isSuccess(status) {
		t.Fatalf("manager removed the owner (status %d)", status)
	}
	denied(t, "manager removes owner", status, body)
}

// ownerMembershipID finds the membership row of the organization's owner.
func (w *world) ownerMembershipID(t *testing.T) string {
	t.Helper()
	var members []struct {
		ID      string `json:"id"`
		RoleKey string `json:"role_key"`
	}
	w.alice.decode("/api/v1/organizations/"+w.orgA+"/members", &members)
	for _, m := range members {
		if m.RoleKey == "owner" {
			return m.ID
		}
	}
	t.Fatal("owner membership not found")
	return ""
}

// TestPermissionRemovalTakesEffectImmediately is the "revoked but still signed
// in" case. The access token stays valid for its TTL, so authorization must be
// resolved from the database on every request rather than trusted from the JWT.
func TestPermissionRemovalTakesEffectImmediately(t *testing.T) {
	w := setup(t)

	status, _, body := w.mia.do(http.MethodPatch, "/api/v1/organizations/"+w.orgA,
		map[string]string{"description": "before"}, nil)
	mustStatus(t, "manager edit before removal", status, http.StatusOK, body)

	// Demote with the same access token still in Mia's hands.
	status, _, body = w.alice.do(http.MethodPatch,
		"/api/v1/organizations/"+w.orgA+"/members/"+w.memberMia,
		map[string]string{"role_key": "member"}, nil)
	mustStatus(t, "demote manager", status, http.StatusOK, body)

	status, _, body = w.mia.do(http.MethodPatch, "/api/v1/organizations/"+w.orgA,
		map[string]string{"description": "after"}, nil)
	if isSuccess(status) {
		t.Fatalf("a demoted manager can still edit the organization (status %d)", status)
	}
	denied(t, "demoted manager edit", status, body)

	status, _, _ = w.mia.do(http.MethodGet, "/api/v1/organizations/"+w.orgA, nil, nil)
	mustStatus(t, "member can still read", status, http.StatusOK, "")
}

// TestRemovedMemberLosesAccess covers full removal rather than demotion.
func TestRemovedMemberLosesAccess(t *testing.T) {
	w := setup(t)

	status, _, body := w.uma.do(http.MethodGet, "/api/v1/organizations/"+w.orgA+"/branches", nil, nil)
	mustStatus(t, "member reads before removal", status, http.StatusOK, body)

	status, _, body = w.alice.do(http.MethodDelete,
		"/api/v1/organizations/"+w.orgA+"/members/"+w.memberUma, nil, nil)
	mustStatus(t, "remove member", status, http.StatusNoContent, body)

	status, _, body = w.uma.do(http.MethodGet, "/api/v1/organizations/"+w.orgA+"/branches", nil, nil)
	if isSuccess(status) {
		t.Fatalf("a removed member still reads the tenant (status %d)", status)
	}
	denied(t, "removed member read", status, body)
}

// TestPermissionsEndpointIsScoped proves /authz/me/permissions answers for the
// requested tenant rather than globally.
func TestPermissionsEndpointIsScoped(t *testing.T) {
	w := setup(t)

	var own struct {
		Permissions []string `json:"permissions"`
	}
	w.alice.decode("/api/v1/authz/me/permissions?scope=organization&scope_id="+w.orgA, &own)
	if len(own.Permissions) == 0 {
		t.Fatal("owner has no permissions in their own organization")
	}

	var foreign struct {
		Permissions []string `json:"permissions"`
	}
	w.alice.decode("/api/v1/authz/me/permissions?scope=organization&scope_id="+w.orgB, &foreign)
	if len(foreign.Permissions) != 0 {
		t.Fatalf("permissions leaked for another tenant: %v", foreign.Permissions)
	}
}

// TestPlatformAdminBoundary checks that a normal user cannot reach the admin
// surface.
func TestPlatformAdminBoundary(t *testing.T) {
	w := setup(t)

	for _, path := range []string{"/api/v1/admin/users", "/api/v1/authz/admin/roles"} {
		status, _, body := w.alice.do(http.MethodGet, path, nil, nil)
		if isSuccess(status) {
			t.Fatalf("normal user reached %s (status %d)", path, status)
		}
		denied(t, path, status, body)
	}
}

// TestOversizedBodyIsRejected checks the body limit without transferring a huge
// payload.
//
// The request declares a Content-Length above the limit and sends no body. A
// server that enforces the limit from the header answers 413 immediately; one
// that only checks while streaming waits for bytes that never arrive, which the
// deadline turns into a failure instead of a hang. Sending 9 MB through Docker
// Desktop's port proxy is not worth the risk: it has been observed to wedge the
// forwarder for every subsequent request.
func TestOversizedBodyIsRejected(t *testing.T) {
	w := setup(t)

	host := strings.TrimPrefix(w.base, "http://")
	host = strings.TrimPrefix(host, "https://")

	conn, err := net.DialTimeout("tcp", host, 5*time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

	const declared = 9 << 20 // 9 MiB, above the configured 8M limit
	request := strings.Join([]string{
		"POST /api/v1/organizations/" + w.orgA + "/offerings HTTP/1.1",
		"Host: " + host,
		"Authorization: Bearer " + w.alice.token,
		"Content-Type: application/json",
		fmt.Sprintf("Content-Length: %d", declared),
		"Connection: close",
		"",
		"",
	}, "\r\n")

	if _, err := conn.Write([]byte(request)); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Read just the status line. Waiting for the whole response would block,
	// because the server may keep the connection open for the body it expects.
	buf := make([]byte, 64)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("the server did not reject the declared size: %v", err)
	}
	statusLine := string(buf[:n])
	if !strings.Contains(statusLine, "413") && !strings.Contains(statusLine, "400") {
		t.Fatalf("status = %q, want 413 for a body above the limit", strings.SplitN(statusLine, "\r\n", 2)[0])
	}
}
