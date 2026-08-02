package model

import (
"time"

"gorm.io/gorm"
)

type File struct {
ID           uint           `json:"id" gorm:"primaryKey"`
UserID       uint           `json:"user_id" gorm:"not null;index"`
FileName     string         `json:"file_name" gorm:"not null"`          // GÃƒÂ¼venli isim
OriginalName string         `json:"original_name" gorm:"not null"`      // KullanÃ„Â±cÃ„Â±nÃ„Â±n verdiÃ„Å¸i isim
FileSize     int64          `json:"file_size" gorm:"not null"`
FileType     string         `json:"file_type" gorm:"not null"`
MimeType     string         `json:"mime_type" gorm:"not null"`
Path         string         `json:"path" gorm:"not null"`
URL          string         `json:"url" gorm:"not null"`
StorageType  string         `json:"storage_type" gorm:"default:local"`
IsPublic     bool           `json:"is_public" gorm:"default:false"`
IsDeleted    bool           `json:"is_deleted" gorm:"default:false"`
DeletedAt    *time.Time     `json:"deleted_at"`
CreatedAt    time.Time      `json:"created_at"`
UpdatedAt    time.Time      `json:"updated_at"`
Deleted      gorm.DeletedAt `json:"-" gorm:"index"`
}

func (File) TableName() string {
return "files"
}

type FileResponse struct {
ID           uint      `json:"id"`
FileName     string    `json:"file_name"`
OriginalName string    `json:"original_name"`
FileSize     int64     `json:"file_size"`
FileType     string    `json:"file_type"`
MimeType     string    `json:"mime_type"`
URL          string    `json:"url"`
StorageType  string    `json:"storage_type"`
IsPublic     bool      `json:"is_public"`
CreatedAt    time.Time `json:"created_at"`
}