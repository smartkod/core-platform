// Package model holds the identity persistence types.
//
// These are the only structs mapped to tables; HTTP DTOs live in api so a
// schema change never leaks into the wire format.
package model

import "time"

// User is a platform account. One user exists per application database; there
// is no cross-application identity (PRD.md §5).
type User struct {
	ID              string     `gorm:"column:id;primaryKey"`
	Email           string     `gorm:"column:email"`
	Phone           string     `gorm:"column:phone"`
	PasswordHash    string     `gorm:"column:password_hash"`
	Name            string     `gorm:"column:name"`
	Locale          string     `gorm:"column:locale"`
	AvatarURL       string     `gorm:"column:avatar_url"`
	Role            string     `gorm:"column:role"`
	Status          string     `gorm:"column:status"`
	EmailVerifiedAt *time.Time `gorm:"column:email_verified_at"`
	PhoneVerifiedAt *time.Time `gorm:"column:phone_verified_at"`
	ClosedAt        *time.Time `gorm:"column:closed_at"`
	LastLoginAt     *time.Time `gorm:"column:last_login_at"`
	TokenVersion    int        `gorm:"column:token_version"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`
	DeletedAt       *time.Time `gorm:"column:deleted_at"`
}

// TableName pins the table so a struct rename cannot silently change the schema.
func (User) TableName() string { return "users" }

// Role and status values.
const (
	RoleUser      = "user"
	RoleAdmin     = "admin"
	RoleModerator = "moderator"

	StatusActive    = "active"
	StatusSuspended = "suspended"
	StatusClosed    = "closed"
	StatusDeleted   = "deleted"
)

// RefreshToken is one issued refresh credential. Only the hash is stored.
type RefreshToken struct {
	ID         string     `gorm:"column:id;primaryKey"`
	UserID     string     `gorm:"column:user_id"`
	TokenHash  string     `gorm:"column:token_hash"`
	FamilyID   string     `gorm:"column:family_id"`
	Device     string     `gorm:"column:device"`
	IP         string     `gorm:"column:ip"`
	Audience   string     `gorm:"column:audience"`
	ExpiresAt  time.Time  `gorm:"column:expires_at"`
	RevokedAt  *time.Time `gorm:"column:revoked_at"`
	ReplacedBy *string    `gorm:"column:replaced_by"`
	CreatedAt  time.Time  `gorm:"column:created_at"`
}

// TableName pins the table name.
func (RefreshToken) TableName() string { return "refresh_tokens" }

// OneTimeToken backs e-mail verification and password reset.
type OneTimeToken struct {
	ID        string     `gorm:"column:id;primaryKey"`
	UserID    string     `gorm:"column:user_id"`
	Purpose   string     `gorm:"column:purpose"`
	TokenHash string     `gorm:"column:token_hash"`
	ExpiresAt time.Time  `gorm:"column:expires_at"`
	UsedAt    *time.Time `gorm:"column:used_at"`
	CreatedAt time.Time  `gorm:"column:created_at"`
}

// TableName pins the table name.
func (OneTimeToken) TableName() string { return "one_time_tokens" }

// One-time token purposes.
const (
	PurposeVerifyEmail   = "verify_email"
	PurposeResetPassword = "reset_password"
)
