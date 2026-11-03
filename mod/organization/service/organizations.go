// Package service implements the organization rules.
package service

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"core-platform/contract/core"
	"core-platform/mod/organization/model"
	"core-platform/mod/organization/repository"
	"core-platform/platform/uuid"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,118}[a-z0-9]$`)

// Organizations implements core.Organizations.
type Organizations struct {
	store *repository.Store
	authz core.Authorizer
}

// New builds the organization service.
func New(store *repository.Store, authz core.Authorizer) *Organizations {
	return &Organizations{store: store, authz: authz}
}

// Create creates an organization and its owner grant in one operation.
//
// The owner is taken from the input, which the handler fills from the
// authenticated principal — never from the request body.
func (s *Organizations) Create(ctx context.Context, in core.CreateOrganizationInput) (*core.Organization, error) {
	if in.OwnerID == "" {
		return nil, core.ErrUnauthorized
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, core.Invalid("name", "a name is required")
	}
	slug := strings.ToLower(strings.TrimSpace(in.Slug))
	if slug == "" {
		return nil, core.Invalid("slug", "a slug is required")
	}
	if !slugPattern.MatchString(slug) {
		return nil, core.Invalid("slug", "use lowercase letters, digits and hyphens")
	}

	taken, err := s.store.SlugTaken(ctx, slug, "")
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, core.ErrConflict
	}

	now := time.Now().UTC()
	rec := &model.Organization{
		ID:                uuid.New(),
		Slug:              slug,
		Name:              name,
		Email:             strings.TrimSpace(in.Email),
		Phone:             strings.TrimSpace(in.Phone),
		Website:           strings.TrimSpace(in.Website),
		OwnerID:           in.OwnerID,
		Status:            core.OrgStatusDraft,
		VerificationLevel: core.VerificationNone,
		Locale:            defaultString(in.Locale, "en"),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := s.store.Create(ctx, rec); err != nil {
		return nil, err
	}

	// The owner membership and the owner grant are what make the organization
	// usable; failing after the insert would leave an organization nobody can
	// administer, so the error is surfaced and the caller can retry creation.
	if err := s.store.CreateMembership(ctx, &model.Membership{
		ID:             uuid.New(),
		OrganizationID: rec.ID,
		UserID:         &rec.OwnerID,
		RoleKey:        "owner",
		Status:         core.MemberStatusActive,
		InvitedAt:      now,
		AcceptedAt:     &now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		return nil, err
	}

	if s.authz != nil {
		if _, err := s.authz.Grant(ctx, core.GrantInput{
			UserID:    rec.OwnerID,
			RoleKey:   "owner",
			Scope:     core.ScopeOrganization,
			ScopeID:   rec.ID,
			GrantedBy: rec.OwnerID,
		}); err != nil {
			return nil, err
		}
	}

	return toCoreOrganization(rec), nil
}

// Update writes mutable profile fields.
func (s *Organizations) Update(ctx context.Context, in core.UpdateOrganizationInput) (*core.Organization, error) {
	rec, err := s.store.ByID(ctx, in.ID)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return nil, core.Invalid("name", "a name is required")
		}
		rec.Name = name
	}
	if in.Description != nil {
		rec.Description = *in.Description
	}
	if in.LogoURL != nil {
		rec.LogoURL = *in.LogoURL
	}
	if in.Email != nil {
		rec.Email = strings.TrimSpace(*in.Email)
	}
	if in.Phone != nil {
		rec.Phone = strings.TrimSpace(*in.Phone)
	}
	if in.Website != nil {
		rec.Website = strings.TrimSpace(*in.Website)
	}
	if in.Locale != nil {
		rec.Locale = *in.Locale
	}

	if err := s.store.Update(ctx, rec); err != nil {
		return nil, err
	}
	return toCoreOrganization(rec), nil
}

// ByID returns one organization.
func (s *Organizations) ByID(ctx context.Context, id string) (*core.Organization, error) {
	rec, err := s.store.ByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toCoreOrganization(rec), nil
}

// BySlug returns one organization.
func (s *Organizations) BySlug(ctx context.Context, slug string) (*core.Organization, error) {
	rec, err := s.store.BySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return toCoreOrganization(rec), nil
}

// List returns a page of organizations.
func (s *Organizations) List(ctx context.Context, f core.OrganizationFilter) ([]core.Organization, int64, error) {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 20
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	rows, total, err := s.store.List(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]core.Organization, 0, len(rows))
	for i := range rows {
		out = append(out, *toCoreOrganization(&rows[i]))
	}
	return out, total, nil
}

// SetStatus applies a publishing decision after checking the transition.
func (s *Organizations) SetStatus(ctx context.Context, id, status, actorID string) (*core.Organization, error) {
	rec, err := s.store.ByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !allowedTransition(rec.Status, status) {
		return nil, core.StateError("cannot move from " + rec.Status + " to " + status)
	}
	if err := s.store.SetStatus(ctx, id, rec.Status, status, actorID, ""); err != nil {
		return nil, err
	}
	rec.Status = status
	return toCoreOrganization(rec), nil
}

// TransferOwnership moves the owner grant to another active member.
func (s *Organizations) TransferOwnership(ctx context.Context, id, newOwnerID, actorID string) error {
	org, err := s.store.ByID(ctx, id)
	if err != nil {
		return err
	}
	if org.OwnerID == newOwnerID {
		return nil
	}

	m, err := s.store.MembershipFor(ctx, id, newOwnerID)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return core.Invalid("new_owner_id", "the new owner must already be a member")
		}
		return err
	}
	if m.Status != core.MemberStatusActive {
		return core.Invalid("new_owner_id", "the new owner must have an active membership")
	}

	now := time.Now().UTC()
	m.RoleKey = "owner"
	m.AcceptedAt = firstTime(m.AcceptedAt, now)
	if err := s.store.UpdateMembership(ctx, m); err != nil {
		return err
	}

	oldOwnerID := org.OwnerID
	old, err := s.store.MembershipFor(ctx, id, oldOwnerID)
	if err == nil {
		old.RoleKey = "manager"
		if err := s.store.UpdateMembership(ctx, old); err != nil {
			return err
		}
	}

	if err := s.store.SetOwner(ctx, id, newOwnerID); err != nil {
		return err
	}

	if s.authz != nil {
		if _, err := s.authz.Grant(ctx, core.GrantInput{
			UserID:    newOwnerID,
			RoleKey:   "owner",
			Scope:     core.ScopeOrganization,
			ScopeID:   id,
			GrantedBy: actorID,
		}); err != nil && !errors.Is(err, core.ErrConflict) {
			return err
		}
		if err := s.revokeRole(ctx, oldOwnerID, "owner", id, actorID); err != nil {
			return err
		}
	}
	return nil
}

// --- Branches ------------------------------------------------------------

// CreateBranch adds a branch.
func (s *Organizations) CreateBranch(ctx context.Context, in core.BranchInput) (*core.Branch, error) {
	if in.OrganizationID == "" {
		return nil, core.Invalid("organization_id", "an organization is required")
	}
	if strings.TrimSpace(in.Name) == "" {
		return nil, core.Invalid("name", "a name is required")
	}
	if in.Type == "" {
		in.Type = core.BranchTypePhysical
	}
	if !validBranchType(in.Type) {
		return nil, core.Invalid("type", "unknown branch type")
	}
	if in.Status == "" {
		in.Status = core.BranchStatusActive
	}

	now := time.Now().UTC()
	rec := &model.Branch{
		ID:             uuid.New(),
		OrganizationID: in.OrganizationID,
		Name:           strings.TrimSpace(in.Name),
		Type:           in.Type,
		AddressLine:    in.AddressLine,
		Country:        strings.ToUpper(in.Country),
		City:           in.City,
		District:       in.District,
		PostalCode:     in.PostalCode,
		Latitude:       optionalFloat(in.Latitude),
		Longitude:      optionalFloat(in.Longitude),
		Phone:          in.Phone,
		Public:         in.Public,
		Status:         in.Status,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.store.CreateBranch(ctx, rec); err != nil {
		return nil, err
	}
	return toCoreBranch(rec), nil
}

// UpdateBranch writes a branch.
func (s *Organizations) UpdateBranch(ctx context.Context, in core.BranchInput) (*core.Branch, error) {
	rec, err := s.store.Branch(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	if in.OrganizationID != "" && in.OrganizationID != rec.OrganizationID {
		return nil, core.ErrNotFound
	}
	if in.Name != "" {
		rec.Name = strings.TrimSpace(in.Name)
	}
	if in.Type != "" {
		if !validBranchType(in.Type) {
			return nil, core.Invalid("type", "unknown branch type")
		}
		rec.Type = in.Type
	}
	rec.AddressLine = in.AddressLine
	rec.Country = strings.ToUpper(in.Country)
	rec.City = in.City
	rec.District = in.District
	rec.PostalCode = in.PostalCode
	rec.Latitude = optionalFloat(in.Latitude)
	rec.Longitude = optionalFloat(in.Longitude)
	rec.Phone = in.Phone
	rec.Public = in.Public
	if in.Status != "" {
		rec.Status = in.Status
	}

	if err := s.store.UpdateBranch(ctx, rec); err != nil {
		return nil, err
	}
	return toCoreBranch(rec), nil
}

// Branches lists an organization's branches.
func (s *Organizations) Branches(ctx context.Context, organizationID string) ([]core.Branch, error) {
	rows, err := s.store.Branches(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	out := make([]core.Branch, 0, len(rows))
	for i := range rows {
		out = append(out, *toCoreBranch(&rows[i]))
	}
	return out, nil
}

// Branch returns one branch.
func (s *Organizations) Branch(ctx context.Context, id string) (*core.Branch, error) {
	rec, err := s.store.Branch(ctx, id)
	if err != nil {
		return nil, err
	}
	return toCoreBranch(rec), nil
}

// DeleteBranch removes a branch when the actor is allowed to.
func (s *Organizations) DeleteBranch(ctx context.Context, id, actorID string) error {
	rec, err := s.store.Branch(ctx, id)
	if err != nil {
		return err
	}
	return s.store.DeleteBranch(ctx, id, rec.OrganizationID)
}

// --- Memberships ---------------------------------------------------------

// InviteMember creates an invitation.
func (s *Organizations) InviteMember(ctx context.Context, in core.InviteMemberInput) (*core.Membership, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if email == "" {
		return nil, core.Invalid("email", "an e-mail address is required")
	}
	if !validMemberRole(in.RoleKey) {
		return nil, core.Invalid("role_key", "unknown role for a member")
	}
	if in.RoleKey == "owner" {
		// Ownership transfer is an explicit operation, not an invitation.
		return nil, core.Invalid("role_key", "transfer ownership instead of inviting a second owner")
	}

	existing, err := s.store.Memberships(ctx, in.OrganizationID)
	if err == nil {
		for _, m := range existing {
			if m.Status == core.MemberStatusRemoved {
				continue
			}
			if strings.EqualFold(m.InvitedEmail, email) {
				return nil, core.ErrConflict
			}
		}
	}

	now := time.Now().UTC()
	rec := &model.Membership{
		ID:             uuid.New(),
		OrganizationID: in.OrganizationID,
		BranchID:       optional(in.BranchID),
		InvitedEmail:   email,
		RoleKey:        in.RoleKey,
		Status:         core.MemberStatusInvited,
		InvitedBy:      optional(in.InvitedBy),
		InvitedAt:      now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.store.CreateMembership(ctx, rec); err != nil {
		return nil, err
	}
	return toCoreMembership(rec), nil
}

// AcceptInvitation activates an invitation for an account.
func (s *Organizations) AcceptInvitation(ctx context.Context, membershipID, userID string) (*core.Membership, error) {
	rec, err := s.store.Membership(ctx, membershipID)
	if err != nil {
		return nil, err
	}
	if rec.Status != core.MemberStatusInvited {
		return nil, core.StateError("the invitation is no longer pending")
	}
	// A user may already belong to the organization through another invitation.
	// Activating this one would violate the (organization, user) uniqueness and
	// leave the request looking like a server error.
	if existing, err := s.store.MembershipFor(ctx, rec.OrganizationID, userID); err == nil &&
		existing.ID != rec.ID {
		return nil, core.ErrConflict
	}

	now := time.Now().UTC()
	rec.UserID = &userID
	rec.Status = core.MemberStatusActive
	rec.AcceptedAt = &now
	if err := s.store.UpdateMembership(ctx, rec); err != nil {
		return nil, err
	}

	if s.authz != nil {
		if _, err := s.authz.Grant(ctx, core.GrantInput{
			UserID:    userID,
			RoleKey:   rec.RoleKey,
			Scope:     core.ScopeOrganization,
			ScopeID:   rec.OrganizationID,
			GrantedBy: deref(rec.InvitedBy),
		}); err != nil && !errors.Is(err, core.ErrConflict) {
			return nil, err
		}
	}
	return toCoreMembership(rec), nil
}

// UpdateMemberRole changes a member's role and the matching grant.
func (s *Organizations) UpdateMemberRole(ctx context.Context, membershipID, roleKey, actorID string) (*core.Membership, error) {
	if !validMemberRole(roleKey) {
		return nil, core.Invalid("role_key", "unknown role for a member")
	}

	rec, err := s.store.Membership(ctx, membershipID)
	if err != nil {
		return nil, err
	}
	if rec.RoleKey == "owner" && roleKey != "owner" {
		return nil, core.Invalid("role_key", "use ownership transfer to change the owner")
	}
	if roleKey == "owner" {
		return nil, core.Invalid("role_key", "use ownership transfer to change the owner")
	}

	oldRole := rec.RoleKey
	rec.RoleKey = roleKey
	if err := s.store.UpdateMembership(ctx, rec); err != nil {
		return nil, err
	}

	if s.authz != nil && rec.UserID != nil {
		if err := s.revokeRole(ctx, *rec.UserID, oldRole, rec.OrganizationID, actorID); err != nil {
			return nil, err
		}
		if _, err := s.authz.Grant(ctx, core.GrantInput{
			UserID:    *rec.UserID,
			RoleKey:   roleKey,
			Scope:     core.ScopeOrganization,
			ScopeID:   rec.OrganizationID,
			GrantedBy: actorID,
		}); err != nil && !errors.Is(err, core.ErrConflict) {
			return nil, err
		}
	}
	return toCoreMembership(rec), nil
}

// RemoveMember deactivates a membership, refusing to strand the organization.
func (s *Organizations) RemoveMember(ctx context.Context, membershipID, actorID string) error {
	rec, err := s.store.Membership(ctx, membershipID)
	if err != nil {
		return err
	}
	if rec.RoleKey == "owner" {
		return core.StateError("transfer ownership before removing the owner")
	}

	now := time.Now().UTC()
	rec.Status = core.MemberStatusRemoved
	rec.RemovedAt = &now
	if err := s.store.UpdateMembership(ctx, rec); err != nil {
		return err
	}

	if s.authz != nil && rec.UserID != nil {
		return s.revokeRole(ctx, *rec.UserID, rec.RoleKey, rec.OrganizationID, actorID)
	}
	return nil
}

// Members lists an organization's members.
func (s *Organizations) Members(ctx context.Context, organizationID string) ([]core.Membership, error) {
	rows, err := s.store.Memberships(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	out := make([]core.Membership, 0, len(rows))
	for i := range rows {
		out = append(out, *toCoreMembership(&rows[i]))
	}
	return out, nil
}

// MembershipFor returns the membership linking a user to an organization.
func (s *Organizations) MembershipFor(ctx context.Context, organizationID, userID string) (*core.Membership, error) {
	rec, err := s.store.MembershipFor(ctx, organizationID, userID)
	if err != nil {
		return nil, err
	}
	return toCoreMembership(rec), nil
}

// Membership returns one membership by id.
//
// Handlers use it to prove that a membership actually belongs to the tenant in
// the route before acting on it.
func (s *Organizations) Membership(ctx context.Context, id string) (*core.Membership, error) {
	rec, err := s.store.Membership(ctx, id)
	if err != nil {
		return nil, err
	}
	return toCoreMembership(rec), nil
}

// AcceptPendingInvitations attaches pending invitations to a new account.
func (s *Organizations) AcceptPendingInvitations(ctx context.Context, userID, email string) (int, error) {
	return s.store.AcceptInvitations(ctx, userID, email)
}

// --- Regions and service areas ------------------------------------------

// Regions lists region children.
func (s *Organizations) Regions(ctx context.Context, parentID string) ([]core.Region, error) {
	rows, err := s.store.Regions(ctx, parentID)
	if err != nil {
		return nil, err
	}
	out := make([]core.Region, 0, len(rows))
	for i := range rows {
		out = append(out, toCoreRegion(&rows[i]))
	}
	return out, nil
}

// CreateRegion adds a region node.
func (s *Organizations) CreateRegion(ctx context.Context, in core.Region) (*core.Region, error) {
	if in.Name == "" || in.Code == "" {
		return nil, core.Invalid("name", "a name and code are required")
	}
	switch in.Level {
	case core.RegionLevelCountry, core.RegionLevelCity, core.RegionLevelDistrict:
	default:
		return nil, core.Invalid("level", "unknown region level")
	}

	rec := &model.Region{
		ID:       uuid.New(),
		ParentID: optional(in.ParentID),
		Level:    in.Level,
		Code:     in.Code,
		Name:     in.Name,
		Country:  strings.ToUpper(in.Country),
		Locale:   defaultString(in.Locale, "en"),
	}
	if err := s.store.CreateRegion(ctx, rec); err != nil {
		return nil, err
	}
	out := toCoreRegion(rec)
	return &out, nil
}

// AddServiceArea records where an organization serves customers.
func (s *Organizations) AddServiceArea(ctx context.Context, in core.ServiceArea) (*core.ServiceArea, error) {
	if in.OrganizationID == "" {
		return nil, core.Invalid("organization_id", "an organization is required")
	}
	switch in.Mode {
	case core.ServiceModeOnSite, core.ServiceModeOnline, core.ServiceModeAtCustomer, core.ServiceModeHybrid:
	default:
		return nil, core.Invalid("mode", "unknown service mode")
	}
	if in.BranchID != "" {
		b, err := s.store.Branch(ctx, in.BranchID)
		if err != nil {
			return nil, err
		}
		if b.OrganizationID != in.OrganizationID {
			return nil, core.Invalid("branch_id", "the branch belongs to another organization")
		}
	}
	if in.RegionID != "" {
		if _, err := s.store.Region(ctx, in.RegionID); err != nil {
			return nil, err
		}
	}

	rec := &model.ServiceArea{
		ID:             uuid.New(),
		OrganizationID: in.OrganizationID,
		BranchID:       optional(in.BranchID),
		RegionID:       optional(in.RegionID),
		Mode:           in.Mode,
		CreatedAt:      time.Now().UTC(),
	}
	if err := s.store.AddServiceArea(ctx, rec); err != nil {
		return nil, err
	}
	return &core.ServiceArea{
		ID:             rec.ID,
		OrganizationID: rec.OrganizationID,
		BranchID:       deref(rec.BranchID),
		RegionID:       deref(rec.RegionID),
		Mode:           rec.Mode,
		CreatedAt:      rec.CreatedAt,
	}, nil
}

// ServiceAreas lists an organization's service areas.
func (s *Organizations) ServiceAreas(ctx context.Context, organizationID string) ([]core.ServiceArea, error) {
	rows, err := s.store.ServiceAreas(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	out := make([]core.ServiceArea, 0, len(rows))
	for i := range rows {
		out = append(out, core.ServiceArea{
			ID:             rows[i].ID,
			OrganizationID: rows[i].OrganizationID,
			BranchID:       deref(rows[i].BranchID),
			RegionID:       deref(rows[i].RegionID),
			Mode:           rows[i].Mode,
			CreatedAt:      rows[i].CreatedAt,
		})
	}
	return out, nil
}

// ServiceArea returns one service area.
func (s *Organizations) ServiceArea(ctx context.Context, id string) (*core.ServiceArea, error) {
	rec, err := s.store.ServiceArea(ctx, id)
	if err != nil {
		return nil, err
	}
	return &core.ServiceArea{
		ID:             rec.ID,
		OrganizationID: rec.OrganizationID,
		BranchID:       deref(rec.BranchID),
		RegionID:       deref(rec.RegionID),
		Mode:           rec.Mode,
		CreatedAt:      rec.CreatedAt,
	}, nil
}

// RemoveServiceArea deletes a service area inside its organization.
func (s *Organizations) RemoveServiceArea(ctx context.Context, id, actorID string) error {
	rec, err := s.store.ServiceArea(ctx, id)
	if err != nil {
		return err
	}
	return s.store.DeleteServiceArea(ctx, id, rec.OrganizationID)
}

// revokeRole revokes one role's assignment in one organization.
func (s *Organizations) revokeRole(ctx context.Context, userID, roleKey, organizationID, actorID string) error {
	list, err := s.authz.Assignments(ctx, userID)
	if err != nil {
		return err
	}
	for _, a := range list {
		if a.RoleKey == roleKey &&
			a.Scope == core.ScopeOrganization &&
			a.ScopeID == organizationID &&
			a.RevokedAt == nil {
			if err := s.authz.Revoke(ctx, a.ID, actorID); err != nil {
				return err
			}
		}
	}
	return nil
}

// --- Conversion ----------------------------------------------------------

func toCoreOrganization(m *model.Organization) *core.Organization {
	return &core.Organization{
		ID:                m.ID,
		Slug:              m.Slug,
		Name:              m.Name,
		Description:       m.Description,
		LogoURL:           m.LogoURL,
		Email:             m.Email,
		Phone:             m.Phone,
		Website:           m.Website,
		OwnerID:           m.OwnerID,
		Status:            m.Status,
		VerificationLevel: m.VerificationLevel,
		Locale:            m.Locale,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
	}
}

func toCoreBranch(b *model.Branch) *core.Branch {
	out := &core.Branch{
		ID:             b.ID,
		OrganizationID: b.OrganizationID,
		Name:           b.Name,
		Type:           b.Type,
		AddressLine:    b.AddressLine,
		Country:        b.Country,
		City:           b.City,
		District:       b.District,
		PostalCode:     b.PostalCode,
		Phone:          b.Phone,
		Public:         b.Public,
		Status:         b.Status,
		CreatedAt:      b.CreatedAt,
		UpdatedAt:      b.UpdatedAt,
	}
	if b.Latitude != nil {
		out.Latitude = *b.Latitude
	}
	if b.Longitude != nil {
		out.Longitude = *b.Longitude
	}
	return out
}

func toCoreMembership(m *model.Membership) *core.Membership {
	out := &core.Membership{
		ID:             m.ID,
		OrganizationID: m.OrganizationID,
		BranchID:       deref(m.BranchID),
		UserID:         deref(m.UserID),
		InvitedEmail:   m.InvitedEmail,
		RoleKey:        m.RoleKey,
		Status:         m.Status,
		InvitedBy:      deref(m.InvitedBy),
		InvitedAt:      m.InvitedAt,
		AcceptedAt:     m.AcceptedAt,
		RemovedAt:      m.RemovedAt,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
	return out
}

func toCoreRegion(r *model.Region) core.Region {
	out := core.Region{
		ID:      r.ID,
		Level:   r.Level,
		Code:    r.Code,
		Name:    r.Name,
		Country: r.Country,
		Locale:  r.Locale,
	}
	if r.ParentID != nil {
		out.ParentID = *r.ParentID
	}
	return out
}

// --- Small helpers -------------------------------------------------------

func allowedTransition(from, to string) bool {
	switch from {
	case core.OrgStatusDraft:
		return to == core.OrgStatusPendingVerification || to == core.OrgStatusClosed
	case core.OrgStatusPendingVerification:
		return to == core.OrgStatusActive || to == core.OrgStatusDraft || to == core.OrgStatusClosed
	case core.OrgStatusActive:
		return to == core.OrgStatusSuspended || to == core.OrgStatusClosed
	case core.OrgStatusSuspended:
		return to == core.OrgStatusActive || to == core.OrgStatusClosed
	case core.OrgStatusClosed:
		return false
	}
	return false
}

func validBranchType(t string) bool {
	return t == core.BranchTypePhysical || t == core.BranchTypeServiceArea || t == core.BranchTypeOnline
}

func validMemberRole(r string) bool {
	switch r {
	case "owner", "manager", "staff", "member":
		return true
	}
	return false
}

func optionalFloat(v float64) *float64 {
	if v == 0 {
		return nil
	}
	return &v
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func defaultString(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

func firstTime(t *time.Time, def time.Time) *time.Time {
	if t != nil {
		return t
	}
	return &def
}
