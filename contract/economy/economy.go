// Package economy is the public contract of the L2 economy module.
//
// Applications import this package (via Deps.Economy) instead of the module's
// internals, for the same reason contract/core exists: a replacement ledger or
// ads engine stays possible without touching application code. The interface is
// deliberately smaller than the module's own service surface.
//
// Money rule: every amount is int64 minor units (for TOKEN, 1 jeton = 1).
// float never crosses this boundary.
package economy

import (
	"context"
	"errors"
	"time"
)

// Sentinel errors. The HTTP layer maps them in one place, like core.Err*.
var (
	// ErrUnknownCurrency means the currency code is not in economy.currencies.
	ErrUnknownCurrency = errors.New("economy: unknown currency")
	// ErrRewardTooEarly means the claim came before required_seconds elapsed
	// on the server clock.
	ErrRewardTooEarly = errors.New("economy: reward claim too early")
	// ErrRewardLimitReached means the daily or monthly earn cap is filled.
	ErrRewardLimitReached = errors.New("economy: reward limit reached")
	// ErrAdNotAvailable means no eligible rewarded ad exists right now.
	ErrAdNotAvailable = errors.New("economy: no rewarded ad available")
	// ErrAdInvalid is a malformed ad (bad media type, non-https affiliate URL).
	ErrAdInvalid = errors.New("economy: ad definition invalid")
	// ErrInvalidAmount means a movement amount is missing or not positive.
	ErrInvalidAmount = errors.New("economy: invalid amount")
	// ErrSessionNotOpen means the view session does not exist, is not owned by
	// the caller, or has already been rewarded.
	ErrSessionNotOpen = errors.New("economy: view session not open")
	// ErrDuplicateEntry means an idempotency key was reused with different
	// inputs or the write raced a concurrent duplicate; retry with a key.
	ErrDuplicateEntry = errors.New("economy: duplicate ledger entry")
)

// Reason codes (L2.md 3.4). The string value is persisted in wallet_entries
// and must never change once a version ships.
const (
	ReasonSignupGrant   = "signup_grant"
	ReasonDailyBonus    = "daily_bonus"
	ReasonAdWatchReward = "ad_watch_reward"
	ReasonAdClickReward = "ad_click_reward"
	ReasonPurchase      = "purchase"
	ReasonRefund        = "refund"
	ReasonAdminCredit   = "admin_credit"
	ReasonAdminDebit    = "admin_debit"
	ReasonSpendFeature  = "spend_feature"
	ReasonExpire        = "expire"
)

// UserEarnRelatedReasons is the reason set the daily/monthly earn caps count.
// Watch and click rewards share one pool (L2.md decision E13).
var UserEarnRelatedReasons = []string{ReasonAdWatchReward, ReasonAdClickReward}

// Standard currencies. The configured list is authoritative; TOKEN is the
// default jeton code for scaffolded apps.
const (
	CurrencyToken = "TOKEN"
	CurrencyTRY   = "TRY"
)

// Entry is one immutable ledger movement.
type Entry struct {
	ID             string    `json:"id"`
	AccountID      string    `json:"account_id"`
	UserID         string    `json:"user_id"`
	Currency       string    `json:"currency"`
	Delta          int64     `json:"delta"`         // never zero
	BalanceAfter   int64     `json:"balance_after"` // never negative
	Reason         string    `json:"reason"`
	ActorID        string    `json:"actor_id,omitempty"` // admin/acting user, if any
	RefType        string    `json:"ref_type,omitempty"`
	RefID          string    `json:"ref_id,omitempty"`
	IdempotencyKey string    `json:"-"` // not revealed in APIs
	CreatedAt      time.Time `json:"created_at"`
}

// Movement is the input to a credit or debit. Amount is always positive; the
// reason determines the direction of the ledger meaning.
type Movement struct {
	UserID         string
	Currency       string
	Amount         int64 // > 0
	Reason         string
	ActorID        string
	RefType        string
	RefID          string
	IdempotencyKey string // optional; makes the movement replay-safe
}

// Balance is one currency's account snapshot.
type Balance struct {
	UserID    string    `json:"user_id"`
	Currency  string    `json:"currency"`
	Balance   int64     `json:"balance"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Wallet is the ledger service. Implementations must be race-safe: the
// balance update is a single conditional UPDATE and the invariant
// SUM(delta) == balance must hold at every instant.
type Wallet interface {
	// EnsureAccount creates the (user, currency) account if missing and
	// applies the signup grant exactly once per chain of account creation.
	// It is therefore the first call a UI makes for a wallet view.
	EnsureAccount(ctx context.Context, userID, currency string) (Balance, error)
	// Credit adds amount to the account and appends one ledger entry.
	Credit(ctx context.Context, in Movement) (Entry, error)
	// Debit subtracts amount; a negative resulting balance is rejected with
	// ErrInsufficientBalance and nothing is written.
	Debit(ctx context.Context, in Movement) (Entry, error)
	// Balance returns the current snapshot; a missing account reads as zero.
	Balance(ctx context.Context, userID, currency string) (Balance, error)
	// Transactions lists the user's own ledger entries, newest first.
	// limit of 0 means "default page size"; the handler applies paging caps.
	Transactions(ctx context.Context, userID, currency string, limit, offset int) ([]Entry, int, error)
}

// Ad is one rewarded-ad creative. Media: "video" (media_url), "image"
// (media_url) or "text" (body_text). target_url may be empty; it must be
// https:// when present (open-redirect defence).
type Ad struct {
	ID           string     `json:"id"`
	Code         string     `json:"code"`
	Title        string     `json:"title"`
	MediaType    string     `json:"media_type"`
	BodyText     string     `json:"body_text,omitempty"`
	MediaURL     string     `json:"media_url,omitempty"`
	Content      string     `json:"content,omitempty"`
	TargetURL    string     `json:"target_url,omitempty"`
	WatchSeconds int        `json:"watch_seconds"`
	WatchReward  int64      `json:"watch_reward"`
	ClickReward  int64      `json:"click_reward"`
	StartsAt     *time.Time `json:"starts_at,omitempty"`
	EndsAt       *time.Time `json:"ends_at,omitempty"`
	Active       bool       `json:"active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// AdViewSession is a server-stamped watch session. required_seconds and
// server_started_at come from the server; the client cannot influence them.
type AdViewSession struct {
	ID              string    `json:"id"`
	AdID            string    `json:"ad_id"`
	UserID          string    `json:"user_id"`
	RequiredSeconds int       `json:"required_seconds"`
	Status          string    `json:"status"` // "started" | "rewarded"
	ServerStartedAt time.Time `json:"server_started_at"`
	ExpiresAt       time.Time `json:"expires_at"`
}

// Session status values persisted in ad_view_sessions.
const (
	SessionStarted  = "started"
	SessionRewarded = "rewarded"
)

// RewardWindow is the user-facing earn budget for one horizon.
type RewardWindow struct {
	Started int64 // earned in this window so far
	Max     int64 // cap for this window
	Elapsed time.Duration
	Renewal time.Time // when the window resets
}

// RewardBudget answers "how much can I still earn?" for the dashboard.
type RewardBudget struct {
	Daily   RewardWindow `json:"daily"`
	Monthly RewardWindow `json:"monthly"`
}

// RewardSettings are the admin-editable economy dials (reward_settings).
// Changing them affects only future rewards; the ledger never changes.
type RewardSettings struct {
	SignupGrant        int64     `json:"signup_grant"`
	AdWatchReward      int64     `json:"ad_watch_reward"`
	DailyRewardCap     int64     `json:"daily_reward_cap"`
	MonthlyRewardCap   int64     `json:"monthly_reward_cap"`
	ClickRewardEnabled bool      `json:"click_reward_enabled"`
	UpdatedAt          time.Time `json:"updated_at"`
	UpdatedBy          string    `json:"updated_by"`
}

// DefaultRewardSettings matches L2.md rev 1.1 defaults:
// 10 welcome tokens, 1 per watched ad, 3/day, 10/month.
func DefaultRewardSettings() RewardSettings {
	return RewardSettings{
		SignupGrant:        10,
		AdWatchReward:      1,
		DailyRewardCap:     3,
		MonthlyRewardCap:   10,
		ClickRewardEnabled: true,
	}
}

// Ads is the rewarded-ads service: serving, watch sessions, clicks, limits
// and the admin CRUD/setting endpoints.
type Ads interface {
	// Pick selects the ad the user can currently earn tokens from: an active
	// ad with a positive reward and unused session for today. ErrAdNotAvailable
	// when the caps are full or nothing qualifies.
	Pick(ctx context.Context, userID, currency string) (Ad, RewardBudget, error)
	// StartView opens a server-stamped watch session. One session per
	// (user, ad, day); a second attempt for the same pair returns the
	// existing session when it is still open.
	StartView(ctx context.Context, userID, adID string) (AdViewSession, error)
	// ClaimView validates elapsed time on the server clock and pays
	// ad_watch_reward exactly once. ErrRewardTooEarly before the duration,
	// ErrRewardLimitReached when the caps intervene.
	ClaimView(ctx context.Context, userID, sessionID string) (Entry, error)
	// RecordClick logs a click (unique per user/ad/day), pays click reward
	// when enabled, and returns the affiliate target for the 302.
	RecordClick(ctx context.Context, userID, adID string) (target string, entry *Entry, err error)
	// Budget returns the user's remaining earn budget for today and month.
	Budget(ctx context.Context, userID, currency string) (RewardBudget, error)
	// Settings returns the current dials.
	Settings(ctx context.Context) (RewardSettings, error)
	// UpdateSettings applies new dials (admin only). updatedBy is recorded
	// in the audit trail on the settings row.
	UpdateSettings(ctx context.Context, in RewardSettings, updatedBy string) (RewardSettings, error)
	// Admin CRUD.
	CreateAd(ctx context.Context, in AdInput) (Ad, error)
	UpdateAd(ctx context.Context, id string, in AdInput) (Ad, error)
	DeleteAd(ctx context.Context, id string) error
	ListAds(ctx context.Context, limit, offset int, activeOnly bool) ([]Ad, int, error)
	AdStats(ctx context.Context, id string) (AdStats, error)
}

// AdInput is the payload admin CRUD accepts; validation lives in the module.
type AdInput struct {
	Code         string
	Title        string
	MediaType    string
	BodyText     string
	MediaURL     string
	Content      string
	TargetURL    string
	WatchSeconds int
	WatchReward  int64
	ClickReward  int64
	StartsAt     *time.Time
	EndsAt       *time.Time
	Active       bool
}

// AdStats is the admin statistics row for one ad.
type AdStats struct {
	AdID         string `json:"ad_id"`
	Sessions     int64  `json:"sessions"`
	Rewarded     int64  `json:"rewarded"`
	RewardedSum  int64  `json:"rewarded_sum"`
	Clicks       int64  `json:"clicks"`
	ClickRewards int64  `json:"click_rewards"`
}
