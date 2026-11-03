// Package repository is the only layer that touches the organization tables.
//
// Every look-up that serves a tenant-scoped route takes the organization id as
// an argument. A query that can return rows from another tenant is a bug here,
// not a caller's responsibility to filter afterwards.
package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"core-platform/contract/core"
	"core-platform/mod/organization/model"
	"core-platform/platform/uuid"
)

// Store is the organization store.
type Store struct{ db *gorm.DB }

// NewStore builds the store.
func NewStore(db *gorm.DB) *Store { return &Store{db: db} }

// --- Organizations -------------------------------------------------------

// Create inserts an organization.
func (s *Store) Create(ctx context.Context, o *model.Organization) error {
	return s.db.WithContext(ctx).Create(o).Error
}

// Update writes the mutable profile columns.
func (s *Store) Update(ctx context.Context, o *model.Organization) error {
	o.UpdatedAt = time.Now().UTC()
	return s.db.WithContext(ctx).Model(&model.Organization{}).
		Where("id = ?", o.ID).
		Updates(map[string]any{
			"name":        o.Name,
			"description": o.Description,
			"logo_url":    o.LogoURL,
			"email":       o.Email,
			"phone":       o.Phone,
			"website":     o.Website,
			"locale":      o.Locale,
			"updated_at":  o.UpdatedAt,
		}).Error
}

// ByID returns an organization or ErrNotFound.
func (s *Store) ByID(ctx context.Context, id string) (*model.Organization, error) {
	var o model.Organization
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&o).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, core.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// BySlug returns an organization or ErrNotFound.
func (s *Store) BySlug(ctx context.Context, slug string) (*model.Organization, error) {
	var o model.Organization
	err := s.db.WithContext(ctx).Where("slug = ?", slug).First(&o).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, core.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// SlugTaken reports whether a slug is already used, optionally ignoring one id.
func (s *Store) SlugTaken(ctx context.Context, slug, exceptID string) (bool, error) {
	q := s.db.WithContext(ctx).Model(&model.Organization{}).Where("slug = ?", slug)
	if exceptID != "" {
		q = q.Where("id <> ?", exceptID)
	}
	var n int64
	if err := q.Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// List returns a page of organizations.
func (s *Store) List(ctx context.Context, f core.OrganizationFilter) ([]model.Organization, int64, error) {
	q := s.db.WithContext(ctx).Model(&model.Organization{})
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.Query != "" {
		like := "%" + strings.ToLower(f.Query) + "%"
		q = q.Where("lower(name) LIKE ? OR lower(description) LIKE ?", like, like)
	}
	if f.RegionID != "" {
		q = q.Where(`EXISTS (
			SELECT 1 FROM service_areas sa
			WHERE sa.organization_id = organizations.id AND sa.region_id = ?
		)`, f.RegionID)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []model.Organization
	err := q.Order("created_at DESC").
		Limit(f.Limit).Offset(f.Offset).
		Find(&out).Error
	return out, total, err
}

// SetStatus writes the new status and appends an audit event atomically.
func (s *Store) SetStatus(ctx context.Context, id, from, to, actorID, reason string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&model.Organization{}).
			Where("id = ? AND status = ?", id, from).
			Updates(map[string]any{"status": to, "updated_at": time.Now().UTC()})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return core.StateError("the organization changed state; reload and retry")
		}
		return tx.Create(&model.StatusEvent{
			ID:             newID(),
			OrganizationID: id,
			FromStatus:     from,
			ToStatus:       to,
			ActorID:        optional(actorID),
			Reason:         reason,
			CreatedAt:      time.Now().UTC(),
		}).Error
	})
}

// SetVerification writes the verification level.
func (s *Store) SetVerification(ctx context.Context, id, level string) error {
	return s.db.WithContext(ctx).Model(&model.Organization{}).
		Where("id = ?", id).
		Updates(map[string]any{"verification_level": level, "updated_at": time.Now().UTC()}).Error
}

// SetOwner changes the owning account.
func (s *Store) SetOwner(ctx context.Context, id, ownerID string) error {
	return s.db.WithContext(ctx).Model(&model.Organization{}).
		Where("id = ?", id).
		Updates(map[string]any{"owner_id": ownerID, "updated_at": time.Now().UTC()}).Error
}

// --- Branches ------------------------------------------------------------

// CreateBranch inserts a branch.
func (s *Store) CreateBranch(ctx context.Context, b *model.Branch) error {
	return s.db.WithContext(ctx).Create(b).Error
}

// UpdateBranch writes a branch row.
func (s *Store) UpdateBranch(ctx context.Context, b *model.Branch) error {
	b.UpdatedAt = time.Now().UTC()
	return s.db.WithContext(ctx).Model(&model.Branch{}).
		Where("id = ? AND organization_id = ?", b.ID, b.OrganizationID).
		Updates(map[string]any{
			"name":         b.Name,
			"type":         b.Type,
			"address_line": b.AddressLine,
			"country":      b.Country,
			"city":         b.City,
			"district":     b.District,
			"postal_code":  b.PostalCode,
			"latitude":     b.Latitude,
			"longitude":    b.Longitude,
			"phone":        b.Phone,
			"public":       b.Public,
			"status":       b.Status,
			"updated_at":   b.UpdatedAt,
		}).Error
}

// Branch returns one branch or ErrNotFound.
func (s *Store) Branch(ctx context.Context, id string) (*model.Branch, error) {
	var b model.Branch
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&b).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, core.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// Branches lists an organization's branches.
func (s *Store) Branches(ctx context.Context, organizationID string) ([]model.Branch, error) {
	var out []model.Branch
	err := s.db.WithContext(ctx).
		Where("organization_id = ?", organizationID).
		Order("name").
		Find(&out).Error
	return out, err
}

// DeleteBranch removes a branch inside its organization.
func (s *Store) DeleteBranch(ctx context.Context, id, organizationID string) error {
	res := s.db.WithContext(ctx).
		Where("id = ? AND organization_id = ?", id, organizationID).
		Delete(&model.Branch{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return core.ErrNotFound
	}
	return nil
}

// --- Memberships ---------------------------------------------------------

// CreateMembership inserts a membership.
func (s *Store) CreateMembership(ctx context.Context, m *model.Membership) error {
	return s.db.WithContext(ctx).Create(m).Error
}

// Membership returns one membership or ErrNotFound.
func (s *Store) Membership(ctx context.Context, id string) (*model.Membership, error) {
	var m model.Membership
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, core.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// MembershipFor returns the membership linking a user to an organization.
func (s *Store) MembershipFor(ctx context.Context, organizationID, userID string) (*model.Membership, error) {
	var m model.Membership
	err := s.db.WithContext(ctx).
		Where("organization_id = ? AND user_id = ?", organizationID, userID).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, core.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// Memberships lists an organization's members.
func (s *Store) Memberships(ctx context.Context, organizationID string) ([]model.Membership, error) {
	var out []model.Membership
	err := s.db.WithContext(ctx).
		Where("organization_id = ?", organizationID).
		Order("created_at").
		Find(&out).Error
	return out, err
}

// UpdateMembership writes role and status changes.
func (s *Store) UpdateMembership(ctx context.Context, m *model.Membership) error {
	m.UpdatedAt = time.Now().UTC()
	return s.db.WithContext(ctx).Model(&model.Membership{}).
		Where("id = ?", m.ID).
		Updates(map[string]any{
			"branch_id":   m.BranchID,
			"user_id":     m.UserID,
			"role_key":    m.RoleKey,
			"status":      m.Status,
			"accepted_at": m.AcceptedAt,
			"removed_at":  m.RemovedAt,
			"updated_at":  m.UpdatedAt,
		}).Error
}

// AcceptInvitations attaches every pending invitation for an address to a user.
//
// An organization the user already belongs to is skipped: the unique index on
// (organization_id, user_id) would otherwise abort the whole batch for one
// stale invitation.
func (s *Store) AcceptInvitations(ctx context.Context, userID, email string) (int, error) {
	now := time.Now().UTC()
	res := s.db.WithContext(ctx).Exec(`
		UPDATE memberships m
		SET user_id = ?, status = ?, accepted_at = ?, updated_at = ?
		WHERE m.invited_email = ?
		  AND m.status = ?
		  AND m.user_id IS NULL
		  AND NOT EXISTS (
		      SELECT 1 FROM memberships x
		      WHERE x.organization_id = m.organization_id AND x.user_id = ?
		  )`,
		userID, core.MemberStatusActive, now, now,
		email, core.MemberStatusInvited, userID)
	if res.Error != nil {
		return 0, res.Error
	}
	return int(res.RowsAffected), nil
}

// --- Regions -------------------------------------------------------------

// Regions lists the children of a region, or the roots when parentID is empty.
func (s *Store) Regions(ctx context.Context, parentID string) ([]model.Region, error) {
	var out []model.Region
	q := s.db.WithContext(ctx).Model(&model.Region{})
	if parentID == "" {
		q = q.Where("parent_id IS NULL")
	} else {
		q = q.Where("parent_id = ?", parentID)
	}
	err := q.Order("name").Find(&out).Error
	return out, err
}

// Region returns one region or ErrNotFound.
func (s *Store) Region(ctx context.Context, id string) (*model.Region, error) {
	var r model.Region
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&r).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, core.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// CreateRegion inserts a region node.
func (s *Store) CreateRegion(ctx context.Context, r *model.Region) error {
	return s.db.WithContext(ctx).Create(r).Error
}

// --- Service areas -------------------------------------------------------

// AddServiceArea inserts a service area.
func (s *Store) AddServiceArea(ctx context.Context, a *model.ServiceArea) error {
	return s.db.WithContext(ctx).Create(a).Error
}

// ServiceAreas lists an organization's service areas.
func (s *Store) ServiceAreas(ctx context.Context, organizationID string) ([]model.ServiceArea, error) {
	var out []model.ServiceArea
	err := s.db.WithContext(ctx).
		Where("organization_id = ?", organizationID).
		Order("created_at").
		Find(&out).Error
	return out, err
}

// ServiceArea returns one service area or ErrNotFound.
func (s *Store) ServiceArea(ctx context.Context, id string) (*model.ServiceArea, error) {
	var a model.ServiceArea
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, core.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// DeleteServiceArea removes a service area inside its organization.
func (s *Store) DeleteServiceArea(ctx context.Context, id, organizationID string) error {
	res := s.db.WithContext(ctx).
		Where("id = ? AND organization_id = ?", id, organizationID).
		Delete(&model.ServiceArea{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return core.ErrNotFound
	}
	return nil
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func newID() string { return uuid.New() }
