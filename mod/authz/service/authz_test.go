package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"core-platform/contract/core"
)

// The default answer must be no. An empty identity is the easiest way to check
// that without a database: Allowed short-circuits before touching the store.
func TestAllowedDeniesIncompleteChecks(t *testing.T) {
	a := New(nil, nil)

	cases := []core.Check{
		{},
		{UserID: "u1"},
		{UserID: "u1", Action: core.ActionCatalogRead},
		{UserID: "u1", Action: core.ActionCatalogRead, Scope: core.ScopeOrganization},
		{UserID: "u1", Action: core.ActionCatalogRead, Scope: "made-up", ScopeID: "o1"},
	}

	for _, c := range cases {
		ok, err := a.Allowed(context.Background(), c)
		if err != nil {
			t.Fatalf("Allowed(%+v): %v", c, err)
		}
		if ok {
			t.Fatalf("Allowed(%+v) = true, want false", c)
		}
	}
}

func TestScopeIDFor(t *testing.T) {
	e := echo.New()
	principal := &core.Principal{UserID: "user-1"}

	req := httptest.NewRequest(http.MethodGet, "/organizations/org-1/branches/branch-1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("org_id", "branch_id")
	c.SetParamValues("org-1", "branch-1")

	orgID, err := scopeIDFor(c, principal, core.ScopeOrganization)
	if err != nil || orgID != "org-1" {
		t.Fatalf("organization scope = %q, %v", orgID, err)
	}

	branchID, err := scopeIDFor(c, principal, core.ScopeBranch)
	if err != nil || branchID != "branch-1" {
		t.Fatalf("branch scope = %q, %v", branchID, err)
	}

	selfID, err := scopeIDFor(c, principal, core.ScopeSelf)
	if err != nil || selfID != "user-1" {
		t.Fatalf("self scope = %q, %v", selfID, err)
	}

	platformID, err := scopeIDFor(c, principal, core.ScopePlatform)
	if err != nil || platformID != "" {
		t.Fatalf("platform scope = %q, %v", platformID, err)
	}
}

func TestScopeIDMissingParamIsForbidden(t *testing.T) {
	e := echo.New()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())

	_, err := scopeIDFor(c, &core.Principal{UserID: "u1"}, core.ScopeOrganization)
	if !errors.Is(err, core.ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

// fakeUsers is the smallest possible UserStore: it exists to prove the platform
// role fallback without a database.
type fakeUsers struct{ user *core.User }

func (f fakeUsers) GetUser(context.Context, string) (*core.User, error) { return f.user, nil }

func (f fakeUsers) FindUserByEmail(context.Context, string) (*core.User, error) {
	return f.user, nil
}

func TestPlatformRoleFallback(t *testing.T) {
	check := core.Check{
		UserID:  "u1",
		Action:  core.ActionOrganizationPublish,
		Scope:   core.ScopeOrganization,
		ScopeID: "org-1",
	}

	admin := New(nil, fakeUsers{user: &core.User{ID: "u1", Role: core.RoleAdmin}})
	if ok, err := admin.Allowed(context.Background(), check); err != nil || !ok {
		t.Fatalf("admin allowed = %v, %v", ok, err)
	}

	moderator := New(nil, fakeUsers{user: &core.User{ID: "u1", Role: core.RoleModerator}})
	// A moderator may read and review, but may not publish an organization.
	if ok, _ := moderator.Allowed(context.Background(), check); ok {
		t.Fatal("moderator must not publish an organization")
	}
	read := check
	read.Action = core.ActionOrganizationRead
	if ok, err := moderator.Allowed(context.Background(), read); err != nil || !ok {
		t.Fatalf("moderator read = %v, %v", ok, err)
	}
}
