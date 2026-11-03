package core

import (
	"context"
	"time"
)

// Offering statuses describe publishing, not a business transaction.
const (
	OfferingStatusDraft     = "draft"
	OfferingStatusPublished = "published"
	OfferingStatusArchived  = "archived"
)

// Offering visibility widens or narrows discovery.
const (
	VisibilityPublic   = "public"
	VisibilityUnlisted = "unlisted"
	VisibilityPrivate  = "private"
)

// Taxonomy statuses.
const (
	TaxonomyStatusActive   = "active"
	TaxonomyStatusInactive = "inactive"
)

// TaxonomyNode is one node of the central category tree.
//
// It is the only authoritative source of category names. Free text must never
// stand in for a category reference, which is what keeps discovery, filtering
// and reporting consistent across sectors.
type TaxonomyNode struct {
	ID          string `json:"id"`
	ParentID    string `json:"parent_id,omitempty"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	// Published controls whether the node may be used in public discovery.
	Published bool   `json:"published"`
	Sort      int    `json:"sort"`
	Locale    string `json:"locale"`
	// ExtensionSchemaRef points at an extension-owned field schema. The core
	// stores the reference and never interprets it.
	ExtensionSchemaRef string    `json:"extension_schema_ref,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// Offering is the sector-independent catalogue record.
//
// It answers "what does this business offer", not "how is it sold". Duration,
// capacity, SKU, stock, package contents and price calculation are engine
// concerns and are deliberately absent.
type Offering struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	BranchID       string `json:"branch_id,omitempty"`
	TaxonomyID     string `json:"taxonomy_id"`
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	Summary        string `json:"summary"`
	Description    string `json:"description"`
	// MediaRefs are file ids owned by the files module. Access is authorized
	// separately when the file is fetched.
	MediaRefs []string `json:"media_refs"`
	// ServiceMode is one of the generic modes (on_site, online, at_customer,
	// hybrid). It describes presentation, not availability.
	ServiceMode string   `json:"service_mode"`
	Status      string   `json:"status"`
	Visibility  string   `json:"visibility"`
	Locale      string   `json:"locale"`
	Tags        []string `json:"tags"`
	// PriceFrom and PriceCurrency are informational only. No calculation,
	// tax, discount or collection logic may be derived from them.
	PriceFrom     float64 `json:"price_from,omitempty"`
	PriceCurrency string  `json:"price_currency,omitempty"`
	// ExtensionRef lets an engine attach its own record without the core
	// knowing anything about it.
	ExtensionRef *ExtensionRef `json:"extension_ref,omitempty"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

// OfferingVariant is a named presentation variant of an offering.
//
// Only shared fields live here. A variant's duration, capacity, SKU or content
// belongs to the engine that owns the variant's meaning.
type OfferingVariant struct {
	ID           string        `json:"id"`
	OfferingID   string        `json:"offering_id"`
	Name         string        `json:"name"`
	Status       string        `json:"status"`
	Visibility   string        `json:"visibility"`
	MediaRefs    []string      `json:"media_refs"`
	ExtensionRef *ExtensionRef `json:"extension_ref,omitempty"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

// ExtensionRef identifies a record owned by an extension or engine.
//
// The core stores the reference and a label; it never dereferences the record
// itself. This is the boundary that keeps engine data out of the core database.
type ExtensionRef struct {
	ExtensionID string `json:"extension_id"`
	// ResourceType and ResourceID are the extension's own identifiers.
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	// Label is an optional, non-authoritative display hint.
	Label string `json:"label,omitempty"`
}

// OfferingInput creates or replaces an offering.
type OfferingInput struct {
	ID             string
	OrganizationID string
	BranchID       string
	TaxonomyID     string
	Name           string
	Slug           string
	Summary        string
	Description    string
	MediaRefs      []string
	ServiceMode    string
	Status         string
	Visibility     string
	Locale         string
	Tags           []string
	PriceFrom      float64
	PriceCurrency  string
	ExtensionRef   *ExtensionRef
}

// OfferingFilter narrows a listing.
type OfferingFilter struct {
	OrganizationID string
	BranchID       string
	TaxonomyID     string
	Status         string
	Visibility     string
	RegionID       string
	Query          string
	Limit          int
	Offset         int
}

// VariantInput creates or replaces an offering variant.
type VariantInput struct {
	ID           string
	OfferingID   string
	Name         string
	Status       string
	Visibility   string
	MediaRefs    []string
	ExtensionRef *ExtensionRef
}

// TaxonomyInput creates or replaces a taxonomy node.
type TaxonomyInput struct {
	ID                 string
	ParentID           string
	Slug               string
	Name               string
	Description        string
	Status             string
	Published          bool
	Sort               int
	Locale             string
	ExtensionSchemaRef string
}

// Catalog is the catalogue service surface other modules use.
type Catalog interface {
	// Taxonomy
	CreateTaxonomyNode(ctx context.Context, in TaxonomyInput) (*TaxonomyNode, error)
	UpdateTaxonomyNode(ctx context.Context, in TaxonomyInput) (*TaxonomyNode, error)
	TaxonomyNode(ctx context.Context, id string) (*TaxonomyNode, error)
	TaxonomyNodeBySlug(ctx context.Context, slug string) (*TaxonomyNode, error)
	TaxonomyChildren(ctx context.Context, parentID string) ([]TaxonomyNode, error)
	TaxonomyPath(ctx context.Context, id string) ([]TaxonomyNode, error)

	// Organization categories
	SetOrganizationCategories(ctx context.Context, organizationID string, taxonomyIDs []string, actorID string) error
	OrganizationCategories(ctx context.Context, organizationID string) ([]TaxonomyNode, error)

	// Offerings
	CreateOffering(ctx context.Context, in OfferingInput) (*Offering, error)
	UpdateOffering(ctx context.Context, in OfferingInput) (*Offering, error)
	Offering(ctx context.Context, id string) (*Offering, error)
	Offerings(ctx context.Context, f OfferingFilter) ([]Offering, int64, error)
	SetOfferingStatus(ctx context.Context, id, status, actorID string) (*Offering, error)
	DeleteOffering(ctx context.Context, id, actorID string) error

	// Variants
	CreateVariant(ctx context.Context, in VariantInput) (*OfferingVariant, error)
	UpdateVariant(ctx context.Context, in VariantInput) (*OfferingVariant, error)
	Variants(ctx context.Context, offeringID string) ([]OfferingVariant, error)
	DeleteVariant(ctx context.Context, id, actorID string) error
}
