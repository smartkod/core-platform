// Package repository is the only layer that touches the catalog tables.
package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"core-platform/contract/core"
	"core-platform/mod/catalog/model"
)

// Store is the catalog store.
type Store struct{ db *gorm.DB }

// NewStore builds the store.
func NewStore(db *gorm.DB) *Store { return &Store{db: db} }

// --- Taxonomy ------------------------------------------------------------

// CreateTaxonomy inserts a node.
func (s *Store) CreateTaxonomy(ctx context.Context, n *model.TaxonomyNode) error {
	return s.db.WithContext(ctx).Create(n).Error
}

// UpdateTaxonomy writes a node.
func (s *Store) UpdateTaxonomy(ctx context.Context, n *model.TaxonomyNode) error {
	n.UpdatedAt = time.Now().UTC()
	return s.db.WithContext(ctx).Model(&model.TaxonomyNode{}).
		Where("id = ?", n.ID).
		Updates(map[string]any{
			"parent_id":            n.ParentID,
			"slug":                 n.Slug,
			"name":                 n.Name,
			"description":          n.Description,
			"status":               n.Status,
			"published":            n.Published,
			"sort":                 n.Sort,
			"locale":               n.Locale,
			"extension_schema_ref": n.ExtensionSchemaRef,
			"updated_at":           n.UpdatedAt,
		}).Error
}

// Taxonomy returns one node or ErrNotFound.
func (s *Store) Taxonomy(ctx context.Context, id string) (*model.TaxonomyNode, error) {
	var n model.TaxonomyNode
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&n).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, core.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &n, nil
}

// TaxonomyBySlug returns one node or ErrNotFound.
func (s *Store) TaxonomyBySlug(ctx context.Context, slug string) (*model.TaxonomyNode, error) {
	var n model.TaxonomyNode
	err := s.db.WithContext(ctx).Where("slug = ?", slug).First(&n).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, core.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &n, nil
}

// TaxonomyChildren lists the children of a node, or the roots when parentID is
// empty.
func (s *Store) TaxonomyChildren(ctx context.Context, parentID string) ([]model.TaxonomyNode, error) {
	var out []model.TaxonomyNode
	q := s.db.WithContext(ctx).Model(&model.TaxonomyNode{}).
		Where("published = true AND status = ?", core.TaxonomyStatusActive)
	if parentID == "" {
		q = q.Where("parent_id IS NULL")
	} else {
		q = q.Where("parent_id = ?", parentID)
	}
	err := q.Order("sort, name").Find(&out).Error
	return out, err
}

// --- Organization categories --------------------------------------------

// SetOrganizationCategories replaces the category set for one organization.
func (s *Store) SetOrganizationCategories(ctx context.Context, organizationID string, ids []string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(
			"DELETE FROM organization_categories WHERE organization_id = ?",
			organizationID).Error; err != nil {
			return err
		}
		for _, id := range ids {
			if err := tx.Exec(
				"INSERT INTO organization_categories (organization_id, taxonomy_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
				organizationID, id).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// OrganizationCategories lists the categories an organization belongs to.
func (s *Store) OrganizationCategories(ctx context.Context, organizationID string) ([]model.TaxonomyNode, error) {
	var out []model.TaxonomyNode
	err := s.db.WithContext(ctx).Raw(`
		SELECT t.* FROM taxonomy_nodes t
		JOIN organization_categories oc ON oc.taxonomy_id = t.id
		WHERE oc.organization_id = ?
		ORDER BY t.sort, t.name`, organizationID).Scan(&out).Error
	return out, err
}

// --- Offerings -----------------------------------------------------------

// CreateOffering inserts an offering.
func (s *Store) CreateOffering(ctx context.Context, o *model.Offering) error {
	return s.db.WithContext(ctx).Create(o).Error
}

// UpdateOffering writes the editable columns.
func (s *Store) UpdateOffering(ctx context.Context, o *model.Offering) error {
	o.UpdatedAt = time.Now().UTC()
	return s.db.WithContext(ctx).Model(&model.Offering{}).
		Where("id = ? AND organization_id = ?", o.ID, o.OrganizationID).
		Updates(map[string]any{
			"branch_id":      o.BranchID,
			"taxonomy_id":    o.TaxonomyID,
			"name":           o.Name,
			"slug":           o.Slug,
			"summary":        o.Summary,
			"description":    o.Description,
			"media_refs":     o.MediaRefs,
			"service_mode":   o.ServiceMode,
			"visibility":     o.Visibility,
			"locale":         o.Locale,
			"tags":           o.Tags,
			"price_from":     o.PriceFrom,
			"price_currency": o.PriceCurrency,
			"extension_ref":  o.ExtensionRef,
			"updated_at":     o.UpdatedAt,
		}).Error
}

// Offering returns one offering or ErrNotFound.
func (s *Store) Offering(ctx context.Context, id string) (*model.Offering, error) {
	var o model.Offering
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&o).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, core.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// OfferingSlugTaken reports whether a slug is used inside one organization.
func (s *Store) OfferingSlugTaken(ctx context.Context, organizationID, slug, exceptID string) (bool, error) {
	q := s.db.WithContext(ctx).Model(&model.Offering{}).
		Where("organization_id = ? AND slug = ?", organizationID, slug)
	if exceptID != "" {
		q = q.Where("id <> ?", exceptID)
	}
	var n int64
	if err := q.Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// ListOfferings returns a page of offerings.
func (s *Store) ListOfferings(ctx context.Context, f core.OfferingFilter) ([]model.Offering, int64, error) {
	q := s.db.WithContext(ctx).Model(&model.Offering{})
	if f.OrganizationID != "" {
		q = q.Where("offerings.organization_id = ?", f.OrganizationID)
	}
	if f.BranchID != "" {
		q = q.Where("offerings.branch_id = ?", f.BranchID)
	}
	if f.TaxonomyID != "" {
		// A category match includes its descendants, otherwise browsing a
		// parent category would show nothing.
		q = q.Where(`offerings.taxonomy_id IN (
			WITH RECURSIVE tree AS (
				SELECT id FROM taxonomy_nodes WHERE id = ?
				UNION ALL
				SELECT t.id FROM taxonomy_nodes t JOIN tree ON t.parent_id = tree.id
			)
			SELECT id FROM tree
		)`, f.TaxonomyID)
	}
	if f.Status != "" {
		q = q.Where("offerings.status = ?", f.Status)
	}
	if f.Visibility != "" {
		q = q.Where("offerings.visibility = ?", f.Visibility)
	}
	if f.RegionID != "" {
		q = q.Where(`offerings.organization_id IN (
			SELECT organization_id FROM service_areas WHERE region_id = ?
		)`, f.RegionID)
	}
	if f.Query != "" {
		like := "%" + strings.ToLower(f.Query) + "%"
		q = q.Where("lower(offerings.name) LIKE ? OR lower(offerings.summary) LIKE ?", like, like)
	}
	// Public discovery must not surface an offering whose business is not
	// published. This is a read-only containment join: catalog never writes the
	// organizations table, it only refuses to leak a suspended tenant.
	if f.Status == core.OfferingStatusPublished && f.Visibility == core.VisibilityPublic {
		q = q.Joins("JOIN organizations o ON o.id = offerings.organization_id").
			Where("o.status = ?", core.OrgStatusActive)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []model.Offering
	err := q.Order("offerings.created_at DESC").
		Limit(f.Limit).Offset(f.Offset).
		Find(&out).Error
	return out, total, err
}

// SetOfferingStatus applies a publishing transition atomically.
func (s *Store) SetOfferingStatus(ctx context.Context, id, organizationID, from, to string) error {
	res := s.db.WithContext(ctx).Model(&model.Offering{}).
		Where("id = ? AND organization_id = ? AND status = ?", id, organizationID, from).
		Updates(map[string]any{"status": to, "updated_at": time.Now().UTC()})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return core.StateError("the offering changed state; reload and retry")
	}
	return nil
}

// DeleteOffering removes an offering inside its organization.
func (s *Store) DeleteOffering(ctx context.Context, id, organizationID string) error {
	res := s.db.WithContext(ctx).
		Where("id = ? AND organization_id = ?", id, organizationID).
		Delete(&model.Offering{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return core.ErrNotFound
	}
	return nil
}

// --- Variants ------------------------------------------------------------

// CreateVariant inserts a variant.
func (s *Store) CreateVariant(ctx context.Context, v *model.Variant) error {
	return s.db.WithContext(ctx).Create(v).Error
}

// UpdateVariant writes a variant.
func (s *Store) UpdateVariant(ctx context.Context, v *model.Variant) error {
	v.UpdatedAt = time.Now().UTC()
	return s.db.WithContext(ctx).Model(&model.Variant{}).
		Where("id = ? AND offering_id = ?", v.ID, v.OfferingID).
		Updates(map[string]any{
			"name":          v.Name,
			"status":        v.Status,
			"visibility":    v.Visibility,
			"media_refs":    v.MediaRefs,
			"extension_ref": v.ExtensionRef,
			"updated_at":    v.UpdatedAt,
		}).Error
}

// Variants lists an offering's variants.
func (s *Store) Variants(ctx context.Context, offeringID string) ([]model.Variant, error) {
	var out []model.Variant
	err := s.db.WithContext(ctx).
		Where("offering_id = ?", offeringID).
		Order("created_at").
		Find(&out).Error
	return out, err
}

// Variant returns one variant or ErrNotFound.
func (s *Store) Variant(ctx context.Context, id string) (*model.Variant, error) {
	var v model.Variant
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&v).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, core.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// DeleteVariant removes a variant inside its offering.
func (s *Store) DeleteVariant(ctx context.Context, id, offeringID string) error {
	res := s.db.WithContext(ctx).
		Where("id = ? AND offering_id = ?", id, offeringID).
		Delete(&model.Variant{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return core.ErrNotFound
	}
	return nil
}
