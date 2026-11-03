// Package model holds the organization persistence types.
package model

import "time"

// Organization is a business or institution on the platform.
type Organization struct {
	ID                string    `gorm:"column:id;primaryKey"`
	Slug              string    `gorm:"column:slug"`
	Name              string    `gorm:"column:name"`
	Description       string    `gorm:"column:description"`
	LogoURL           string    `gorm:"column:logo_url"`
	Email             string    `gorm:"column:email"`
	Phone             string    `gorm:"column:phone"`
	Website           string    `gorm:"column:website"`
	OwnerID           string    `gorm:"column:owner_id"`
	Status            string    `gorm:"column:status"`
	VerificationLevel string    `gorm:"column:verification_level"`
	Locale            string    `gorm:"column:locale"`
	CreatedAt         time.Time `gorm:"column:created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at"`
}

// TableName pins the table name.
func (Organization) TableName() string { return "organizations" }

// Branch is one location or channel.
type Branch struct {
	ID             string    `gorm:"column:id;primaryKey"`
	OrganizationID string    `gorm:"column:organization_id"`
	Name           string    `gorm:"column:name"`
	Type           string    `gorm:"column:type"`
	AddressLine    string    `gorm:"column:address_line"`
	Country        string    `gorm:"column:country"`
	City           string    `gorm:"column:city"`
	District       string    `gorm:"column:district"`
	PostalCode     string    `gorm:"column:postal_code"`
	Latitude       *float64  `gorm:"column:latitude"`
	Longitude      *float64  `gorm:"column:longitude"`
	Phone          string    `gorm:"column:phone"`
	Public         bool      `gorm:"column:public"`
	Status         string    `gorm:"column:status"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

// TableName pins the table name.
func (Branch) TableName() string { return "branches" }

// Membership links a person to an organization.
type Membership struct {
	ID             string     `gorm:"column:id;primaryKey"`
	OrganizationID string     `gorm:"column:organization_id"`
	BranchID       *string    `gorm:"column:branch_id"`
	UserID         *string    `gorm:"column:user_id"`
	InvitedEmail   string     `gorm:"column:invited_email"`
	RoleKey        string     `gorm:"column:role_key"`
	Status         string     `gorm:"column:status"`
	InvitedBy      *string    `gorm:"column:invited_by"`
	InvitedAt      time.Time  `gorm:"column:invited_at"`
	AcceptedAt     *time.Time `gorm:"column:accepted_at"`
	RemovedAt      *time.Time `gorm:"column:removed_at"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at"`
}

// TableName pins the table name.
func (Membership) TableName() string { return "memberships" }

// Region is a node of the geographic reference tree.
type Region struct {
	ID       string  `gorm:"column:id;primaryKey"`
	ParentID *string `gorm:"column:parent_id"`
	Level    string  `gorm:"column:level"`
	Code     string  `gorm:"column:code"`
	Name     string  `gorm:"column:name"`
	Country  string  `gorm:"column:country"`
	Locale   string  `gorm:"column:locale"`
}

// TableName pins the table name.
func (Region) TableName() string { return "regions" }

// ServiceArea is where an organization serves customers.
type ServiceArea struct {
	ID             string    `gorm:"column:id;primaryKey"`
	OrganizationID string    `gorm:"column:organization_id"`
	BranchID       *string   `gorm:"column:branch_id"`
	RegionID       *string   `gorm:"column:region_id"`
	Mode           string    `gorm:"column:mode"`
	CreatedAt      time.Time `gorm:"column:created_at"`
}

// TableName pins the table name.
func (ServiceArea) TableName() string { return "service_areas" }

// StatusEvent records one publishing decision.
type StatusEvent struct {
	ID             string    `gorm:"column:id;primaryKey"`
	OrganizationID string    `gorm:"column:organization_id"`
	FromStatus     string    `gorm:"column:from_status"`
	ToStatus       string    `gorm:"column:to_status"`
	ActorID        *string   `gorm:"column:actor_id"`
	Reason         string    `gorm:"column:reason"`
	CreatedAt      time.Time `gorm:"column:created_at"`
}

// TableName pins the table name.
func (StatusEvent) TableName() string { return "organization_status_events" }
