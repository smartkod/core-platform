package core

import (
	"context"
	"time"
)

// Organization statuses. They describe the platform's publishing decision, not
// the business's own workflow.
const (
	OrgStatusDraft               = "draft"
	OrgStatusPendingVerification = "pending_verification"
	OrgStatusActive              = "active"
	OrgStatusSuspended           = "suspended"
	OrgStatusClosed              = "closed"
)

// Organization verification levels are policy markers, not sector documents.
const (
	VerificationNone     = "none"
	VerificationBasic    = "basic"
	VerificationVerified = "verified"
)

// Organization is the shared identity of a business or institution on the
// platform.
//
// It holds only what every sector needs. Opening hours, capacity, payroll and
// licensing workflow belong to an engine or extension, never here.
type Organization struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	LogoURL     string `json:"logo_url"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Website     string `json:"website"`
	// OwnerID is the account that created the organization and holds its
	// owner role. Ownership transfer is an explicit operation, not an update.
	OwnerID string `json:"owner_id"`
	Status  string `json:"status"`
	// VerificationLevel is set by a platform decision.
	VerificationLevel string    `json:"verification_level"`
	Locale            string    `json:"locale"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Branch is a physical location, a served area or an online channel.
const (
	BranchTypePhysical    = "physical"
	BranchTypeServiceArea = "service_area"
	BranchTypeOnline      = "online"
)

// Branch statuses.
const (
	BranchStatusActive    = "active"
	BranchStatusInactive  = "inactive"
	BranchStatusSuspended = "suspended"
)

// Branch is one place or channel where an organization operates.
type Branch struct {
	ID             string  `json:"id"`
	OrganizationID string  `json:"organization_id"`
	Name           string  `json:"name"`
	Type           string  `json:"type"`
	AddressLine    string  `json:"address_line"`
	Country        string  `json:"country"`
	City           string  `json:"city"`
	District       string  `json:"district"`
	PostalCode     string  `json:"postal_code"`
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	Phone          string  `json:"phone"`
	// Public controls whether the exact address is shown in discovery. A
	// sensitive location stays hidden while remaining usable internally.
	Public    bool      `json:"public"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Membership statuses.
const (
	MemberStatusInvited   = "invited"
	MemberStatusActive    = "active"
	MemberStatusSuspended = "suspended"
	MemberStatusRemoved   = "removed"
)

// Membership links an account to an organization.
//
// It is not the same thing as an account: an invited person may not have signed
// up yet, which is why the invitation carries an e-mail address and the user id
// is filled in only when the invitation is accepted.
type Membership struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organization_id"`
	BranchID       string     `json:"branch_id,omitempty"`
	UserID         string     `json:"user_id,omitempty"`
	InvitedEmail   string     `json:"invited_email,omitempty"`
	RoleKey        string     `json:"role_key"`
	Status         string     `json:"status"`
	InvitedBy      string     `json:"invited_by,omitempty"`
	InvitedAt      time.Time  `json:"invited_at"`
	AcceptedAt     *time.Time `json:"accepted_at,omitempty"`
	RemovedAt      *time.Time `json:"removed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Region is a generic geographic reference: country, city or district.
const (
	RegionLevelCountry  = "country"
	RegionLevelCity     = "city"
	RegionLevelDistrict = "district"
)

// Region is a node of the standard geographic reference tree.
//
// It is intentionally generic: no postal-code routing, no delivery zones and no
// distance calculation live here.
type Region struct {
	ID       string `json:"id"`
	ParentID string `json:"parent_id,omitempty"`
	Level    string `json:"level"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Country  string `json:"country"`
	// Locale is the language the name is written in.
	Locale string `json:"locale"`
}

// Service modes for an area.
const (
	ServiceModeOnSite     = "on_site"
	ServiceModeOnline     = "online"
	ServiceModeAtCustomer = "at_customer"
	ServiceModeHybrid     = "hybrid"
)

// ServiceArea states where an organization serves customers.
type ServiceArea struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	BranchID       string    `json:"branch_id,omitempty"`
	RegionID       string    `json:"region_id,omitempty"`
	Mode           string    `json:"mode"`
	CreatedAt      time.Time `json:"created_at"`
}

// CreateOrganizationInput is one organization creation request.
type CreateOrganizationInput struct {
	Name    string
	Slug    string
	Email   string
	Phone   string
	Website string
	Locale  string
	// OwnerID is the account that becomes owner. It is taken from the
	// authenticated principal, never from the request body.
	OwnerID string
}

// UpdateOrganizationInput carries the mutable profile fields.
//
// Nil means "leave unchanged", which is what lets a client send only the field
// it edited.
type UpdateOrganizationInput struct {
	ID          string
	Name        *string
	Description *string
	LogoURL     *string
	Email       *string
	Phone       *string
	Website     *string
	Locale      *string
}

// OrganizationFilter narrows a listing.
type OrganizationFilter struct {
	Status   string
	RegionID string
	Query    string
	Limit    int
	Offset   int
}

// BranchInput creates or replaces a branch.
type BranchInput struct {
	ID             string
	OrganizationID string
	Name           string
	Type           string
	AddressLine    string
	Country        string
	City           string
	District       string
	PostalCode     string
	Latitude       float64
	Longitude      float64
	Phone          string
	Public         bool
	Status         string
}

// InviteMemberInput creates a membership invitation.
type InviteMemberInput struct {
	OrganizationID string
	BranchID       string
	Email          string
	RoleKey        string
	InvitedBy      string
}

// Organizations is the organization service surface other modules use.
type Organizations interface {
	Create(ctx context.Context, in CreateOrganizationInput) (*Organization, error)
	Update(ctx context.Context, in UpdateOrganizationInput) (*Organization, error)
	ByID(ctx context.Context, id string) (*Organization, error)
	BySlug(ctx context.Context, slug string) (*Organization, error)
	List(ctx context.Context, f OrganizationFilter) ([]Organization, int64, error)
	// SetStatus changes the publishing decision. Callers must have checked
	// the matching permission; the service validates the transition itself.
	SetStatus(ctx context.Context, id, status, actorID string) (*Organization, error)
	// TransferOwnership moves the owner role to another active member.
	TransferOwnership(ctx context.Context, id, newOwnerID, actorID string) error

	CreateBranch(ctx context.Context, in BranchInput) (*Branch, error)
	UpdateBranch(ctx context.Context, in BranchInput) (*Branch, error)
	Branches(ctx context.Context, organizationID string) ([]Branch, error)
	Branch(ctx context.Context, id string) (*Branch, error)
	DeleteBranch(ctx context.Context, id, actorID string) error

	InviteMember(ctx context.Context, in InviteMemberInput) (*Membership, error)
	AcceptInvitation(ctx context.Context, membershipID, userID string) (*Membership, error)
	UpdateMemberRole(ctx context.Context, membershipID, roleKey, actorID string) (*Membership, error)
	RemoveMember(ctx context.Context, membershipID, actorID string) error
	Members(ctx context.Context, organizationID string) ([]Membership, error)
	// MembershipFor returns the membership linking a user to an organization,
	// or ErrNotFound. It is how a handler resolves the tenant an actor acts in.
	MembershipFor(ctx context.Context, organizationID, userID string) (*Membership, error)
	// AcceptPendingInvitations attaches any invitation sent to a verified
	// e-mail address to the account that just registered.
	AcceptPendingInvitations(ctx context.Context, userID, email string) (int, error)

	Regions(ctx context.Context, parentID string) ([]Region, error)
	CreateRegion(ctx context.Context, in Region) (*Region, error)

	AddServiceArea(ctx context.Context, in ServiceArea) (*ServiceArea, error)
	ServiceAreas(ctx context.Context, organizationID string) ([]ServiceArea, error)
	RemoveServiceArea(ctx context.Context, id, actorID string) error
}
