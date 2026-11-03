// Package model holds the authorization persistence types.
package model

import "time"

// Role is a named bundle of permissions.
type Role struct {
	Key         string    `gorm:"column:key;primaryKey"`
	Name        string    `gorm:"column:name"`
	Description string    `gorm:"column:description"`
	System      bool      `gorm:"column:system"`
	CreatedAt   time.Time `gorm:"column:created_at"`
}

// TableName pins the table name.
func (Role) TableName() string { return "roles" }

// Permission is one named capability.
type Permission struct {
	Key         string    `gorm:"column:key;primaryKey"`
	Description string    `gorm:"column:description"`
	Scope       string    `gorm:"column:scope"`
	CreatedAt   time.Time `gorm:"column:created_at"`
}

// TableName pins the table name.
func (Permission) TableName() string { return "permissions" }

// RolePermission links a role to a permission.
type RolePermission struct {
	RoleKey       string `gorm:"column:role_key;primaryKey"`
	PermissionKey string `gorm:"column:permission_key;primaryKey"`
}

// TableName pins the table name.
func (RolePermission) TableName() string { return "role_permissions" }

// Assignment grants one role in one scope to one user.
type Assignment struct {
	ID        string     `gorm:"column:id;primaryKey"`
	UserID    string     `gorm:"column:user_id"`
	RoleKey   string     `gorm:"column:role_key"`
	Scope     string     `gorm:"column:scope"`
	ScopeID   string     `gorm:"column:scope_id"`
	GrantedBy *string    `gorm:"column:granted_by"`
	ExpiresAt *time.Time `gorm:"column:expires_at"`
	RevokedAt *time.Time `gorm:"column:revoked_at"`
	CreatedAt time.Time  `gorm:"column:created_at"`
}

// TableName pins the table name.
func (Assignment) TableName() string { return "role_assignments" }
