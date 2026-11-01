// Package model holds the files persistence type.
package model

import "time"

// File is one stored object's metadata.
type File struct {
	ID           string     `gorm:"column:id;primaryKey"`
	OwnerID      string     `gorm:"column:owner_id"`
	ObjectKey    string     `gorm:"column:object_key"`
	OriginalName string     `gorm:"column:original_name"`
	ContentType  string     `gorm:"column:content_type"`
	SizeBytes    int64      `gorm:"column:size_bytes"`
	Checksum     string     `gorm:"column:checksum"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	DeletedAt    *time.Time `gorm:"column:deleted_at"`
}

// TableName pins the table so a struct rename cannot silently change the schema.
func (File) TableName() string { return "files" }
