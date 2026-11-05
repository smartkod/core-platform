// Package api holds the catalog HTTP layer.
package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"core-platform/contract/core"
	"core-platform/mod/catalog/service"
	"core-platform/platform/httpx"
	"core-platform/platform/uuid"
)

// Handler serves the catalog endpoints.
type Handler struct {
	svc   *service.Catalog
	authz core.Authorizer
}

// NewHandler builds the handler.
func NewHandler(svc *service.Catalog, authz core.Authorizer) *Handler {
	return &Handler{svc: svc, authz: authz}
}

// Register mounts the routes.
func (h *Handler) Register(g *echo.Group, id core.Identity, public, write echo.MiddlewareFunc) {
	writeOnly := []echo.MiddlewareFunc{}
	if write != nil {
		writeOnly = append(writeOnly, write)
	}

	// Public discovery.
	if public != nil {
		g.GET("/taxonomy", h.taxonomy, public)
		g.GET("/taxonomy/:slug", h.taxonomyBySlug, public)
		g.GET("/offerings", h.publicOfferings, public)
		g.GET("/offerings/:id", h.publicOffering, public)
	} else {
		g.GET("/taxonomy", h.taxonomy)
		g.GET("/taxonomy/:slug", h.taxonomyBySlug)
		g.GET("/offerings", h.publicOfferings)
		g.GET("/offerings/:id", h.publicOffering)
	}

	authed := g.Group("", id.Middleware())

	// Tenant-scoped catalogue. :org_id is the only tenant the authorizer will
	// accept, and the service re-checks ownership on every write.
	org := authed.Group("/organizations/:org_id",
		h.authz.Require(core.ActionCatalogRead, core.ScopeOrganization))
	org.GET("/offerings", h.organizationOfferings)
	org.GET("/categories", h.organizationCategories)

	manage := authed.Group("/organizations/:org_id",
		h.authz.Require(core.ActionCatalogManage, core.ScopeOrganization))
	manage.POST("/offerings", h.createOffering, writeOnly...)
	manage.PUT("/offerings/:offering_id", h.updateOffering)
	manage.DELETE("/offerings/:offering_id", h.deleteOffering)
	manage.POST("/offerings/:offering_id/variants", h.createVariant, writeOnly...)
	manage.PUT("/offerings/:offering_id/variants/:variant_id", h.updateVariant)
	manage.DELETE("/offerings/:offering_id/variants/:variant_id", h.deleteVariant)
	manage.PUT("/categories", h.setOrganizationCategories)

	publish := authed.Group("/organizations/:org_id",
		h.authz.Require(core.ActionCatalogPublish, core.ScopeOrganization))
	publish.POST("/offerings/:offering_id/status", h.setOfferingStatus)

	// Platform taxonomy management.
	admin := authed.Group("/admin/taxonomy", id.RequireRole(core.RoleAdmin))
	admin.POST("", h.createTaxonomy,
		append([]echo.MiddlewareFunc{h.authz.Require(core.ActionTaxonomyManage, core.ScopePlatform)}, writeOnly...)...)
	admin.PATCH("/:id", h.updateTaxonomy,
		h.authz.Require(core.ActionTaxonomyManage, core.ScopePlatform))
}

// --- Public discovery ----------------------------------------------------

func (h *Handler) taxonomy(c echo.Context) error {
	parentID := c.QueryParam("parent_id")
	if parentID != "" && !uuid.IsValid(parentID) {
		// A malformed id is a client mistake, not a server fault. Rejecting it
		// here keeps an invalid uuid literal out of the query, where the driver
		// would turn it into a 500 and a noisy log line.
		return core.Invalid("parent_id", "expected a UUID")
	}
	list, err := h.svc.TaxonomyChildren(c.Request().Context(), parentID)
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusOK, list)
}

func (h *Handler) taxonomyBySlug(c echo.Context) error {
	node, err := h.svc.TaxonomyNodeBySlug(c.Request().Context(), c.Param("slug"))
	if err != nil {
		return err
	}
	if !node.Published || node.Status != core.TaxonomyStatusActive {
		return core.ErrNotFound
	}
	return httpx.RespondOK(c, http.StatusOK, node)
}

func (h *Handler) publicOfferings(c echo.Context) error {
	limit, offset := pageParams(c)
	f := core.OfferingFilter{
		TaxonomyID: c.QueryParam("taxonomy_id"),
		RegionID:   c.QueryParam("region_id"),
		Query:      c.QueryParam("q"),
		Status:     core.OfferingStatusPublished,
		Visibility: core.VisibilityPublic,
		Limit:      limit,
		Offset:     offset,
	}
	if v := c.QueryParam("organization_id"); v != "" {
		f.OrganizationID = v
	}
	list, total, err := h.svc.Offerings(c.Request().Context(), f)
	if err != nil {
		return err
	}
	return httpx.RespondList(c, list, total, limit, offset)
}

func (h *Handler) publicOffering(c echo.Context) error {
	o, err := h.svc.PublicOffering(c.Request().Context(), c.Param("id"))
	if err != nil {
		return err
	}
	if o.Status != core.OfferingStatusPublished || o.Visibility == core.VisibilityPrivate {
		return core.ErrNotFound
	}
	return httpx.RespondOK(c, http.StatusOK, o)
}

// --- Tenant catalogue ----------------------------------------------------

func (h *Handler) organizationOfferings(c echo.Context) error {
	limit, offset := pageParams(c)
	list, total, err := h.svc.Offerings(c.Request().Context(), core.OfferingFilter{
		OrganizationID: c.Param("org_id"),
		Status:         c.QueryParam("status"),
		TaxonomyID:     c.QueryParam("taxonomy_id"),
		Query:          c.QueryParam("q"),
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		return err
	}
	return httpx.RespondList(c, list, total, limit, offset)
}

func (h *Handler) organizationCategories(c echo.Context) error {
	list, err := h.svc.OrganizationCategories(c.Request().Context(), c.Param("org_id"))
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusOK, list)
}

func (h *Handler) setOrganizationCategories(c echo.Context) error {
	p := core.UserFrom(c)
	var in struct {
		TaxonomyIDs []string `json:"taxonomy_ids"`
	}
	if err := c.Bind(&in); err != nil {
		return core.ErrValidation
	}
	if err := h.svc.SetOrganizationCategories(c.Request().Context(),
		c.Param("org_id"), in.TaxonomyIDs, p.UserID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) createOffering(c echo.Context) error {
	var in offeringBody
	if err := c.Bind(&in); err != nil {
		return core.ErrValidation
	}
	o, err := h.svc.CreateOffering(c.Request().Context(), in.toInput(c.Param("org_id")))
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusCreated, o)
}

func (h *Handler) updateOffering(c echo.Context) error {
	var in offeringBody
	if err := c.Bind(&in); err != nil {
		return core.ErrValidation
	}
	input := in.toInput(c.Param("org_id"))
	input.ID = c.Param("offering_id")
	o, err := h.svc.UpdateOffering(c.Request().Context(), input)
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusOK, o)
}

func (h *Handler) deleteOffering(c echo.Context) error {
	p := core.UserFrom(c)
	// The offering id is caller-supplied, so the tenant in the path proves
	// nothing on its own. assertOfferingInOrg resolves the offering and
	// compares it to :org_id; without it a member of organization A could
	// delete organization B's draft by naming their own tenant in the path.
	if err := h.assertOfferingInOrg(c, c.Param("offering_id")); err != nil {
		return err
	}
	if err := h.svc.DeleteOffering(c.Request().Context(), c.Param("offering_id"), p.UserID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) setOfferingStatus(c echo.Context) error {
	var in struct {
		Status string `json:"status"`
	}
	if err := c.Bind(&in); err != nil {
		return core.ErrValidation
	}
	// Resolve the offering first so a route naming another tenant's offering
	// cannot change its status.
	o, err := h.svc.Offering(c.Request().Context(), c.Param("offering_id"))
	if err != nil {
		return err
	}
	if o.OrganizationID != c.Param("org_id") {
		return core.ErrNotFound
	}
	updated, err := h.svc.SetOfferingStatus(c.Request().Context(), o.ID, in.Status, "")
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusOK, updated)
}

func (h *Handler) createVariant(c echo.Context) error {
	var in variantBody
	if err := c.Bind(&in); err != nil {
		return core.ErrValidation
	}
	input := in.toInput()
	input.OfferingID = c.Param("offering_id")
	if err := h.assertOfferingInOrg(c, input.OfferingID); err != nil {
		return err
	}
	v, err := h.svc.CreateVariant(c.Request().Context(), input)
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusCreated, v)
}

func (h *Handler) updateVariant(c echo.Context) error {
	var in variantBody
	if err := c.Bind(&in); err != nil {
		return core.ErrValidation
	}
	input := in.toInput()
	input.ID = c.Param("variant_id")
	input.OfferingID = c.Param("offering_id")
	if err := h.assertOfferingInOrg(c, input.OfferingID); err != nil {
		return err
	}
	v, err := h.svc.UpdateVariant(c.Request().Context(), input)
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusOK, v)
}

func (h *Handler) deleteVariant(c echo.Context) error {
	p := core.UserFrom(c)
	if err := h.assertOfferingInOrg(c, c.Param("offering_id")); err != nil {
		return err
	}
	// The offering check alone is not enough: the variant id is independent
	// of it. Passing the offering through to the service lets it reject a
	// variant that belongs to a different offering (and therefore possibly a
	// different tenant).
	if err := h.svc.DeleteVariant(c.Request().Context(),
		c.Param("offering_id"), c.Param("variant_id"), p.UserID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// assertOfferingInOrg proves the offering belongs to the tenant in the route.
func (h *Handler) assertOfferingInOrg(c echo.Context, offeringID string) error {
	o, err := h.svc.Offering(c.Request().Context(), offeringID)
	if err != nil {
		return err
	}
	if o.OrganizationID != c.Param("org_id") {
		return core.ErrNotFound
	}
	return nil
}

// --- Platform taxonomy ---------------------------------------------------

func (h *Handler) createTaxonomy(c echo.Context) error {
	var in taxonomyBody
	if err := c.Bind(&in); err != nil {
		return core.ErrValidation
	}
	n, err := h.svc.CreateTaxonomyNode(c.Request().Context(), in.toInput(""))
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusCreated, n)
}

func (h *Handler) updateTaxonomy(c echo.Context) error {
	var in taxonomyBody
	if err := c.Bind(&in); err != nil {
		return core.ErrValidation
	}
	n, err := h.svc.UpdateTaxonomyNode(c.Request().Context(), in.toInput(c.Param("id")))
	if err != nil {
		return err
	}
	return httpx.RespondOK(c, http.StatusOK, n)
}

// --- Bodies --------------------------------------------------------------

type offeringBody struct {
	TaxonomyID    string             `json:"taxonomy_id"`
	BranchID      string             `json:"branch_id"`
	Name          string             `json:"name"`
	Slug          string             `json:"slug"`
	Summary       string             `json:"summary"`
	Description   string             `json:"description"`
	MediaRefs     []string           `json:"media_refs"`
	ServiceMode   string             `json:"service_mode"`
	Visibility    string             `json:"visibility"`
	Locale        string             `json:"locale"`
	Tags          []string           `json:"tags"`
	PriceFrom     float64            `json:"price_from"`
	PriceCurrency string             `json:"price_currency"`
	ExtensionRef  *core.ExtensionRef `json:"extension_ref"`
}

func (b offeringBody) toInput(organizationID string) core.OfferingInput {
	return core.OfferingInput{
		OrganizationID: organizationID,
		BranchID:       b.BranchID,
		TaxonomyID:     b.TaxonomyID,
		Name:           b.Name,
		Slug:           b.Slug,
		Summary:        b.Summary,
		Description:    b.Description,
		MediaRefs:      b.MediaRefs,
		ServiceMode:    b.ServiceMode,
		Visibility:     b.Visibility,
		Locale:         b.Locale,
		Tags:           b.Tags,
		PriceFrom:      b.PriceFrom,
		PriceCurrency:  b.PriceCurrency,
		ExtensionRef:   b.ExtensionRef,
	}
}

type variantBody struct {
	Name         string             `json:"name"`
	Status       string             `json:"status"`
	Visibility   string             `json:"visibility"`
	MediaRefs    []string           `json:"media_refs"`
	ExtensionRef *core.ExtensionRef `json:"extension_ref"`
}

func (b variantBody) toInput() core.VariantInput {
	return core.VariantInput{
		Name:         b.Name,
		Status:       b.Status,
		Visibility:   b.Visibility,
		MediaRefs:    b.MediaRefs,
		ExtensionRef: b.ExtensionRef,
	}
}

type taxonomyBody struct {
	ParentID           string `json:"parent_id"`
	Slug               string `json:"slug"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	Status             string `json:"status"`
	Published          bool   `json:"published"`
	Sort               int    `json:"sort"`
	Locale             string `json:"locale"`
	ExtensionSchemaRef string `json:"extension_schema_ref"`
}

func (b taxonomyBody) toInput(id string) core.TaxonomyInput {
	return core.TaxonomyInput{
		ID:                 id,
		ParentID:           b.ParentID,
		Slug:               b.Slug,
		Name:               b.Name,
		Description:        b.Description,
		Status:             b.Status,
		Published:          b.Published,
		Sort:               b.Sort,
		Locale:             b.Locale,
		ExtensionSchemaRef: b.ExtensionSchemaRef,
	}
}

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
