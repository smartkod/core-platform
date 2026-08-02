package model

import (
"time"

"gorm.io/gorm"
)

// User kullanÄ±cÄ± modeli (Auth modÃ¼lÃ¼nden geniÅŸletilmiÅŸ)
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

// TableName tablo adÄ±nÄ± dÃ¶ndÃ¼rÃ¼r
func (User) TableName() string {
return "users"
}

// UpdateUserRequest kullanÄ±cÄ± gÃ¼ncelleme isteÄŸi
type UpdateUserRequest struct {
Name     string `json:"name"`
Language string `json:"language"`
Avatar   string `json:"avatar"`
}

// UserResponse kullanÄ±cÄ± yanÄ±tÄ±
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
