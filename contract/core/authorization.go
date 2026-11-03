package core

import (
	"context"
	"time"

	"github.com/labstack/echo/v4"
)

// ScopeType is the level at which a permission applies.
//
// A permission is never granted globally by accident: an assignment always
// names a scope and the identifier inside it. This is what makes "manager of
// company A, employee of company B" expressible without a second role system.
type ScopeType string

// Scope levels, from widest to narrowest.
const (
	// ScopePlatform is the whole platform (administrators, moderators).
	ScopePlatform ScopeType = "platform"
	// ScopeOrganization is one business or organization.
	ScopeOrganization ScopeType = "organization"
	// ScopeBranch is one branch/location of an organization.
	ScopeBranch ScopeType = "branch"
	// ScopeSelf is the actor's own record only.
	ScopeSelf ScopeType = "self"
)

// ValidScopeType reports whether s is one of the defined scope levels.
func ValidScopeType(s ScopeType) bool {
	switch s {
	case ScopePlatform, ScopeOrganization, ScopeBranch, ScopeSelf:
		return true
	}
	return false
}

// Permission is a named capability, e.g. "catalog.offering.publish".
//
// Keys are namespaced by the module that owns them. The core owns the keys it
// checks itself; an extension or engine registers its own keys under its
// namespace prefix and never has to change the core.
type Permission struct {
	Key         string `json:"key"`
	Description string `json:"description"`
	// Scope is the narrowest scope the permission may be granted at.
	Scope ScopeType `json:"scope"`
}

// Role is a named bundle of permissions.
//
// A role is not a hierarchy: it is a set. Two roles held in the same scope add
// up, and holding a role in one organization says nothing about any other.
type Role struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	// System marks a built-in role that cannot be deleted.
	System bool `json:"system"`
	// Permissions are the keys granted by this role.
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
}

// Assignment grants one role over one scope to one user.
type Assignment struct {
	ID      string    `json:"id"`
	UserID  string    `json:"user_id"`
	RoleKey string    `json:"role_key"`
	Scope   ScopeType `json:"scope"`
	ScopeID string    `json:"scope_id"`
	// GrantedBy is the actor that created the grant, for the audit trail.
	GrantedBy string     `json:"granted_by,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// Check is one authorization question.
//
// The caller resolves the resource first, then states which scope the action
// falls under. A client-supplied scope id is never trusted on its own: the
// handler loads the resource and derives the scope from the stored record.
type Check struct {
	UserID  string    `json:"user_id"`
	Action  string    `json:"action"`
	Scope   ScopeType `json:"scope"`
	ScopeID string    `json:"scope_id"`
}

// GrantInput creates an assignment.
type GrantInput struct {
	UserID    string
	RoleKey   string
	Scope     ScopeType
	ScopeID   string
	GrantedBy string
	ExpiresAt *time.Time
}

// Authorizer is the platform's single authorization surface.
//
// The default answer is "no": a check that matches no active assignment is
// denied. Implementations must evaluate assignments on the server side and must
// never accept an "is allowed" flag from the request.
type Authorizer interface {
	// Authorize returns nil when the actor may perform the action in the
	// given scope, and ErrForbidden otherwise.
	Authorize(ctx context.Context, c Check) error
	// Allowed is Authorize without the error, for UI hints. It must not be
	// used as the only guard on a protected action.
	Allowed(ctx context.Context, c Check) (bool, error)
	// Permissions returns the effective permission keys a user holds in a
	// scope. Platform grants apply to every scope; an organization grant never
	// widens to the platform scope.
	Permissions(ctx context.Context, userID string, scope ScopeType, scopeID string) ([]string, error)
	// Assignments lists a user's active grants.
	Assignments(ctx context.Context, userID string) ([]Assignment, error)
	// Grant adds a role assignment.
	Grant(ctx context.Context, in GrantInput) (*Assignment, error)
	// Revoke disables an assignment. It never deletes the row, so the history
	// stays auditable.
	Revoke(ctx context.Context, assignmentID, revokedBy string) error
	// Roles lists the roles defined on the platform.
	Roles(ctx context.Context) ([]Role, error)
	// UpsertRole creates or replaces a role definition.
	UpsertRole(ctx context.Context, role Role) error
	// RegisterPermission records a permission key so it can be granted.
	RegisterPermission(ctx context.Context, p Permission) error
	// Require is HTTP middleware for one action and scope.
	//
	// The scope id is taken from a route parameter by convention:
	// :org_id for organization, :branch_id for branch, and the authenticated
	// user id for self. Anything else is a programming error and is denied.
	Require(action string, scope ScopeType) echo.MiddlewareFunc
}

// Action keys owned by the core. Modules register their own under their
// namespace; these are the ones the core checks itself.
const (
	ActionOrganizationRead    = "organization.read"
	ActionOrganizationManage  = "organization.manage"
	ActionOrganizationPublish = "organization.publish"
	ActionBranchManage        = "org.branch.manage"
	ActionMemberManage        = "org.member.manage"
	ActionCatalogRead         = "catalog.read"
	ActionCatalogManage       = "catalog.manage"
	ActionCatalogPublish      = "catalog.publish"
	ActionTaxonomyManage      = "taxonomy.manage"
	ActionReviewRespond       = "review.respond"
	ActionPlatformAdmin       = "platform.admin"
	ActionModerationReview    = "moderation.review"
)
