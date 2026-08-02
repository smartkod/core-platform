package model

import (
"time"

"gorm.io/gorm"
)

// Wallet kullanÄ±cÄ± cÃ¼zdanÄ±
type Wallet struct {
ID          uint           json:"id" gorm:"primaryKey"
UserID      uint           json:"user_id" gorm:"uniqueIndex;not null"
Balance     int            json:"balance" gorm:"default:0" // Jeton miktarÄ±
TotalEarned int            json:"total_earned" gorm:"default:0"
TotalSpent  int            json:"total_spent" gorm:"default:0"
CreatedAt   time.Time      json:"created_at"
UpdatedAt   time.Time      json:"updated_at"
DeletedAt   gorm.DeletedAt json:"-" gorm:"index"
}

func (Wallet) TableName() string {
return "wallets"
}

// WalletTransaction cÃ¼zdan iÅŸlemi
type WalletTransaction struct {
ID          uint           json:"id" gorm:"primaryKey"
UserID      uint           json:"user_id" gorm:"not null;index"
Type        string         json:"type" gorm:"not null" // earn, spend, bonus, reward, purchase
Amount      int            json:"amount" gorm:"not null"
Description string         json:"description"
Reference   string         json:"reference" // Ä°lgili iÅŸlem ID'si
Balance     int            json:"balance"   // Ä°ÅŸlem sonrasÄ± bakiye
CreatedAt   time.Time      json:"created_at"
DeletedAt   gorm.DeletedAt json:"-" gorm:"index"
}

func (WalletTransaction) TableName() string {
return "wallet_transactions"
}

// WalletResponse cÃ¼zdan yanÄ±tÄ±
type WalletResponse struct {
Balance     int json:"balance"
TotalEarned int json:"total_earned"
TotalSpent  int json:"total_spent"
}

// EarnRequest jeton kazanma isteÄŸi
type EarnRequest struct {
Amount      int    json:"amount" validate:"required,min=1"
Type        string json:"type" validate:"required" // daily_bonus, ad_reward, task, referral
Description string json:"description"
}

// SpendRequest jeton harcama isteÄŸi
type SpendRequest struct {
Amount      int    json:"amount" validate:"required,min=1"
Type        string json:"type" validate:"required" // ai_reading, premium_report, gift
Description string json:"description"
}

// TransactionResponse iÅŸlem yanÄ±tÄ±
type TransactionResponse struct {
ID          uint      json:"id"
Type        string    json:"type"
Amount      int       json:"amount"
Description string    json:"description"
Balance     int       json:"balance"
CreatedAt   time.Time json:"created_at"
}
