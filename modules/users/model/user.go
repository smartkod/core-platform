package model

import (
"time"

"gorm.io/gorm"
)

// User kullanıcı modeli (Auth modülünden genişletilmiş)
type User struct {
ID        uint           `json:"id" gorm:"primaryKey"`
Email     string         `json:"email" gorm:"uniqueIndex;not null"`
Password  string         `json:"-" gorm:"not null"`
Name      string         `json:"name"`
Language  string         `json:"language" gorm:"default:'tr'"`
Avatar    string         `json:"avatar"`
IsActive  bool           `json:"is_active" gorm:"default:true"`
LastLogin *time.Time     `json:"last_login"`
CreatedAt time.Time      `json:"created_at"`
UpdatedAt time.Time      `json:"updated_at"`
DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName tablo adını döndürür
func (User) TableName() string {
return "users"
}

// UpdateUserRequest kullanıcı güncelleme isteği
type UpdateUserRequest struct {
Name     string `json:"name"`
Language string `json:"language"`
Avatar   string `json:"avatar"`
}

// UserResponse kullanıcı yanıtı
type UserResponse struct {
ID        uint      `json:"id"`
Email     string    `json:"email"`
Name      string    `json:"name"`
Language  string    `json:"language"`
Avatar    string    `json:"avatar"`
IsActive  bool      `json:"is_active"`
LastLogin *time.Time `json:"last_login,omitempty"`
CreatedAt time.Time `json:"created_at"`
UpdatedAt time.Time `json:"updated_at"`
}
