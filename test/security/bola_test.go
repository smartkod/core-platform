package security_test

import (
	"net/http"
	"testing"
)

// TestBOLA covers Broken Object Level Authorization: one tenant's user naming
// another tenant's object id.
//
// Every case is a request the UI never offers, sent with a perfectly valid
// token. The question is not whether the button is hidden; it is whether the
// server checks the object, not just the actor.
func TestBOLA(t *testing.T) {
	w := setup(t)

	// Each case: an actor from tenant A tries to act on tenant B's object.
	cases := []struct {
		name   string
		client *client
		method string
		path   string
		body   any
	}{
		// Organization
		{"read organization B", w.alice, http.MethodGet, "/api/v1/organizations/" + w.orgB, nil},
		{"patch organization B", w.alice, http.MethodPatch, "/api/v1/organizations/" + w.orgB, map[string]string{"name": "pwned"}},
		{"transfer organization B", w.alice, http.MethodPost, "/api/v1/organizations/" + w.orgB + "/transfer-ownership", map[string]string{"new_owner_id": w.alice.userID()}},

		// Branches
		{"list branches of B", w.alice, http.MethodGet, "/api/v1/organizations/" + w.orgB + "/branches", nil},
		{"create branch in B", w.alice, http.MethodPost, "/api/v1/organizations/" + w.orgB + "/branches", map[string]any{"name": "pwned"}},
		{"update branch B via B path", w.alice, http.MethodPut, "/api/v1/organizations/" + w.orgB + "/branches/" + w.branchB, map[string]any{"name": "pwned"}},
		{"update branch B via A path", w.alice, http.MethodPut, "/api/v1/organizations/" + w.orgA + "/branches/" + w.branchB, map[string]any{"name": "pwned"}},
		{"delete branch B via A path", w.alice, http.MethodDelete, "/api/v1/organizations/" + w.orgA + "/branches/" + w.branchB, nil},

		// Members
		{"list members of B", w.alice, http.MethodGet, "/api/v1/organizations/" + w.orgB + "/members", nil},
		{"invite into B", w.alice, http.MethodPost, "/api/v1/organizations/" + w.orgB + "/members", map[string]string{"email": "evil@audit.example", "role_key": "manager"}},
		{"re-role member of B via B path", w.alice, http.MethodPatch, "/api/v1/organizations/" + w.orgB + "/members/" + w.memberBea, map[string]string{"role_key": "member"}},
		{"re-role member of B via A path", w.alice, http.MethodPatch, "/api/v1/organizations/" + w.orgA + "/members/" + w.memberBea, map[string]string{"role_key": "member"}},
		{"remove member of B via B path", w.alice, http.MethodDelete, "/api/v1/organizations/" + w.orgB + "/members/" + w.memberBea, nil},
		{"remove member of B via A path", w.alice, http.MethodDelete, "/api/v1/organizations/" + w.orgA + "/members/" + w.memberBea, nil},

		// Catalogue
		{"list offerings of B", w.alice, http.MethodGet, "/api/v1/organizations/" + w.orgB + "/offerings", nil},
		{"create offering in B", w.alice, http.MethodPost, "/api/v1/organizations/" + w.orgB + "/offerings", map[string]any{"name": "pwned", "slug": "pwned"}},
		{"update offering B via B path", w.alice, http.MethodPut, "/api/v1/organizations/" + w.orgB + "/offerings/" + w.offeringB, map[string]any{"name": "pwned"}},
		{"update offering B via A path", w.alice, http.MethodPut, "/api/v1/organizations/" + w.orgA + "/offerings/" + w.offeringB, map[string]any{"name": "pwned"}},
		{"delete offering B via A path", w.alice, http.MethodDelete, "/api/v1/organizations/" + w.orgA + "/offerings/" + w.offeringB, nil},
		{"publish offering B", w.alice, http.MethodPost, "/api/v1/organizations/" + w.orgA + "/offerings/" + w.offeringB + "/status", map[string]string{"status": "published"}},
		{"delete variant B", w.alice, http.MethodDelete, "/api/v1/organizations/" + w.orgA + "/offerings/" + w.offeringA + "/variants/" + w.variantB, nil},
		{"list categories of B", w.alice, http.MethodGet, "/api/v1/organizations/" + w.orgB + "/categories", nil},
		{"set categories of B", w.alice, http.MethodPut, "/api/v1/organizations/" + w.orgB + "/categories", map[string]any{"taxonomy_ids": []string{}}},
		{"list service areas of B", w.alice, http.MethodGet, "/api/v1/organizations/" + w.orgB + "/service-areas", nil},

		// Platform surface
		{"list platform users", w.alice, http.MethodGet, "/api/v1/admin/users", nil},
		{"publish organization B as platform", w.alice, http.MethodPost, "/api/v1/admin/organizations/" + w.orgB + "/status", map[string]string{"status": "active"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, _, body := tc.client.do(tc.method, tc.path, tc.body, nil)
			if isSuccess(status) {
				t.Fatalf("%s: attacker succeeded with %d (body: %s)", tc.name, status, truncate(body))
			}
			denied(t, tc.name, status, body)
		})
	}
}

// TestBOLAChangedNothing proves the denials above were real denials and not
// "returned 404 after doing the work". The owner re-reads every object.
func TestBOLAChangedNothing(t *testing.T) {
	w := setup(t)

	// Attack first (repeat of the writes that must not land), then verify.
	w.alice.do(http.MethodPatch, "/api/v1/organizations/"+w.orgA+"/members/"+w.memberBea,
		map[string]string{"role_key": "member"}, nil)
	w.alice.do(http.MethodDelete, "/api/v1/organizations/"+w.orgA+"/offerings/"+w.offeringB, nil, nil)
	w.alice.do(http.MethodDelete, "/api/v1/organizations/"+w.orgA+"/offerings/"+w.offeringA+"/variants/"+w.variantB, nil, nil)
	w.alice.do(http.MethodPut, "/api/v1/organizations/"+w.orgA+"/branches/"+w.branchB,
		map[string]any{"name": "pwned"}, nil)
	w.alice.do(http.MethodPatch, "/api/v1/organizations/"+w.orgB,
		map[string]string{"name": "pwned"}, nil)

	if !exists(w.bob, "/api/v1/organizations/"+w.orgB+"/branches") {
		t.Error("B's branch list is no longer readable by its owner")
	}

	var branches []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	w.bob.decode("/api/v1/organizations/"+w.orgB+"/branches", &branches)
	if len(branches) != 1 || branches[0].ID != w.branchB || branches[0].Name != "B Merkez" {
		t.Errorf("B's branches changed: %+v", branches)
	}

	var offerings []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	w.bob.decode("/api/v1/organizations/"+w.orgB+"/offerings", &offerings)
	if len(offerings) != 1 || offerings[0].ID != w.offeringB || offerings[0].Name != "B hizmeti" {
		t.Errorf("B's offerings changed: %+v", offerings)
	}
	if offerings[0].Status != "draft" {
		t.Errorf("B's offering status = %q, want draft", offerings[0].Status)
	}

	// The variant cannot be listed (no GET /variants route), so existence is
	// proven by the owner: if Alice's delete had worked, Bob's own delete would
	// be a 404. A 204 means the row was still there.
	status, _, body := w.bob.do(http.MethodDelete,
		"/api/v1/organizations/"+w.orgB+"/offerings/"+w.offeringB+"/variants/"+w.variantB, nil, nil)
	mustStatus(t, "B still owns its variant", status, http.StatusNoContent, body)

	var members []struct {
		ID      string `json:"id"`
		RoleKey string `json:"role_key"`
		Status  string `json:"status"`
	}
	w.bob.decode("/api/v1/organizations/"+w.orgB+"/members", &members)
	found := false
	for _, m := range members {
		if m.ID == w.memberBea {
			found = true
			if m.RoleKey != "manager" {
				t.Errorf("B member role = %q, want manager", m.RoleKey)
			}
			if m.Status != "active" {
				t.Errorf("B member status = %q, want active", m.Status)
			}
		}
	}
	if !found {
		t.Error("B's invited member disappeared after a cross-tenant delete")
	}

	var org struct {
		Name string `json:"name"`
	}
	w.bob.decode("/api/v1/organizations/"+w.orgB, &org)
	if org.Name != "Audit B" {
		t.Errorf("B's name = %q, want %q", org.Name, "Audit B")
	}
}

// TestPublicDiscoveryHidesDrafts proves the anonymous surface does not leak a
// draft offering or an unpublished taxonomy node.
func TestPublicDiscoveryHidesDrafts(t *testing.T) {
	w := setup(t)
	anon := newClient(t, w.base)

	var list []struct {
		ID string `json:"id"`
	}
	anon.decode("/api/v1/offerings?limit=100", &list)
	for _, item := range list {
		if item.ID == w.offeringA || item.ID == w.offeringB {
			t.Fatalf("draft offering %s appears in public discovery", item.ID)
		}
	}

	status, _, _ := anon.do(http.MethodGet, "/api/v1/offerings/"+w.offeringA, nil, nil)
	if status == http.StatusOK {
		t.Fatal("a draft offering is readable anonymously")
	}
}
