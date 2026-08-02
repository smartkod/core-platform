package model

import (
"time"

"gorm.io/gorm"
)

// PremiumPlan premium plan modeli
type PremiumPlan struct {
ID          uint           json:"id" gorm:"primaryKey"
Name        string         json:"name" gorm:"not null"
Description string         json:"description"
Price       float64        json:"price" gorm:"not null"
Currency    string         json:"currency" gorm:"default:TRY"
Duration    int            json:"duration" gorm:"not null" // GÃ¼n cinsinden
Features    string         json:"features" gorm:"type:jsonb"
IsActive    bool           json:"is_active" gorm:"default:true"
CreatedAt   time.Time      json:"created_at"
UpdatedAt   time.Time      json:"updated_at"
DeletedAt   gorm.DeletedAt json:"-" gorm:"index"
}

func (PremiumPlan) TableName() string {
return "premium_plans"
}

// UserPremium kullanÄ±cÄ± premium kaydÄ±
type UserPremium struct {
ID          uint           json:"id" gorm:"primaryKey"
UserID      uint           json:"user_id" gorm:"uniqueIndex;not null"
PlanID      uint           json:"plan_id" gorm:"not null"
StartDate   time.Time      json:"start_date" gorm:"not null"
EndDate     *time.Time     json:"end_date"
IsActive    bool           json:"is_active" gorm:"default:true"
AutoRenew   bool           json:"auto_renew" gorm:"default:false"
CanceledAt  *time.Time     json:"canceled_at"
CreatedAt   time.Time      json:"created_at"
UpdatedAt   time.Time      json:"updated_at"
DeletedAt   gorm.DeletedAt json:"-" gorm:"index"
}

func (UserPremium) TableName() string {
return "user_premiums"
}

// PremiumPurchaseRequest premium satÄ±n alma isteÄŸi
type PremiumPurchaseRequest struct {
PlanID    uint json:"plan_id" validate:"required"
AutoRenew bool json:"auto_renew"
}

// PremiumResponse premium yanÄ±tÄ±
type PremiumResponse struct {
IsActive       bool       json:"is_active"
PlanName       string     json:"plan_name,omitempty"
PlanID         uint       json:"plan_id,omitempty"
StartDate      *time.Time json:"start_date,omitempty"
EndDate        *time.Time json:"end_date,omitempty"
DaysRemaining  int        json:"days_remaining,omitempty"
IsLifetime     bool       json:"is_lifetime"
AutoRenew      bool       json:"auto_renew"
}
