// Package model holds the catalog persistence types.
//
// JSON columns are stored as text and encoded by the service. That keeps the
// persistence layer free of the wire shape and means a change to the JSON
// representation never requires a migration.
package model

import "time"

// TaxonomyNode is one node of the central category tree.
type TaxonomyNode struct {
	ID                 string    `gorm:"column:id;primaryKey"`
	ParentID           *string   `gorm:"column:parent_id"`
	Slug               string    `gorm:"column:slug"`
	Name               string    `gorm:"column:name"`
	Description        string    `gorm:"column:description"`
	Status             string    `gorm:"column:status"`
	Published          bool      `gorm:"column:published"`
	Sort               int       `gorm:"column:sort"`
	Locale             string    `gorm:"column:locale"`
	ExtensionSchemaRef string    `gorm:"column:extension_schema_ref"`
	CreatedAt          time.Time `gorm:"column:created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at"`
}

// TableName pins the table name.
func (TaxonomyNode) TableName() string { return "taxonomy_nodes" }

// Offering is a sector-independent catalogue record.
type Offering struct {
	ID             string    `gorm:"column:id;primaryKey"`
	OrganizationID string    `gorm:"column:organization_id"`
	BranchID       *string   `gorm:"column:branch_id"`
	TaxonomyID     string    `gorm:"column:taxonomy_id"`
	Name           string    `gorm:"column:name"`
	Slug           string    `gorm:"column:slug"`
	Summary        string    `gorm:"column:summary"`
	Description    string    `gorm:"column:description"`
	MediaRefs      string    `gorm:"column:media_refs"`
	ServiceMode    string    `gorm:"column:service_mode"`
	Status         string    `gorm:"column:status"`
	Visibility     string    `gorm:"column:visibility"`
	Locale         string    `gorm:"column:locale"`
	Tags           string    `gorm:"column:tags"`
	PriceFrom      *float64  `gorm:"column:price_from"`
	PriceCurrency  string    `gorm:"column:price_currency"`
	ExtensionRef   *string   `gorm:"column:extension_ref"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

// TableName pins the table name.
func (Offering) TableName() string { return "offerings" }

// Variant is a named presentation variant of an offering.
type Variant struct {
	ID           string    `gorm:"column:id;primaryKey"`
	OfferingID   string    `gorm:"column:offering_id"`
	Name         string    `gorm:"column:name"`
	Status       string    `gorm:"column:status"`
	Visibility   string    `gorm:"column:visibility"`
	MediaRefs    string    `gorm:"column:media_refs"`
	ExtensionRef *string   `gorm:"column:extension_ref"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

// TableName pins the table name.
func (Variant) TableName() string { return "offering_variants" }
