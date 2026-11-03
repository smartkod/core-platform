// Package api holds the authorization HTTP layer.
package api

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"core-platform/contract/core"
	"core-platform/mod/authz/service"
	"core-platform/platform/httpx"
)

// Handler serves the authorization endpoints.
type Handler struct {
	svc *service.Authz
}

// NewHandler builds the handler.
func NewHandler(svc *service.Authz) *Handler { return &Handler{svc: svc} }

// Register mounts the routes.
//
// Reading and writing role definitions is a platform duty; granting a role in
// an organization is checked by the organization module against its own
// membership rules before it calls Grant.
func (h *Handler) Register(g *echo.Group, id core.Identity) {
	authed := g.Group("/authz", id.Middleware())

	authed.GET("/me/assignments", h.myAssignments)
	authed.GET("/me/permissions", h.myPermissions)

	admin := authed.Group("/admin", id.RequireRole(core.RoleAdmin))
	admin.GET("/roles", h.listRoles)
	admin.POST("/roles", h.upsertRole)
	admin.GET("/users/:user_id/assignments", h.userAssignments)
	admin.POST("/grants", h.grant)
	admin.DELETE("/grants/:id", h.revoke)
}

func (h *Handler) myAssignments(c echo.Context) error {
	p := core.UserFrom(c)
	if p == nil {
		return core.ErrUnauthorized
	}
	list, err := h.svc.Assignments(c.Request().Context(), p.UserID)
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusOK, list)
}

func (h *Handler) myPermissions(c echo.Context) error {
	p := core.UserFrom(c)
	if p == nil {
		return core.ErrUnauthorized
	}

	scope := core.ScopeType(c.QueryParam("scope"))
	if scope == "" {
		scope = core.ScopeOrganization
	}
	scopeID := c.QueryParam("scope_id")

	keys, err := h.svc.Permissions(c.Request().Context(), p.UserID, scope, scopeID)
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusOK, map[string]any{
		"scope":       scope,
		"scope_id":    scopeID,
		"permissions": keys,
	})
}

func (h *Handler) listRoles(c echo.Context) error {
	roles, err := h.svc.Roles(c.Request().Context())
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusOK, roles)
}

func (h *Handler) upsertRole(c echo.Context) error {
	var in struct {
		Key         string   `json:"key"`
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Permissions []string `json:"permissions"`
	}
	if err := c.Bind(&in); err != nil {
		return core.ErrValidation
	}

	if err := h.svc.UpsertRole(c.Request().Context(), core.Role{
		Key:         in.Key,
		Name:        in.Name,
		Description: in.Description,
		Permissions: in.Permissions,
	}); err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusOK, map[string]string{"key": in.Key})
}

func (h *Handler) userAssignments(c echo.Context) error {
	userID := c.Param("user_id")
	if userID == "" {
		return core.ErrValidation
	}
	list, err := h.svc.Assignments(c.Request().Context(), userID)
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusOK, list)
}

func (h *Handler) grant(c echo.Context) error {
	var in struct {
		UserID    string `json:"user_id"`
		RoleKey   string `json:"role_key"`
		Scope     string `json:"scope"`
		ScopeID   string `json:"scope_id"`
		ExpiresAt string `json:"expires_at"`
	}
	if err := c.Bind(&in); err != nil {
		return core.ErrValidation
	}

	p := core.UserFrom(c)
	grant := core.GrantInput{
		UserID:    in.UserID,
		RoleKey:   in.RoleKey,
		Scope:     core.ScopeType(in.Scope),
		ScopeID:   in.ScopeID,
		GrantedBy: p.UserID,
	}
	if in.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, in.ExpiresAt)
		if err != nil {
			return core.Invalid("expires_at", "expected an RFC3339 timestamp")
		}
		grant.ExpiresAt = &t
	}

	a, err := h.svc.Grant(c.Request().Context(), grant)
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusCreated, a)
}

func (h *Handler) revoke(c echo.Context) error {
	p := core.UserFrom(c)
	if p == nil {
		return core.ErrUnauthorized
	}
	if err := h.svc.Revoke(c.Request().Context(), c.Param("id"), p.UserID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
