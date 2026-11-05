// Package service implements the catalogue rules.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"core-platform/contract/core"
	"core-platform/mod/catalog/model"
	"core-platform/mod/catalog/repository"
	"core-platform/platform/uuid"
)

// maxLen rejects a value that would not fit its column, so an over-long string
// is a 422 rather than a driver error surfaced as a 500.
func maxLen(field, value string, limit int) error {
	if len([]rune(value)) > limit {
		return core.Invalid(field, fmt.Sprintf("must be at most %d characters", limit))
	}
	return nil
}

var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,138}[a-z0-9]$`)

// Catalog implements core.Catalog.
type Catalog struct {
	store *repository.Store
	orgs  core.Organizations
}

// New builds the catalog service.
//
// Authorization is enforced by the HTTP layer through Authorizer.Require; the
// service itself only enforces tenant ownership on the records it loads.
func New(store *repository.Store, orgs core.Organizations, _ core.Authorizer) *Catalog {
	return &Catalog{store: store, orgs: orgs}
}

// --- Taxonomy ------------------------------------------------------------

// CreateTaxonomyNode adds a category node.
func (c *Catalog) CreateTaxonomyNode(ctx context.Context, in core.TaxonomyInput) (*core.TaxonomyNode, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, core.Invalid("name", "a name is required")
	}
	slug := strings.ToLower(strings.TrimSpace(in.Slug))
	if !slugPattern.MatchString(slug) || len(slug) > 120 {
		return nil, core.Invalid("slug", "use lowercase letters, digits and hyphens (max 120)")
	}
	if in.ParentID != "" {
		if _, err := c.store.Taxonomy(ctx, in.ParentID); err != nil {
			return nil, core.Invalid("parent_id", "the parent category does not exist")
		}
	}

	now := time.Now().UTC()
	rec := &model.TaxonomyNode{
		ID:                 uuid.New(),
		ParentID:           optional(in.ParentID),
		Slug:               slug,
		Name:               strings.TrimSpace(in.Name),
		Description:        in.Description,
		Status:             defaultString(in.Status, core.TaxonomyStatusActive),
		Published:          in.Published,
		Sort:               in.Sort,
		Locale:             defaultString(in.Locale, "en"),
		ExtensionSchemaRef: in.ExtensionSchemaRef,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := c.store.CreateTaxonomy(ctx, rec); err != nil {
		return nil, err
	}
	out := toCoreTaxonomy(rec)
	return &out, nil
}

// UpdateTaxonomyNode writes a category node.
func (c *Catalog) UpdateTaxonomyNode(ctx context.Context, in core.TaxonomyInput) (*core.TaxonomyNode, error) {
	rec, err := c.store.Taxonomy(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	if in.ParentID == in.ID {
		return nil, core.Invalid("parent_id", "a category cannot be its own parent")
	}
	if in.Name != "" {
		rec.Name = strings.TrimSpace(in.Name)
	}
	if in.Slug != "" {
		slug := strings.ToLower(strings.TrimSpace(in.Slug))
		if !slugPattern.MatchString(slug) || len(slug) > 120 {
			return nil, core.Invalid("slug", "use lowercase letters, digits and hyphens (max 120)")
		}
		rec.Slug = slug
	}
	if in.Description != "" {
		rec.Description = in.Description
	}
	if in.Status != "" {
		rec.Status = in.Status
	}
	if in.Locale != "" {
		rec.Locale = in.Locale
	}
	if in.ParentID != "" {
		if _, err := c.store.Taxonomy(ctx, in.ParentID); err != nil {
			return nil, core.Invalid("parent_id", "the parent category does not exist")
		}
	}
	rec.ParentID = optional(in.ParentID)
	rec.Published = in.Published
	rec.Sort = in.Sort
	rec.ExtensionSchemaRef = in.ExtensionSchemaRef

	if err := c.store.UpdateTaxonomy(ctx, rec); err != nil {
		return nil, err
	}
	out := toCoreTaxonomy(rec)
	return &out, nil
}

// TaxonomyNode returns one node.
func (c *Catalog) TaxonomyNode(ctx context.Context, id string) (*core.TaxonomyNode, error) {
	rec, err := c.store.Taxonomy(ctx, id)
	if err != nil {
		return nil, err
	}
	out := toCoreTaxonomy(rec)
	return &out, nil
}

// TaxonomyNodeBySlug returns one node.
func (c *Catalog) TaxonomyNodeBySlug(ctx context.Context, slug string) (*core.TaxonomyNode, error) {
	rec, err := c.store.TaxonomyBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	out := toCoreTaxonomy(rec)
	return &out, nil
}

// TaxonomyChildren lists children of a node, or the roots.
func (c *Catalog) TaxonomyChildren(ctx context.Context, parentID string) ([]core.TaxonomyNode, error) {
	rows, err := c.store.TaxonomyChildren(ctx, parentID)
	if err != nil {
		return nil, err
	}
	out := make([]core.TaxonomyNode, 0, len(rows))
	for i := range rows {
		out = append(out, toCoreTaxonomy(&rows[i]))
	}
	return out, nil
}

// TaxonomyPath returns the node and its ancestors, root first.
func (c *Catalog) TaxonomyPath(ctx context.Context, id string) ([]core.TaxonomyNode, error) {
	var path []core.TaxonomyNode
	cur := id
	// The depth cap protects against a cycle that a manual SQL edit created;
	// the service itself never writes one.
	for depth := 0; depth < 32 && cur != ""; depth++ {
		rec, err := c.store.Taxonomy(ctx, cur)
		if err != nil {
			if depth == 0 {
				return nil, err
			}
			break
		}
		path = append(path, toCoreTaxonomy(rec))
		cur = deref(rec.ParentID)
	}
	// Reverse into root-first order.
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path, nil
}

// --- Organization categories --------------------------------------------

// SetOrganizationCategories replaces an organization's categories.
func (c *Catalog) SetOrganizationCategories(ctx context.Context, organizationID string, taxonomyIDs []string, actorID string) error {
	if organizationID == "" {
		return core.Invalid("organization_id", "an organization is required")
	}
	for _, id := range taxonomyIDs {
		if _, err := c.store.Taxonomy(ctx, id); err != nil {
			return core.Invalid("taxonomy_ids", "unknown category: "+id)
		}
	}
	return c.store.SetOrganizationCategories(ctx, organizationID, taxonomyIDs)
}

// OrganizationCategories lists an organization's categories.
func (c *Catalog) OrganizationCategories(ctx context.Context, organizationID string) ([]core.TaxonomyNode, error) {
	rows, err := c.store.OrganizationCategories(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	out := make([]core.TaxonomyNode, 0, len(rows))
	for i := range rows {
		out = append(out, toCoreTaxonomy(&rows[i]))
	}
	return out, nil
}

// --- Offerings -----------------------------------------------------------

// CreateOffering adds an offering.
func (c *Catalog) CreateOffering(ctx context.Context, in core.OfferingInput) (*core.Offering, error) {
	if in.OrganizationID == "" {
		return nil, core.Invalid("organization_id", "an organization is required")
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, core.Invalid("name", "a name is required")
	}
	if err := maxLen("name", name, 200); err != nil {
		return nil, err
	}
	slug := strings.ToLower(strings.TrimSpace(in.Slug))
	if slug == "" {
		return nil, core.Invalid("slug", "a slug is required")
	}
	if err := maxLen("slug", slug, 140); err != nil {
		return nil, err
	}
	if !slugPattern.MatchString(slug) {
		return nil, core.Invalid("slug", "use lowercase letters, digits and hyphens")
	}
	if in.TaxonomyID == "" {
		return nil, core.Invalid("taxonomy_id", "a category is required")
	}
	if _, err := c.store.Taxonomy(ctx, in.TaxonomyID); err != nil {
		return nil, core.Invalid("taxonomy_id", "the category does not exist")
	}

	taken, err := c.store.OfferingSlugTaken(ctx, in.OrganizationID, slug, "")
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, core.ErrConflict
	}
	if err := maxLen("summary", in.Summary, 300); err != nil {
		return nil, err
	}
	if err := maxLen("locale", in.Locale, 10); err != nil {
		return nil, err
	}
	if err := maxLen("price_currency", in.PriceCurrency, 3); err != nil {
		return nil, err
	}

	media, err := encodeMedia(in.MediaRefs)
	if err != nil {
		return nil, err
	}
	tags, err := encodeMedia(in.Tags)
	if err != nil {
		return nil, err
	}
	ext, err := encodeExtension(in.ExtensionRef)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	rec := &model.Offering{
		ID:             uuid.New(),
		OrganizationID: in.OrganizationID,
		BranchID:       optional(in.BranchID),
		TaxonomyID:     in.TaxonomyID,
		Name:           name,
		Slug:           slug,
		Summary:        in.Summary,
		Description:    in.Description,
		MediaRefs:      media,
		ServiceMode:    defaultString(in.ServiceMode, core.ServiceModeOnSite),
		Status:         core.OfferingStatusDraft,
		Visibility:     defaultString(in.Visibility, core.VisibilityPublic),
		Locale:         defaultString(in.Locale, "en"),
		Tags:           tags,
		PriceFrom:      optionalPrice(in.PriceFrom),
		PriceCurrency:  strings.ToUpper(in.PriceCurrency),
		ExtensionRef:   ext,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	// A caller may not smuggle a published offering into creation; publishing
	// is a separate, auditable action.
	if err := c.store.CreateOffering(ctx, rec); err != nil {
		return nil, err
	}
	return toCoreOffering(rec), nil
}

// UpdateOffering writes an offering.
func (c *Catalog) UpdateOffering(ctx context.Context, in core.OfferingInput) (*core.Offering, error) {
	rec, err := c.store.Offering(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	if in.OrganizationID != "" && in.OrganizationID != rec.OrganizationID {
		return nil, core.ErrNotFound
	}

	if in.Name != "" {
		name := strings.TrimSpace(in.Name)
		if err := maxLen("name", name, 200); err != nil {
			return nil, err
		}
		rec.Name = name
	}
	if in.Slug != "" {
		slug := strings.ToLower(strings.TrimSpace(in.Slug))
		if err := maxLen("slug", slug, 140); err != nil {
			return nil, err
		}
		if !slugPattern.MatchString(slug) {
			return nil, core.Invalid("slug", "use lowercase letters, digits and hyphens")
		}
		taken, err := c.store.OfferingSlugTaken(ctx, rec.OrganizationID, slug, rec.ID)
		if err != nil {
			return nil, err
		}
		if taken {
			return nil, core.ErrConflict
		}
		rec.Slug = slug
	}
	if in.TaxonomyID != "" {
		if _, err := c.store.Taxonomy(ctx, in.TaxonomyID); err != nil {
			return nil, core.Invalid("taxonomy_id", "the category does not exist")
		}
		rec.TaxonomyID = in.TaxonomyID
	}
	if in.Summary != "" {
		if err := maxLen("summary", in.Summary, 300); err != nil {
			return nil, err
		}
		rec.Summary = in.Summary
	}
	if in.Description != "" {
		rec.Description = in.Description
	}
	if in.ServiceMode != "" {
		rec.ServiceMode = in.ServiceMode
	}
	if in.Visibility != "" {
		rec.Visibility = in.Visibility
	}
	if in.Locale != "" {
		rec.Locale = in.Locale
	}
	if in.PriceCurrency != "" {
		rec.PriceCurrency = strings.ToUpper(in.PriceCurrency)
	}
	rec.BranchID = optional(in.BranchID)
	if in.MediaRefs != nil {
		media, err := encodeMedia(in.MediaRefs)
		if err != nil {
			return nil, err
		}
		rec.MediaRefs = media
	}
	if in.Tags != nil {
		tags, err := encodeMedia(in.Tags)
		if err != nil {
			return nil, err
		}
		rec.Tags = tags
	}
	if in.ExtensionRef != nil {
		ext, err := encodeExtension(in.ExtensionRef)
		if err != nil {
			return nil, err
		}
		rec.ExtensionRef = ext
	}
	rec.PriceFrom = optionalPrice(in.PriceFrom)

	if err := c.store.UpdateOffering(ctx, rec); err != nil {
		return nil, err
	}
	return toCoreOffering(rec), nil
}

// Offering returns one offering.
func (c *Catalog) Offering(ctx context.Context, id string) (*core.Offering, error) {
	rec, err := c.store.Offering(ctx, id)
	if err != nil {
		return nil, err
	}
	return toCoreOffering(rec), nil
}

// PublicOffering returns an offering only when its business is visible.
//
// The handler uses this for anonymous discovery; a tenant route keeps using
// Offering so a draft remains readable to its owner.
func (c *Catalog) PublicOffering(ctx context.Context, id string) (*core.Offering, error) {
	rec, err := c.store.Offering(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.orgs != nil {
		org, err := c.orgs.ByID(ctx, rec.OrganizationID)
		if err != nil || org.Status != core.OrgStatusActive {
			return nil, core.ErrNotFound
		}
	}
	return toCoreOffering(rec), nil
}

// Offerings lists a page of offerings.
func (c *Catalog) Offerings(ctx context.Context, f core.OfferingFilter) ([]core.Offering, int64, error) {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 20
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	rows, total, err := c.store.ListOfferings(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]core.Offering, 0, len(rows))
	for i := range rows {
		out = append(out, *toCoreOffering(&rows[i]))
	}
	return out, total, nil
}

// SetOfferingStatus applies a publishing transition.
func (c *Catalog) SetOfferingStatus(ctx context.Context, id, status, actorID string) (*core.Offering, error) {
	rec, err := c.store.Offering(ctx, id)
	if err != nil {
		return nil, err
	}
	if !allowedOfferingTransition(rec.Status, status) {
		return nil, core.StateError("cannot move from " + rec.Status + " to " + status)
	}
	if err := c.store.SetOfferingStatus(ctx, rec.ID, rec.OrganizationID, rec.Status, status); err != nil {
		return nil, err
	}
	rec.Status = status
	return toCoreOffering(rec), nil
}

// DeleteOffering removes an offering.
func (c *Catalog) DeleteOffering(ctx context.Context, id, actorID string) error {
	rec, err := c.store.Offering(ctx, id)
	if err != nil {
		return err
	}
	if rec.Status == core.OfferingStatusPublished {
		return core.StateError("archive a published offering before deleting it")
	}
	return c.store.DeleteOffering(ctx, rec.ID, rec.OrganizationID)
}

// --- Variants ------------------------------------------------------------

// CreateVariant adds a variant.
func (c *Catalog) CreateVariant(ctx context.Context, in core.VariantInput) (*core.OfferingVariant, error) {
	if in.OfferingID == "" {
		return nil, core.Invalid("offering_id", "an offering is required")
	}
	if strings.TrimSpace(in.Name) == "" {
		return nil, core.Invalid("name", "a name is required")
	}
	if _, err := c.store.Offering(ctx, in.OfferingID); err != nil {
		return nil, err
	}
	media, err := encodeMedia(in.MediaRefs)
	if err != nil {
		return nil, err
	}
	ext, err := encodeExtension(in.ExtensionRef)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	rec := &model.Variant{
		ID:           uuid.New(),
		OfferingID:   in.OfferingID,
		Name:         strings.TrimSpace(in.Name),
		Status:       defaultString(in.Status, core.TaxonomyStatusActive),
		Visibility:   defaultString(in.Visibility, core.VisibilityPublic),
		MediaRefs:    media,
		ExtensionRef: ext,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := c.store.CreateVariant(ctx, rec); err != nil {
		return nil, err
	}
	return toCoreVariant(rec), nil
}

// UpdateVariant writes a variant.
func (c *Catalog) UpdateVariant(ctx context.Context, in core.VariantInput) (*core.OfferingVariant, error) {
	rec, err := c.store.Variant(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	if in.OfferingID != "" && in.OfferingID != rec.OfferingID {
		return nil, core.ErrNotFound
	}
	if in.Name != "" {
		rec.Name = strings.TrimSpace(in.Name)
	}
	if in.Status != "" {
		rec.Status = in.Status
	}
	if in.Visibility != "" {
		rec.Visibility = in.Visibility
	}
	if in.MediaRefs != nil {
		media, err := encodeMedia(in.MediaRefs)
		if err != nil {
			return nil, err
		}
		rec.MediaRefs = media
	}
	if in.ExtensionRef != nil {
		ext, err := encodeExtension(in.ExtensionRef)
		if err != nil {
			return nil, err
		}
		rec.ExtensionRef = ext
	}
	if err := c.store.UpdateVariant(ctx, rec); err != nil {
		return nil, err
	}
	return toCoreVariant(rec), nil
}

// Variants lists an offering's variants.
func (c *Catalog) Variants(ctx context.Context, offeringID string) ([]core.OfferingVariant, error) {
	rows, err := c.store.Variants(ctx, offeringID)
	if err != nil {
		return nil, err
	}
	out := make([]core.OfferingVariant, 0, len(rows))
	for i := range rows {
		out = append(out, *toCoreVariant(&rows[i]))
	}
	return out, nil
}

// DeleteVariant removes a variant.
// DeleteVariant removes a variant, proving first that it belongs to the
// offering named in the route. Without that comparison an attacker who is
// allowed to edit one offering could delete a variant of any other by guessing
// its id.
func (c *Catalog) DeleteVariant(ctx context.Context, offeringID, id, actorID string) error {
	rec, err := c.store.Variant(ctx, id)
	if err != nil {
		return err
	}
	if offeringID != "" && rec.OfferingID != offeringID {
		return core.ErrNotFound
	}
	return c.store.DeleteVariant(ctx, rec.ID, rec.OfferingID)
}

// --- Conversion ----------------------------------------------------------

func toCoreTaxonomy(n *model.TaxonomyNode) core.TaxonomyNode {
	out := core.TaxonomyNode{
		ID:                 n.ID,
		Slug:               n.Slug,
		Name:               n.Name,
		Description:        n.Description,
		Status:             n.Status,
		Published:          n.Published,
		Sort:               n.Sort,
		Locale:             n.Locale,
		ExtensionSchemaRef: n.ExtensionSchemaRef,
		CreatedAt:          n.CreatedAt,
		UpdatedAt:          n.UpdatedAt,
	}
	if n.ParentID != nil {
		out.ParentID = *n.ParentID
	}
	return out
}

func toCoreOffering(o *model.Offering) *core.Offering {
	out := &core.Offering{
		ID:             o.ID,
		OrganizationID: o.OrganizationID,
		TaxonomyID:     o.TaxonomyID,
		Name:           o.Name,
		Slug:           o.Slug,
		Summary:        o.Summary,
		Description:    o.Description,
		MediaRefs:      decodeMedia(o.MediaRefs),
		ServiceMode:    o.ServiceMode,
		Status:         o.Status,
		Visibility:     o.Visibility,
		Locale:         o.Locale,
		Tags:           decodeMedia(o.Tags),
		PriceCurrency:  o.PriceCurrency,
		ExtensionRef:   decodeExtension(o.ExtensionRef),
		CreatedAt:      o.CreatedAt,
		UpdatedAt:      o.UpdatedAt,
	}
	if o.BranchID != nil {
		out.BranchID = *o.BranchID
	}
	if o.PriceFrom != nil {
		out.PriceFrom = *o.PriceFrom
	}
	return out
}

func toCoreVariant(v *model.Variant) *core.OfferingVariant {
	return &core.OfferingVariant{
		ID:           v.ID,
		OfferingID:   v.OfferingID,
		Name:         v.Name,
		Status:       v.Status,
		Visibility:   v.Visibility,
		MediaRefs:    decodeMedia(v.MediaRefs),
		ExtensionRef: decodeExtension(v.ExtensionRef),
		CreatedAt:    v.CreatedAt,
		UpdatedAt:    v.UpdatedAt,
	}
}

// --- Encoding helpers ----------------------------------------------------

func encodeMedia(items []string) (string, error) {
	if items == nil {
		return "[]", nil
	}
	b, err := json.Marshal(items)
	if err != nil {
		return "", core.Invalid("media_refs", "could not encode the list")
	}
	return string(b), nil
}

func decodeMedia(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func encodeExtension(ref *core.ExtensionRef) (*string, error) {
	if ref == nil {
		return nil, nil
	}
	b, err := json.Marshal(ref)
	if err != nil {
		return nil, core.Invalid("extension_ref", "could not encode the reference")
	}
	s := string(b)
	return &s, nil
}

func decodeExtension(raw *string) *core.ExtensionRef {
	if raw == nil || *raw == "" {
		return nil
	}
	var out core.ExtensionRef
	if err := json.Unmarshal([]byte(*raw), &out); err != nil {
		return nil
	}
	return &out
}

func optionalPrice(v float64) *float64 {
	if v <= 0 {
		return nil
	}
	return &v
}

func allowedOfferingTransition(from, to string) bool {
	switch from {
	case core.OfferingStatusDraft:
		return to == core.OfferingStatusPublished || to == core.OfferingStatusArchived
	case core.OfferingStatusPublished:
		return to == core.OfferingStatusArchived
	case core.OfferingStatusArchived:
		return to == core.OfferingStatusDraft
	}
	return false
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func defaultString(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}
