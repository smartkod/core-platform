// Package api holds the organization HTTP layer.
package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	"core-platform/contract/core"
	"core-platform/mod/organization/service"
	"core-platform/platform/httpx"
)

// Handler serves the organization endpoints.
type Handler struct {
	svc    *service.Organizations
	authz  core.Authorizer
	public echo.MiddlewareFunc
}

// NewHandler builds the handler. public is the identity optional-middleware so
// discovery can be personalized without requiring a session.
func NewHandler(svc *service.Organizations, authz core.Authorizer, public echo.MiddlewareFunc) *Handler {
	return &Handler{svc: svc, authz: authz, public: public}
}

// Register mounts the routes.
func (h *Handler) Register(g *echo.Group, id core.Identity) {
	if h.public != nil {
		g.GET("/organizations", h.list, h.public)
		g.GET("/organizations/by-slug/:slug", h.bySlug, h.public)
	} else {
		g.GET("/organizations", h.list)
		g.GET("/organizations/by-slug/:slug", h.bySlug)
	}
	g.GET("/regions", h.regions)

	authed := g.Group("", id.Middleware())

	// Creating an organization is the one action that has no tenant yet; the
	// owner is taken from the session.
	authed.POST("/organizations", h.create)
	authed.POST("/invitations/:id/accept", h.acceptInvitation)

	// Tenant-scoped routes. The authorizer reads :org_id from the path, so a
	// client cannot act on an organization it merely names in a body.
	org := authed.Group("/organizations/:org_id")
	org.GET("", h.byID, h.authz.Require(core.ActionOrganizationRead, core.ScopeOrganization))
	org.PATCH("", h.update, h.authz.Require(core.ActionOrganizationManage, core.ScopeOrganization))
	org.POST("/transfer-ownership", h.transferOwnership,
		h.authz.Require(core.ActionOrganizationManage, core.ScopeOrganization))

	org.GET("/branches", h.branches, h.authz.Require(core.ActionOrganizationRead, core.ScopeOrganization))
	org.POST("/branches", h.createBranch, h.authz.Require(core.ActionBranchManage, core.ScopeOrganization))
	org.PUT("/branches/:branch_id", h.updateBranch, h.authz.Require(core.ActionBranchManage, core.ScopeOrganization))
	org.DELETE("/branches/:branch_id", h.deleteBranch, h.authz.Require(core.ActionBranchManage, core.ScopeOrganization))

	org.GET("/members", h.members, h.authz.Require(core.ActionOrganizationRead, core.ScopeOrganization))
	org.POST("/members", h.invite, h.authz.Require(core.ActionMemberManage, core.ScopeOrganization))
	org.PATCH("/members/:membership_id", h.updateMember, h.authz.Require(core.ActionMemberManage, core.ScopeOrganization))
	org.DELETE("/members/:membership_id", h.removeMember, h.authz.Require(core.ActionMemberManage, core.ScopeOrganization))

	org.GET("/service-areas", h.serviceAreas, h.authz.Require(core.ActionOrganizationRead, core.ScopeOrganization))
	org.POST("/service-areas", h.addServiceArea, h.authz.Require(core.ActionOrganizationManage, core.ScopeOrganization))
	org.DELETE("/service-areas/:area_id", h.removeServiceArea, h.authz.Require(core.ActionOrganizationManage, core.ScopeOrganization))

	// Platform-only publishing decision.
	admin := authed.Group("/admin/organizations/:org_id", id.RequireRole(core.RoleAdmin))
	admin.POST("/status", h.setStatus,
		h.authz.Require(core.ActionOrganizationPublish, core.ScopeOrganization))
}

// --- Public discovery ----------------------------------------------------

func (h *Handler) list(c echo.Context) error {
	limit, offset := pageParams(c)
	list, total, err := h.svc.List(c.Request().Context(), core.OrganizationFilter{
		Status:   core.OrgStatusActive,
		RegionID: c.QueryParam("region_id"),
		Query:    c.QueryParam("q"),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return err
	}
	return httpx.RespondList(c, list, total, limit, offset)
}

func (h *Handler) bySlug(c echo.Context) error {
	org, err := h.svc.BySlug(c.Request().Context(), c.Param("slug"))
	if err != nil {
		return err
	}
	// A draft is not a public object: answering 404 rather than 403 avoids
	// confirming that an unpublished slug exists.
	if org.Status != core.OrgStatusActive {
		return core.ErrNotFound
	}
	return httpx.RespondOK(c, http.StatusOK, org)
}

func (h *Handler) regions(c echo.Context) error {
	list, err := h.svc.Regions(c.Request().Context(), c.QueryParam("parent_id"))
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusOK, list)
}

// --- Organization --------------------------------------------------------

func (h *Handler) create(c echo.Context) error {
	p := core.UserFrom(c)
	if p == nil {
		return core.ErrUnauthorized
	}

	var in struct {
		Name    string `json:"name"`
		Slug    string `json:"slug"`
		Email   string `json:"email"`
		Phone   string `json:"phone"`
		Website string `json:"website"`
		Locale  string `json:"locale"`
	}
	if err := c.Bind(&in); err != nil {
		return core.ErrValidation
	}

	org, err := h.svc.Create(c.Request().Context(), core.CreateOrganizationInput{
		Name:    in.Name,
		Slug:    in.Slug,
		Email:   in.Email,
		Phone:   in.Phone,
		Website: in.Website,
		Locale:  in.Locale,
		OwnerID: p.UserID,
	})
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusCreated, org)
}

func (h *Handler) byID(c echo.Context) error {
	org, err := h.svc.ByID(c.Request().Context(), c.Param("org_id"))
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusOK, org)
}

func (h *Handler) update(c echo.Context) error {
	var in struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		LogoURL     *string `json:"logo_url"`
		Email       *string `json:"email"`
		Phone       *string `json:"phone"`
		Website     *string `json:"website"`
		Locale      *string `json:"locale"`
	}
	if err := c.Bind(&in); err != nil {
		return core.ErrValidation
	}

	org, err := h.svc.Update(c.Request().Context(), core.UpdateOrganizationInput{
		ID:          c.Param("org_id"),
		Name:        in.Name,
		Description: in.Description,
		LogoURL:     in.LogoURL,
		Email:       in.Email,
		Phone:       in.Phone,
		Website:     in.Website,
		Locale:      in.Locale,
	})
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusOK, org)
}

func (h *Handler) setStatus(c echo.Context) error {
	p := core.UserFrom(c)
	var in struct {
		Status string `json:"status"`
	}
	if err := c.Bind(&in); err != nil {
		return core.ErrValidation
	}

	org, err := h.svc.SetStatus(c.Request().Context(), c.Param("org_id"), in.Status, p.UserID)
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusOK, org)
}

func (h *Handler) transferOwnership(c echo.Context) error {
	p := core.UserFrom(c)
	var in struct {
		NewOwnerID string `json:"new_owner_id"`
	}
	if err := c.Bind(&in); err != nil {
		return core.ErrValidation
	}
	if err := h.svc.TransferOwnership(c.Request().Context(), c.Param("org_id"), in.NewOwnerID, p.UserID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// --- Branches ------------------------------------------------------------

func (h *Handler) branches(c echo.Context) error {
	list, err := h.svc.Branches(c.Request().Context(), c.Param("org_id"))
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusOK, list)
}

func (h *Handler) createBranch(c echo.Context) error {
	var in branchBody
	if err := c.Bind(&in); err != nil {
		return core.ErrValidation
	}
	b, err := h.svc.CreateBranch(c.Request().Context(), in.toInput(c.Param("org_id")))
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusCreated, b)
}

func (h *Handler) updateBranch(c echo.Context) error {
	var in branchBody
	if err := c.Bind(&in); err != nil {
		return core.ErrValidation
	}
	input := in.toInput(c.Param("org_id"))
	input.ID = c.Param("branch_id")
	b, err := h.svc.UpdateBranch(c.Request().Context(), input)
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusOK, b)
}

func (h *Handler) deleteBranch(c echo.Context) error {
	p := core.UserFrom(c)
	b, err := h.svc.Branch(c.Request().Context(), c.Param("branch_id"))
	if err != nil {
		return err
	}
	if b.OrganizationID != c.Param("org_id") {
		return core.ErrNotFound
	}
	if err := h.svc.DeleteBranch(c.Request().Context(), b.ID, p.UserID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

type branchBody struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	AddressLine string  `json:"address_line"`
	Country     string  `json:"country"`
	City        string  `json:"city"`
	District    string  `json:"district"`
	PostalCode  string  `json:"postal_code"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Phone       string  `json:"phone"`
	Public      bool    `json:"public"`
	Status      string  `json:"status"`
}

func (b branchBody) toInput(organizationID string) core.BranchInput {
	return core.BranchInput{
		OrganizationID: organizationID,
		Name:           b.Name,
		Type:           b.Type,
		AddressLine:    b.AddressLine,
		Country:        b.Country,
		City:           b.City,
		District:       b.District,
		PostalCode:     b.PostalCode,
		Latitude:       b.Latitude,
		Longitude:      b.Longitude,
		Phone:          b.Phone,
		Public:         b.Public,
		Status:         b.Status,
	}
}

// --- Members -------------------------------------------------------------

func (h *Handler) members(c echo.Context) error {
	list, err := h.svc.Members(c.Request().Context(), c.Param("org_id"))
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusOK, list)
}

func (h *Handler) invite(c echo.Context) error {
	p := core.UserFrom(c)
	var in struct {
		Email    string `json:"email"`
		RoleKey  string `json:"role_key"`
		BranchID string `json:"branch_id"`
	}
	if err := c.Bind(&in); err != nil {
		return core.ErrValidation
	}

	m, err := h.svc.InviteMember(c.Request().Context(), core.InviteMemberInput{
		OrganizationID: c.Param("org_id"),
		BranchID:       in.BranchID,
		Email:          in.Email,
		RoleKey:        in.RoleKey,
		InvitedBy:      p.UserID,
	})
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusCreated, m)
}

func (h *Handler) updateMember(c echo.Context) error {
	p := core.UserFrom(c)
	var in struct {
		RoleKey string `json:"role_key"`
	}
	if err := c.Bind(&in); err != nil {
		return core.ErrValidation
	}

	m, err := h.svc.UpdateMemberRole(c.Request().Context(), c.Param("membership_id"), in.RoleKey, p.UserID)
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusOK, m)
}

func (h *Handler) removeMember(c echo.Context) error {
	p := core.UserFrom(c)
	m, err := h.svc.Membership(c.Request().Context(), c.Param("membership_id"))
	if err != nil {
		return err
	}
	if m.OrganizationID != c.Param("org_id") {
		return core.ErrNotFound
	}
	if err := h.svc.RemoveMember(c.Request().Context(), m.ID, p.UserID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// acceptInvitation lets the signed-in user accept an invitation addressed to
// their own address. The e-mail comparison happens before the membership is
// changed, so guessing an id reveals nothing and grants nothing.
func (h *Handler) acceptInvitation(c echo.Context) error {
	p := core.UserFrom(c)
	if p == nil {
		return core.ErrUnauthorized
	}

	invitation, err := h.svc.Membership(c.Request().Context(), c.Param("id"))
	if err != nil {
		return err
	}
	if invitation.Status != core.MemberStatusInvited {
		return core.StateError("the invitation is no longer pending")
	}
	if invitation.InvitedEmail != "" && !strings.EqualFold(invitation.InvitedEmail, p.Email) {
		return core.ErrForbidden
	}

	accepted, err := h.svc.AcceptInvitation(c.Request().Context(), c.Param("id"), p.UserID)
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusOK, accepted)
}

// --- Service areas -------------------------------------------------------

func (h *Handler) serviceAreas(c echo.Context) error {
	list, err := h.svc.ServiceAreas(c.Request().Context(), c.Param("org_id"))
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusOK, list)
}

func (h *Handler) addServiceArea(c echo.Context) error {
	var in struct {
		BranchID string `json:"branch_id"`
		RegionID string `json:"region_id"`
		Mode     string `json:"mode"`
	}
	if err := c.Bind(&in); err != nil {
		return core.ErrValidation
	}

	area, err := h.svc.AddServiceArea(c.Request().Context(), core.ServiceArea{
		OrganizationID: c.Param("org_id"),
		BranchID:       in.BranchID,
		RegionID:       in.RegionID,
		Mode:           in.Mode,
	})
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusCreated, area)
}

func (h *Handler) removeServiceArea(c echo.Context) error {
	p := core.UserFrom(c)
	area, err := h.svc.ServiceArea(c.Request().Context(), c.Param("area_id"))
	if err != nil {
		return err
	}
	if area.OrganizationID != c.Param("org_id") {
		return core.ErrNotFound
	}
	if err := h.svc.RemoveServiceArea(c.Request().Context(), area.ID, p.UserID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// --- Helpers -------------------------------------------------------------

func pageParams(c echo.Context) (limit, offset int) {
	limit = 20
	if v := c.QueryParam("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
			if limit > 100 {
				limit = 100
			}
		}
	}
	if v := c.QueryParam("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return limit, offset
}
