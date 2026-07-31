package model

import (
"time"

"gorm.io/gorm"
)

// File dosya modeli
type File struct {
ID          uint           json:"id" gorm:"primaryKey"
UserID      uint           json:"user_id" gorm:"not null;index"
FileName    string         json:"file_name" gorm:"not null"
FileSize    int64          json:"file_size" gorm:"not null"
FileType    string         json:"file_type" gorm:"not null" // image, video, document, audio, other
MimeType    string         json:"mime_type" gorm:"not null"
Path        string         json:"path" gorm:"not null"      // Storage path
URL         string         json:"url" gorm:"not null"       // Public URL
StorageType string         json:"storage_type" gorm:"default:local" // local, s3, gcs
IsPublic    bool           json:"is_public" gorm:"default:false"
IsDeleted   bool           json:"is_deleted" gorm:"default:false"
DeletedAt   *time.Time     json:"deleted_at"
CreatedAt   time.Time      json:"created_at"
UpdatedAt   time.Time      json:"updated_at"
Deleted     gorm.DeletedAt json:"-" gorm:"index"
}

func (File) TableName() string {
return "files"
}

// UploadRequest dosya yükleme isteği
type UploadRequest struct {
FileName    string json:"file_name" validate:"required"
FileType    string json:"file_type" validate:"required" // image, video, document, audio, other
IsPublic    bool   json:"is_public"
StorageType string json:"storage_type" // local, s3, gcs
}

// FileResponse dosya yanıtı
type FileResponse struct {
ID          uint      json:"id"
FileName    string    json:"file_name"
FileSize    int64     json:"file_size"
FileType    string    json:"file_type"
MimeType    string    json:"mime_type"
URL         string    json:"url"
StorageType string    json:"storage_type"
IsPublic    bool      json:"is_public"
CreatedAt   time.Time json:"created_at"
}
