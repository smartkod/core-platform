package repository

// jobs.go holds the queries the scheduled economy jobs run: the order expiry
// sweep and the ledger invariant check (L2.md 3.9).
//
// Both are plain queries on a caller-owned transaction so the job runner can do
// them under one advisory lock and report one consistent picture.

import (
	"gorm.io/gorm"
)

// LedgerDrift is one wallet account whose stored balance disagrees with the sum
// of its entries.
//
// It should never exist: the wallet writes the balance and the entry in the same
// transaction and the schema forbids a rewrite. Finding a row therefore means a
// real bug or manual database surgery, which is exactly why the job reports it
// instead of silently repairing it.
type LedgerDrift struct {
	AccountID    string `gorm:"column:account_id"`
	UserID       string `gorm:"column:user_id"`
	Currency     string `gorm:"column:currency"`
	BalanceMinor int64  `gorm:"column:balance_minor"`
	EntrySum     int64  `gorm:"column:entry_sum"`
}

// LedgerDrift returns every account that breaks SUM(delta) == balance. The
// query walks all accounts, so it is a periodic job rather than a request-path
// check.
func (s *Store) LedgerDrift(tx *gorm.DB) ([]LedgerDrift, error) {
	var out []LedgerDrift
	err := tx.Raw(
		`SELECT a.id AS account_id, a.user_id, a.currency,
		        a.balance_minor,
		        COALESCE(SUM(e.delta_minor), 0)::bigint AS entry_sum
		   FROM wallet_accounts a
		   LEFT JOIN wallet_entries e ON e.account_id = a.id
		  GROUP BY a.id, a.user_id, a.currency, a.balance_minor
		 HAVING a.balance_minor <> COALESCE(SUM(e.delta_minor), 0)
		  ORDER BY a.created_at`,
	).Scan(&out).Error
	return out, err
}
