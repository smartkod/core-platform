// Package model holds the economy module's GORM row structs.
//
// Column names and constraints live in migrations/0001_economy_init.sql;
// these structs mirror them. Money fields are int64 minor units everywhere.
package model

import "time"

// WalletAccount is one (user, currency) token account.
type WalletAccount struct {
	ID           string    `gorm:"primaryKey;type:uuid"`
	UserID       string    `gorm:"type:uuid;index;not null"`
	Currency     string    `gorm:"size:8;not null"`
	BalanceMinor int64     `gorm:"not null;default:0"`
	CreatedAt    time.Time `gorm:"not null"`
	UpdatedAt    time.Time `gorm:"not null"`
}

// WalletEntry is one append-only ledger movement. Updates and deletes are
// blocked by the wallet_entries_no_rewrite trigger.
type WalletEntry struct {
	ID             string    `gorm:"primaryKey;type:uuid"`
	AccountID      string    `gorm:"type:uuid;not null"`
	DeltaMinor     int64     `gorm:"column:delta_minor;not null"`
	BalanceAfter   int64     `gorm:"column:balance_after_minor;not null"`
	ReasonCode     string    `gorm:"size:40;not null"`
	ActorID        *string   `gorm:"type:uuid"`
	RefType        *string   `gorm:"size:40"`
	RefID          *string   `gorm:"type:uuid"`
	IdempotencyKey *string   `gorm:"size:120"`
	CreatedAt      time.Time `gorm:"not null;index"`
}

// Product is one sellable catalog item.
type Product struct {
	ID          string    `gorm:"primaryKey;type:uuid"`
	Code        string    `gorm:"size:60;not null;uniqueIndex"`
	Name        string    `gorm:"size:160;not null"`
	Kind        string    `gorm:"size:16;not null"`
	Currency    string    `gorm:"size:8;not null"`
	AmountMinor int64     `gorm:"not null"`
	Interval    *string   `gorm:"size:12"`
	Meta        []byte    `gorm:"type:jsonb;not null;default:'{}'::jsonb"`
	Active      bool      `gorm:"not null;default:true"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

// Ad is one rewarded-ad creative.
type Ad struct {
	ID           string     `gorm:"primaryKey;type:uuid"`
	Code         string     `gorm:"size:60;not null;uniqueIndex"`
	Title        string     `gorm:"size:160;not null"`
	MediaType    string     `gorm:"size:8;not null"`
	BodyText     *string    `gorm:"type:text"`
	MediaURL     *string    `gorm:"type:text"`
	Content      *string    `gorm:"type:text"`
	TargetURL    *string    `gorm:"type:text"`
	WatchSeconds int        `gorm:"not null"`
	WatchReward  int64      `gorm:"not null"`
	ClickReward  int64      `gorm:"not null"`
	StartsAt     *time.Time `gorm:"timestamptz"`
	EndsAt       *time.Time `gorm:"timestamptz"`
	Active       bool       `gorm:"not null;default:true"`
	CreatedAt    time.Time  `gorm:"not null"`
	UpdatedAt    time.Time  `gorm:"not null"`
}

// AdClick records one affiliate click. The reward uniqueness is enforced by
// the partial unique index; anonymous clicks are statistics only.
type AdClick struct {
	ID         string     `gorm:"primaryKey;type:uuid"`
	AdID       string     `gorm:"type:uuid;not null"`
	UserID     *string    `gorm:"type:uuid"`
	Kind       string     `gorm:"size:8;not null"`
	RewardDate *time.Time `gorm:"type:date"`
	IPHash     *string    `gorm:"size:64"`
	CreatedAt  time.Time  `gorm:"not null"`
}

// AdViewSession is a server-stamped watch session.
type AdViewSession struct {
	ID              string    `gorm:"primaryKey;type:uuid"`
	AdID            string    `gorm:"type:uuid;not null"`
	UserID          string    `gorm:"type:uuid;not null"`
	RequiredSeconds int       `gorm:"not null"`
	Status          string    `gorm:"size:10;not null"`
	ServerStartedAt time.Time `gorm:"timestamptz;not null"`
	ExpiresAt       time.Time `gorm:"timestamptz;not null"`
	RewardMinor     int64     `gorm:"not null;default:0"`
	RewardDate      time.Time `gorm:"type:date;not null"`
	EntryID         *string   `gorm:"type:uuid"`
	CreatedAt       time.Time `gorm:"not null"`
}

// RewardSettings is the singleton row of admin-editable dials.
type RewardSettings struct {
	ID                 int       `gorm:"primaryKey;type:int;not null;default:1"`
	SignupGrant        int64     `gorm:"not null;default:10"`
	AdWatchReward      int64     `gorm:"not null;default:1"`
	DailyRewardCap     int64     `gorm:"not null;default:3"`
	MonthlyRewardCap   int64     `gorm:"not null;default:10"`
	ClickRewardEnabled bool      `gorm:"not null;default:true"`
	UpdatedAt          time.Time `gorm:"not null"`
	UpdatedBy          *string   `gorm:"type:uuid"`
}
