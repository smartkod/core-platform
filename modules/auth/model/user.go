package model

import (
"time"

"gorm.io/gorm"
)

// User kullanıcı modeli
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

// LoginRequest login isteği
type LoginRequest struct {
Email    string `json:"email" validate:"required,email"`
Password string `json:"password" validate:"required,min=6"`
}

// RegisterRequest kayıt isteği
type RegisterRequest struct {
Email    string `json:"email" validate:"required,email"`
Password string `json:"password" validate:"required,min=6"`
Name     string `json:"name" validate:"required"`
}

// LoginResponse login yanıtı
type LoginResponse struct {
AccessToken  string `json:"access_token"`
RefreshToken string `json:"refresh_token"`
TokenType    string `json:"token_type"`
ExpiresIn    int64  `json:"expires_in"`
}

// UserResponse kullanıcı yanıtı
type UserResponse struct {
ID       uint   `json:"id"`
Email    string `json:"email"`
Name     string `json:"name"`
Language string `json:"language"`
Avatar   string `json:"avatar"`
IsActive bool   `json:"is_active"`
}
