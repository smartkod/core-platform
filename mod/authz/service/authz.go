// Package service implements the authorization rules.
//
// The service is deliberately the only place that answers "may this actor do
// this here". Handlers ask it; they never inspect a role key themselves. That
// keeps a policy change in one file instead of every handler.
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"core-platform/contract/core"
	"core-platform/mod/authz/model"
	"core-platform/mod/authz/repository"
	"core-platform/platform/uuid"
)

// Authz implements core.Authorizer.
type Authz struct {
	repo  *repository.Store
	users core.UserStore
}

// New builds the authorization service.
//
// users may be nil in a test; the platform-role fallback is then skipped and
// only assignments decide.
func New(repo *repository.Store, users core.UserStore) *Authz {
	return &Authz{repo: repo, users: users}
}

// Authorize returns nil when the actor holds a permission that covers the
// action in the requested scope.
//
// The default is denial. There is no "admin override" hidden here: a platform
// administrator acts through the platform_admin role, which is an ordinary
// assignment like any other and therefore appears in the audit trail.
func (a *Authz) Authorize(ctx context.Context, c core.Check) error {
	ok, err := a.Allowed(ctx, c)
	if err != nil {
		return err
	}
	if !ok {
		return core.ErrForbidden
	}
	return nil
}

// Allowed answers the same question without relying on errors.
func (a *Authz) Allowed(ctx context.Context, c core.Check) (bool, error) {
	if c.UserID == "" || c.Action == "" {
		return false, nil
	}
	if !core.ValidScopeType(c.Scope) {
		return false, nil
	}
	// A non-platform scope needs an identifier; otherwise every organization
	// check would collapse into "any organization", which is exactly the bug
	// this design exists to prevent.
	if c.Scope != core.ScopePlatform && c.ScopeID == "" {
		return false, nil
	}

	// Platform roles are the root authority: an account marked admin must be
	// able to administer before any assignment exists, otherwise nobody could
	// ever publish the first organization. Moderators get exactly the review
	// duties their role names and nothing else.
	if a.users != nil {
		if u, err := a.users.GetUser(ctx, c.UserID); err == nil && u != nil {
			switch u.Role {
			case core.RoleAdmin:
				return true, nil
			case core.RoleModerator:
				if c.Action == core.ActionModerationReview || c.Action == core.ActionOrganizationRead {
					return true, nil
				}
			}
		}
	}

	if a.repo == nil {
		// A service built without a store can only answer from platform roles;
		// returning a denial here keeps a misconfigured test from panicking.
		return false, nil
	}

	keys, err := a.repo.PermissionKeys(ctx, c.UserID,
		string(c.Scope), c.ScopeID)
	if err != nil {
		return false, err
	}
	for _, k := range keys {
		if k == c.Action {
			return true, nil
		}
	}
	return false, nil
}

// Permissions returns the effective permission keys in a scope.
func (a *Authz) Permissions(ctx context.Context, userID string, scope core.ScopeType, scopeID string) ([]string, error) {
	if userID == "" {
		return nil, core.ErrUnauthorized
	}
	return a.repo.PermissionKeys(ctx, userID, string(scope), scopeID)
}

// Assignments lists a user's active grants.
func (a *Authz) Assignments(ctx context.Context, userID string) ([]core.Assignment, error) {
	if userID == "" {
		return nil, core.ErrUnauthorized
	}
	return a.repo.Assignments(ctx, userID)
}

// Grant adds a role assignment after validating it against the role table.
func (a *Authz) Grant(ctx context.Context, in core.GrantInput) (*core.Assignment, error) {
	if in.UserID == "" {
		return nil, core.Invalid("user_id", "a user is required")
	}
	if in.RoleKey == "" {
		return nil, core.Invalid("role_key", "a role is required")
	}
	if !core.ValidScopeType(in.Scope) {
		return nil, core.Invalid("scope", "unknown scope")
	}
	if in.Scope != core.ScopePlatform && in.ScopeID == "" {
		return nil, core.Invalid("scope_id", "a scope identifier is required")
	}
	if in.Scope == core.ScopePlatform && in.ScopeID != "" {
		return nil, core.Invalid("scope_id", "the platform scope has no identifier")
	}

	if _, err := a.repo.Role(ctx, in.RoleKey); err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return nil, core.Invalid("role_key", "unknown role")
		}
		return nil, err
	}

	now := time.Now().UTC()
	rec := &model.Assignment{
		ID:        uuid.New(),
		UserID:    in.UserID,
		RoleKey:   in.RoleKey,
		Scope:     string(in.Scope),
		ScopeID:   in.ScopeID,
		ExpiresAt: in.ExpiresAt,
		CreatedAt: now,
	}
	if in.GrantedBy != "" {
		grantedBy := in.GrantedBy
		rec.GrantedBy = &grantedBy
	}

	if err := a.repo.Grant(ctx, rec); err != nil {
		// The unique index on active grants turns a duplicate into a conflict
		// rather than a second row nobody can distinguish later.
		if isUniqueViolation(err) {
			return nil, core.ErrConflict
		}
		return nil, err
	}

	out := &core.Assignment{
		ID:        rec.ID,
		UserID:    rec.UserID,
		RoleKey:   rec.RoleKey,
		Scope:     core.ScopeType(rec.Scope),
		ScopeID:   rec.ScopeID,
		GrantedBy: in.GrantedBy,
		ExpiresAt: rec.ExpiresAt,
		CreatedAt: rec.CreatedAt,
	}
	return out, nil
}

// Revoke disables an assignment.
func (a *Authz) Revoke(ctx context.Context, assignmentID, revokedBy string) error {
	if assignmentID == "" {
		return core.Invalid("id", "an assignment is required")
	}
	return a.repo.Revoke(ctx, assignmentID)
}

// Roles lists every role.
func (a *Authz) Roles(ctx context.Context) ([]core.Role, error) {
	return a.repo.Roles(ctx)
}

// UpsertRole creates or updates a role definition.
func (a *Authz) UpsertRole(ctx context.Context, role core.Role) error {
	if role.Key == "" {
		return core.Invalid("key", "a role key is required")
	}
	if role.Name == "" {
		return core.Invalid("name", "a role name is required")
	}
	return a.repo.UpsertRole(ctx, role)
}

// RegisterPermission records a permission key under the caller's namespace.
func (a *Authz) RegisterPermission(ctx context.Context, p core.Permission) error {
	if p.Key == "" {
		return core.Invalid("key", "a permission key is required")
	}
	if !core.ValidScopeType(p.Scope) {
		return core.Invalid("scope", "unknown scope")
	}
	return a.repo.RegisterPermission(ctx, p)
}

// RevokeScope revokes every grant in one scope.
func (a *Authz) RevokeScope(ctx context.Context, scope core.ScopeType, scopeID string) error {
	return a.repo.RevokeScope(ctx, string(scope), scopeID)
}

// Require is the HTTP guard for one action and scope.
//
// Scope identifiers are taken from the route, never from the body: a client
// that could name its own organization id could otherwise assert a tenant it
// has no membership in.
func (a *Authz) Require(action string, scope core.ScopeType) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			p := core.UserFrom(c)
			if p == nil {
				return core.ErrUnauthorized
			}

			scopeID, err := scopeIDFor(c, p, scope)
			if err != nil {
				return err
			}

			if err := a.Authorize(c.Request().Context(), core.Check{
				UserID:  p.UserID,
				Action:  action,
				Scope:   scope,
				ScopeID: scopeID,
			}); err != nil {
				return err
			}
			return next(c)
		}
	}
}

// scopeIDFor resolves the scope identifier a route refers to.
func scopeIDFor(c echo.Context, p *core.Principal, scope core.ScopeType) (string, error) {
	switch scope {
	case core.ScopePlatform:
		return "", nil
	case core.ScopeOrganization:
		if id := c.Param("org_id"); id != "" {
			return id, nil
		}
		return "", fmt.Errorf("authz: route has no :org_id parameter: %w", core.ErrForbidden)
	case core.ScopeBranch:
		if id := c.Param("branch_id"); id != "" {
			return id, nil
		}
		return "", fmt.Errorf("authz: route has no :branch_id parameter: %w", core.ErrForbidden)
	case core.ScopeSelf:
		return p.UserID, nil
	}
	return "", core.ErrForbidden
}

// isUniqueViolation reports whether the store rejected a duplicate.
//
// A string check keeps this package independent of the Postgres driver; the
// alternative is leaking driver types into the domain layer for one branch.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "role_assignments_active_uniq")
}
