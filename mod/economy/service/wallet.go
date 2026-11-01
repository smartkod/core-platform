// Package service is the economy module's business layer.
//
// wallet.go owns the ledger invariants:
//   - amounts are int64 minor units, never float;
//   - the balance move is one conditional UPDATE, not read-then-write;
//   - every movement lands as exactly one append-only wallet_entries row;
//   - a replayed idempotency key returns the original entry, never a second
//     one.
package service

import (
	"context"
	"time"

	"gorm.io/gorm"

	"core-platform/contract/core"
	econ "core-platform/contract/economy"
	"core-platform/mod/economy/model"
	"core-platform/mod/economy/repository"
)

// Paging caps for Transactions; repeated requests can page, oversized pages
// are clamped rather than errors to keep rollover simple.
const (
	defaultPageLimit = 20
	hardPageLimit    = 100
)

// Wallet is the ledger service facade implementing econ.Wallet.
type Wallet struct {
	store *repository.Store
	// currencies is the allow-list from config (economy.currencies). An
	// unknown code is rejected rather than invented (L2.md 3.1).
	currencies map[string]bool
	now        func() time.Time
}

// NewWallet builds the wallet over a store; currencies may be empty, which
// enables just the default TOKEN code for scaffolded apps.
func NewWallet(store *repository.Store, currencies []string, mark core.Clock) *Wallet {
	allowed := map[string]bool{}
	if len(currencies) == 0 {
		currencies = []string{econ.CurrencyToken}
	}
	for _, c := range currencies {
		allowed[c] = true
	}
	now := func() time.Time { return time.Now().UTC() }
	if mark != nil {
		now = func() time.Time { return mark.Now().UTC() }
	}
	return &Wallet{store: store, currencies: allowed, now: now}
}

// knownCurrency validates a code against the allow-list.
func (w *Wallet) knownCurrency(code string) error {
	if !w.currencies[code] {
		return econ.ErrUnknownCurrency
	}
	return nil
}

// EnsureAccount implements econ.Wallet.EnsureAccount. On first creation the
// signup grant (from reward_settings) lands as one entry keyed
// signup_grant:<userID>; the unique ledger index makes replays impossible to
// double it even across concurrent first views.
func (w *Wallet) EnsureAccount(ctx context.Context, userID, currency string) (econ.Balance, error) {
	if currency == "" {
		currency = econ.CurrencyToken
	}
	if userID == "" {
		return econ.Balance{}, econ.ErrInvalidAmount
	}
	if err := w.knownCurrency(currency); err != nil {
		return econ.Balance{}, err
	}

	tx := w.store.DB.Begin()
	if tx.Error != nil {
		return econ.Balance{}, tx.Error
	}
	defer func() { _ = tx.Rollback().Error }()

	now := w.now()
	acc, created, err := w.store.EnsureAccount(tx, userID, currency, now)
	if err != nil {
		return econ.Balance{}, err
	}

	// Snapshot the settings inside the same tx: the grant amount the caller
	// sees is the amount actually paid.
	snap, err := settingsInTx(tx, now)
	if err != nil {
		return econ.Balance{}, err
	}

	if created && snap.SignupGrant > 0 {
		after, err := w.store.ApplyDelta(tx, acc.ID, snap.SignupGrant)
		if err != nil {
			return econ.Balance{}, err
		}
		if err := w.store.InsertEntry(tx, repository.NewEntry(
			acc.ID, snap.SignupGrant, after, econ.ReasonSignupGrant,
			"", "", "", "signup_grant:"+userID, now,
		)); err != nil {
			return econ.Balance{}, err
		}
		acc.BalanceMinor = after
	}

	if err := tx.Commit().Error; err != nil {
		return econ.Balance{}, err
	}
	return econ.Balance{
		UserID:    userID,
		Currency:  currency,
		Balance:   acc.BalanceMinor,
		UpdatedAt: now,
	}, nil
}

// moveOnTx runs one ledger movement on an open transaction handle. It does
// NOT commit: the caller owns the transaction, which is what lets the ads
// service combine a session flip and its ledger entry into one atomic move.
func (w *Wallet) moveOnTx(tx *gorm.DB, in econ.Movement, sign int) (econ.Entry, error) {
	if in.UserID == "" || in.Currency == "" || in.Amount <= 0 {
		return econ.Entry{}, econ.ErrInvalidAmount
	}
	if err := w.knownCurrency(in.Currency); err != nil {
		return econ.Entry{}, err
	}

	acc, _, err := w.store.EnsureAccount(tx, in.UserID, in.Currency, w.now())
	if err != nil {
		return econ.Entry{}, err
	}

	// Replay path: the key finds the original row without touching balances.
	if in.IdempotencyKey != "" {
		prev, found, err := w.store.EntryByIDempotency(tx, acc.ID, in.IdempotencyKey)
		if err != nil {
			return econ.Entry{}, err
		}
		if found {
			return entryOf(prev, in.Currency, in.UserID), nil
		}
	}

	delta := in.Amount
	if sign < 0 {
		delta = -delta
	}

	after, err := w.store.ApplyDelta(tx, acc.ID, delta)
	if err != nil {
		if err == repository.ErrReasonInsufficientBalance {
			return econ.Entry{}, core.ErrInsufficientBalance
		}
		return econ.Entry{}, err
	}

	row := repository.NewEntry(
		acc.ID, delta, after, in.Reason,
		in.ActorID, in.RefType, in.RefID, in.IdempotencyKey, w.now(),
	)
	if err := w.store.InsertEntry(tx, row); err != nil {
		// A concurrent duplicate with the same key is not a failure: whoever
		// wrote first owns the movement, so return their entry (fresh pool
		// read; our tx is about to roll back).
		if in.IdempotencyKey != "" {
			if prev, found, ferr := w.store.EntryByIDempotency(w.store.DB, acc.ID, in.IdempotencyKey); ferr == nil && found {
				return entryOf(prev, in.Currency, in.UserID), nil
			}
		}
		return econ.Entry{}, err
	}
	return entryOf(row, in.Currency, in.UserID), nil
}

// Credit implements econ.Wallet.Credit.
func (w *Wallet) Credit(ctx context.Context, in econ.Movement) (econ.Entry, error) {
	if in.Amount <= 0 {
		return econ.Entry{}, econ.ErrInvalidAmount
	}
	if err := w.knownCurrency(in.Currency); err != nil {
		return econ.Entry{}, err
	}
	tx := w.store.DB.Begin()
	if tx.Error != nil {
		return econ.Entry{}, tx.Error
	}
	defer func() { _ = tx.Rollback().Error }()
	out, err := w.moveOnTx(tx, in, +1)
	if err != nil {
		return econ.Entry{}, err
	}
	return out, tx.Commit().Error
}

// Debit implements econ.Wallet.Debit. A negative result is rejected with
// ErrInsufficientBalance and nothing is written.
func (w *Wallet) Debit(ctx context.Context, in econ.Movement) (econ.Entry, error) {
	if in.Amount <= 0 {
		return econ.Entry{}, econ.ErrInvalidAmount
	}
	if err := w.knownCurrency(in.Currency); err != nil {
		return econ.Entry{}, err
	}
	tx := w.store.DB.Begin()
	if tx.Error != nil {
		return econ.Entry{}, tx.Error
	}
	defer func() { _ = tx.Rollback().Error }()
	out, err := w.moveOnTx(tx, in, -1)
	if err != nil {
		return econ.Entry{}, err
	}
	return out, tx.Commit().Error
}

// settingsOf maps the dial row to the contract type.
func settingsOf(r model.RewardSettings) econ.RewardSettings {
	return econ.RewardSettings{
		SignupGrant:        r.SignupGrant,
		AdWatchReward:      r.AdWatchReward,
		DailyRewardCap:     r.DailyRewardCap,
		MonthlyRewardCap:   r.MonthlyRewardCap,
		ClickRewardEnabled: r.ClickRewardEnabled,
		UpdatedAt:          r.UpdatedAt,
		UpdatedBy:          deref(r.UpdatedBy),
	}
}

// settingsInTx loads the dial row inside the caller's transaction.
func settingsInTx(tx *gorm.DB, now time.Time) (econ.RewardSettings, error) {
	r, err := repository.NewStore(tx).Settings(tx, now)
	if err != nil {
		return econ.RewardSettings{}, err
	}
	return settingsOf(r), nil
}

// Balance implements econ.Wallet.Balance. A missing account reads as zero —
// the caller can call EnsureAccount to materialize it with the signup grant.
func (w *Wallet) Balance(ctx context.Context, userID, currency string) (econ.Balance, error) {
	if currency == "" {
		currency = econ.CurrencyToken
	}
	if err := w.knownCurrency(currency); err != nil {
		return econ.Balance{}, err
	}
	var acc model.WalletAccount
	err := w.store.DB.Where("user_id = ? AND currency = ?", userID, currency).First(&acc).Error
	if err == gorm.ErrRecordNotFound {
		return econ.Balance{UserID: userID, Currency: currency, Balance: 0, UpdatedAt: w.now()}, nil
	}
	if err != nil {
		return econ.Balance{}, err
	}
	return econ.Balance{UserID: userID, Currency: currency, Balance: acc.BalanceMinor, UpdatedAt: w.now()}, nil
}

// Transactions implements econ.Wallet.Transactions with paging caps.
func (w *Wallet) Transactions(ctx context.Context, userID, currency string, limit, offset int) ([]econ.Entry, int, error) {
	if currency == "" {
		currency = econ.CurrencyToken
	}
	if limit <= 0 {
		limit = defaultPageLimit
	}
	if limit > hardPageLimit {
		limit = hardPageLimit
	}
	if offset < 0 {
		offset = 0
	}

	var rows []model.WalletEntry
	db := w.store.DB.
		Table("wallet_entries e").
		Joins("JOIN wallet_accounts a ON a.id = e.account_id").
		Where("a.user_id = ? AND a.currency = ?", userID, currency).
		Order("e.created_at DESC").
		Limit(limit).Offset(offset)
	if err := db.Select("e.*").Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	var total int64
	if err := w.store.DB.
		Table("wallet_entries e").
		Joins("JOIN wallet_accounts a ON a.id = e.account_id").
		Where("a.user_id = ? AND a.currency = ?", userID, currency).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	out := make([]econ.Entry, 0, len(rows))
	for _, r := range rows {
		out = append(out, entryOf(r, currency, userID))
	}
	return out, int(total), nil
}

// entryOf maps a ledger row to the contract type. The user/currency come from
// the owning account, which the caller knows.
func entryOf(r model.WalletEntry, currency, userID string) econ.Entry {
	return econ.Entry{
		ID:             r.ID,
		AccountID:      r.AccountID,
		UserID:         userID,
		Currency:       currency,
		Delta:          r.DeltaMinor,
		BalanceAfter:   r.BalanceAfter,
		Reason:         r.ReasonCode,
		ActorID:        deref(r.ActorID),
		RefType:        deref(r.RefType),
		RefID:          deref(r.RefID),
		IdempotencyKey: deref(r.IdempotencyKey),
		CreatedAt:      r.CreatedAt,
	}
}

// helpers kept here so the models package stays data-only.
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
