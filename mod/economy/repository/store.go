// Package repository is the economy module's data access layer.
//
// Every money mutation goes through the conditional UPDATE in ApplyDelta:
// read-then-write patterns are deliberately absent (L2.md 3.3). All methods
// take a *gorm.DB so the service layer can run multi-statement moves (ledger
// entry + state change) inside one transaction by passing a tx handle, and
// standalone queries by passing the pool.
package repository

import (
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"core-platform/mod/economy/model"
)

// Store bundles the DB handle used by every economy query.
type Store struct {
	DB *gorm.DB
}

// NewStore builds the repository over a database or transaction handle.
func NewStore(db *gorm.DB) *Store { return &Store{DB: db} }

// ErrInsufficient mirrors contract/economy.ErrInsufficientBalance without
// importing the contract here; the service maps it.
type insufficient struct{}

func (insufficient) Error() string { return "insufficient balance" }

// ErrReasonInsufficientBalance is returned by ApplyDelta when the balance
// would go negative; the caller must roll back its transaction.
var ErrReasonInsufficientBalance = insufficient{}

// ErrSessionMissing is returned when a view session cannot be located or
// is not in a claimable state for the caller.
var ErrSessionMissing = fmt.Errorf("view session missing")

// today returns the date bucket for daily caps and unique constraints, in UTC.
// UTC rather than server-local time keeps multi-instance deployments and the
// Linux CI deterministic (L2.md open question A6).
func today(now time.Time) time.Time {
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// firstOfMonth returns the calendar-month boundary for monthly caps, in UTC.
func firstOfMonth(now time.Time) time.Time {
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// ------------------------------------------------------------------
// Wallet primitives
// ------------------------------------------------------------------

// EnsureAccount returns the (user, currency) account, creating it when both
// are missing. created reports whether this call inserted the row: that flag
// gates the one-time signup grant.
func (s *Store) EnsureAccount(tx *gorm.DB, userID, currency string, now time.Time) (model.WalletAccount, bool, error) {
	var existing model.WalletAccount
	err := tx.Where("user_id = ? AND currency = ?", userID, currency).First(&existing).Error
	if err == nil {
		return existing, false, nil
	}
	if err != gorm.ErrRecordNotFound {
		return model.WalletAccount{}, false, err
	}

	acc := model.WalletAccount{ID: uuid.NewString(), UserID: userID, Currency: currency}
	res := tx.Raw(
		`INSERT INTO wallet_accounts (id, user_id, currency, balance_minor, created_at, updated_at)
		 VALUES (?, ?, ?, 0, ?, ?) ON CONFLICT (user_id, currency) DO NOTHING
		 RETURNING id, user_id, currency, balance_minor, created_at, updated_at`,
		acc.ID, userID, currency, now, now,
	).Scan(&acc)
	if res.Error != nil {
		return model.WalletAccount{}, false, res.Error
	}
	if res.RowsAffected == 0 {
		// Lost a create race; the winner is the account we return.
		err := tx.Where("user_id = ? AND currency = ?", userID, currency).First(&existing).Error
		return existing, false, err
	}
	return acc, true, nil
}

// ApplyDelta moves the balance by delta in one conditional UPDATE and returns
// the resulting balance. No rows means the update would have gone negative:
// the caller rolls back and reports insufficient balance.
func (s *Store) ApplyDelta(tx *gorm.DB, accountID string, delta int64) (int64, error) {
	var after int64
	res := tx.Raw(
		`UPDATE wallet_accounts
		    SET balance_minor = balance_minor + ?, updated_at = now()
		  WHERE id = ? AND balance_minor + ? >= 0
		  RETURNING balance_minor`,
		delta, accountID, delta,
	).Scan(&after)
	if res.Error != nil {
		return 0, res.Error
	}
	if res.RowsAffected == 0 {
		return 0, ErrReasonInsufficientBalance
	}
	return after, nil
}

// InsertEntry appends one immutable ledger row.
func (s *Store) InsertEntry(tx *gorm.DB, e model.WalletEntry) error {
	return tx.Create(&e).Error
}

// EntryByIDempotency finds the entry a replayed movement must return.
func (s *Store) EntryByIDempotency(tx *gorm.DB, accountID, key string) (model.WalletEntry, bool, error) {
	if key == "" || !utf8.ValidString(key) {
		return model.WalletEntry{}, false, nil
	}
	var e model.WalletEntry
	err := tx.Where("account_id = ? AND idempotency_key = ?", accountID, key).First(&e).Error
	if err == gorm.ErrRecordNotFound {
		return model.WalletEntry{}, false, nil
	}
	if err != nil {
		return model.WalletEntry{}, false, err
	}
	return e, true, nil
}

// EarnedSince sums movement deltas for the user's earn reasons from since.
// Pass the day boundary for daily caps, the month boundary for monthly caps.
func (s *Store) EarnedSince(tx *gorm.DB, userID, currency string, reasons []string, since time.Time) (int64, error) {
	if len(reasons) == 0 {
		return 0, nil
	}
	var sum int64
	err := tx.Raw(
		`SELECT COALESCE(SUM(e.delta_minor), 0)
		   FROM wallet_entries e
		   JOIN wallet_accounts a ON a.id = e.account_id
		  WHERE a.user_id = ? AND a.currency = ?
		    AND e.reason_code IN ? AND e.created_at >= ?`,
		userID, currency, reasons, since,
	).Scan(&sum).Error
	return sum, err
}

// AccountExists reports whether the account row exists; a missing account
// reads as a zero balance for the cap checks.
func (s *Store) AccountExists(tx *gorm.DB, userID, currency string) (bool, error) {
	var n int64
	err := tx.Model(&model.WalletAccount{}).
		Where("user_id = ? AND currency = ?", userID, currency).
		Count(&n).Error
	return n > 0, err
}

// NewEntry builds a ledger row for a movement.
func NewEntry(accountID string, delta int64, balanceAfter int64, reason, actorID, refType, refID, idemKey string, now time.Time) model.WalletEntry {
	e := model.WalletEntry{
		ID:           uuid.NewString(),
		AccountID:    accountID,
		DeltaMinor:   delta,
		BalanceAfter: balanceAfter,
		ReasonCode:   reason,
		CreatedAt:    now,
	}
	if actorID != "" {
		a := actorID
		e.ActorID = &a
	}
	if refType != "" {
		t := refType
		e.RefType = &t
	}
	if refID != "" {
		r := refID
		e.RefID = &r
	}
	if idemKey != "" {
		k := idemKey
		e.IdempotencyKey = &k
	}
	return e
}

// ------------------------------------------------------------------
// Ads primitives
// ------------------------------------------------------------------

// EligibleAd is one ad row the pick query can serve.
type EligibleAd = model.Ad

// PickEligibleAd selects an active, currently-in-window ad with a positive
// watch reward that still has an unused session for the user today.
// Random order gives light rotation without a dedicated scheduler.
func (s *Store) PickEligibleAd(tx *gorm.DB, userID string, day time.Time, now time.Time) (model.Ad, bool, error) {
	var ad model.Ad
	res := tx.Raw(
		`SELECT * FROM ads a
		  WHERE a.active AND a.watch_reward > 0
		    AND (a.starts_at IS NULL OR a.starts_at <= ?)
		    AND (a.ends_at IS NULL OR a.ends_at >= ?)
		    AND NOT EXISTS (
		        SELECT 1 FROM ad_view_sessions v
		         WHERE v.ad_id = a.id AND v.user_id = ? AND v.reward_date = ?
		    )
		  ORDER BY random()
		  LIMIT 1`,
		now, now, userID, day,
	).Scan(&ad)
	if res.Error != nil {
		return model.Ad{}, false, res.Error
	}
	if res.RowsAffected == 0 {
		return model.Ad{}, false, nil
	}
	return ad, true, nil
}

// AdByID loads one ad.
func (s *Store) AdByID(tx *gorm.DB, id string) (model.Ad, bool, error) {
	var ad model.Ad
	err := tx.Where("id = ?", id).First(&ad).Error
	if err == gorm.ErrRecordNotFound {
		return model.Ad{}, false, nil
	}
	return ad, err == nil, err
}

// OpenSessionToday returns the user's session for one ad and day, if any.
// Both started and already-rewarded sessions prevent duplicates.
func (s *Store) OpenSessionToday(tx *gorm.DB, userID, adID string, day time.Time) (model.AdViewSession, bool, error) {
	var v model.AdViewSession
	err := tx.Where("ad_id = ? AND user_id = ? AND reward_date = ?", adID, userID, day).First(&v).Error
	if err == gorm.ErrRecordNotFound {
		return model.AdViewSession{}, false, nil
	}
	if err != nil {
		return model.AdViewSession{}, false, err
	}
	return v, true, nil
}

// SessionForUpdate locks one view session row for the claim transaction,
// making the started -> rewarded flip the only possible transition even under
// concurrent duplicate claims.
func (s *Store) SessionForUpdate(tx *gorm.DB, id string) (model.AdViewSession, bool, error) {
	var v model.AdViewSession
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).First(&v).Error
	if err == gorm.ErrRecordNotFound {
		return model.AdViewSession{}, false, nil
	}
	if err != nil {
		return model.AdViewSession{}, false, err
	}
	return v, true, nil
}

// CreateSession inserts a new watch session. A unique violation means the
// create raced another request for the same (user, ad, day); the caller
// reports the race by re-reading.
func (s *Store) CreateSession(tx *gorm.DB, v model.AdViewSession) error {
	return tx.Create(&v).Error
}

// RewardSession flips a session to rewarded and links the ledger entry.
// The row-level lock from the caller (SELECT ... FOR UPDATE) makes the
// status flip the only rewarded transition.
func (s *Store) RewardSession(tx *gorm.DB, sessionID, entryID string, reward int64) error {
	updates := map[string]any{
		"status":       "rewarded",
		"reward_minor": reward,
	}
	if entryID != "" {
		updates["entry_id"] = entryID
	}
	res := tx.Model(&model.AdViewSession{}).
		Where("id = ? AND status = ?", sessionID, "started").
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrSessionMissing
	}
	return nil
}

// InsertClickUnlessDup inserts one click and reports whether it is the user's
// first rewarded click for (ad, day) — the reward key. Anonymous visits never
// conflict.
func (s *Store) InsertClickUnlessDup(tx *gorm.DB, c model.AdClick) (bool, error) {
	res := tx.Exec(
		`INSERT INTO ads_clicks (id, ad_id, user_id, kind, reward_date, ip_hash, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (user_id, ad_id, reward_date)
		 WHERE kind = 'click' AND user_id IS NOT NULL AND reward_date IS NOT NULL
		 DO NOTHING`,
		c.ID, c.AdID, c.UserID, c.Kind, c.RewardDate, c.IPHash, c.CreatedAt,
	)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// Settings loads the singleton dial row, creating it with defaults when the
// database has never been seeded. Reads never rewrite the row: updated_at and
// updated_by belong to the last admin save, not to the last read.
func (s *Store) Settings(tx *gorm.DB, now time.Time) (model.RewardSettings, error) {
	var r model.RewardSettings
	err := tx.Raw(
		`INSERT INTO reward_settings (id, updated_at, updated_by)
		 VALUES (1, ?, NULL) ON CONFLICT (id) DO NOTHING
		 RETURNING id, signup_grant, ad_watch_reward, daily_reward_cap,
		           monthly_reward_cap, click_reward_enabled, updated_at, updated_by`,
		now,
	).Scan(&r).Error
	if err != nil {
		return model.RewardSettings{}, err
	}
	if r.ID != 0 {
		return r, nil
	}
	err = tx.Where("id = ?", 1).First(&r).Error
	return r, err
}

// SaveSettings rewrites the singleton dial row. updated_by is the acting
// admin and lands in the audit trail on the row itself.
func (s *Store) SaveSettings(tx *gorm.DB, r model.RewardSettings, now time.Time) error {
	return tx.Model(&model.RewardSettings{}).Where("id = ?", 1).Updates(map[string]any{
		"signup_grant":         r.SignupGrant,
		"ad_watch_reward":      r.AdWatchReward,
		"daily_reward_cap":     r.DailyRewardCap,
		"monthly_reward_cap":   r.MonthlyRewardCap,
		"click_reward_enabled": r.ClickRewardEnabled,
		"updated_at":           now,
		"updated_by":           r.UpdatedBy,
	}).Error
}

// InsertAd / UpdateAd / DeleteAd are the admin CRUD edge.
func (s *Store) InsertAd(tx *gorm.DB, a model.Ad) error { return tx.Create(&a).Error }

func (s *Store) UpdateAd(tx *gorm.DB, a model.Ad) error {
	return tx.Model(&model.Ad{}).Where("id = ?", a.ID).Updates(map[string]any{
		"code":          a.Code,
		"title":         a.Title,
		"media_type":    a.MediaType,
		"body_text":     a.BodyText,
		"media_url":     a.MediaURL,
		"content":       a.Content,
		"target_url":    a.TargetURL,
		"watch_seconds": a.WatchSeconds,
		"watch_reward":  a.WatchReward,
		"click_reward":  a.ClickReward,
		"starts_at":     a.StartsAt,
		"ends_at":       a.EndsAt,
		"active":        a.Active,
		"updated_at":    time.Now(),
	}).Error
}

// DeleteAd hard-deletes the ad and its sessions/clicks (ON DELETE CASCADE).
// Historical ledger entries survive; they carry no dependency on the ad row.
func (s *Store) DeleteAd(tx *gorm.DB, id string) error {
	return tx.Where("id = ?", id).Delete(&model.Ad{}).Error
}

// ListAds pages ads for the admin catalog; activeOnly trims out drafts.
func (s *Store) ListAds(tx *gorm.DB, limit, offset int, activeOnly bool) ([]model.Ad, int, error) {
	q := tx.Model(&model.Ad{})
	if activeOnly {
		q = q.Where("active = ?", true)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var ads []model.Ad
	err := q.Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&ads).Error
	return ads, int(total), err
}

// AdStats aggregates one ad's counters in a single round trip.
func (s *Store) AdStats(tx *gorm.DB, adID string) (
	sessions, rewarded, rewardedSum, clicks, clickRewards int64, err error,
) {
	err = tx.Raw(
		`SELECT
		   (SELECT COUNT(*) FROM ad_view_sessions WHERE ad_id = ?) AS sessions,
		   (SELECT COUNT(*) FROM ad_view_sessions WHERE ad_id = ? AND status = 'rewarded') AS rewarded,
		   (SELECT COALESCE(SUM(reward_minor), 0) FROM ad_view_sessions WHERE ad_id = ? AND status = 'rewarded') AS rewarded_sum,
		   (SELECT COUNT(*) FROM ads_clicks WHERE ad_id = ?) AS clicks,
		   (SELECT COUNT(*) FROM ads_clicks WHERE ad_id = ? AND reward_date IS NOT NULL AND user_id IS NOT NULL AND kind = 'click') AS click_rewards`,
		adID, adID, adID, adID, adID,
	).Row().Scan(&sessions, &rewarded, &rewardedSum, &clicks, &clickRewards)
	return
}
