package model

import (
"time"

"gorm.io/gorm"
)

// User kullanÃ„Â±cÃ„Â± modeli
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

// TableName tablo adÃ„Â±nÃ„Â± dÃƒÂ¶ndÃƒÂ¼rÃƒÂ¼r
func (User) TableName() string {
return "users"
}

// LoginRequest login isteÃ„Å¸i
type LoginRequest struct {
Email    string `json:"email" validate:"required,email"`
Password string `json:"password" validate:"required,min=6"`
}

// RegisterRequest kayÃ„Â±t isteÃ„Å¸i
type RegisterRequest struct {
Email    string `json:"email" validate:"required,email"`
Password string `json:"password" validate:"required,min=6"`
Name     string `json:"name" validate:"required"`
}

// LoginResponse login yanÃ„Â±tÃ„Â±
type LoginResponse struct {
AccessToken  string `json:"access_token"`
RefreshToken string `json:"refresh_token"`
TokenType    string `json:"token_type"`
ExpiresIn    int64  `json:"expires_in"`
}

// UserResponse kullanÃ„Â±cÃ„Â± yanÃ„Â±tÃ„Â±
type UserResponse struct {
ID       uint   `json:"id"`
Email    string `json:"email"`
Name     string `json:"name"`
Language string `json:"language"`
Avatar   string `json:"avatar"`
IsActive bool   `json:"is_active"`
}
