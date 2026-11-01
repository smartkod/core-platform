package service_test

import (
	"context"
	"errors"
	"testing"

	"core-platform/contract/core"
	econ "core-platform/contract/economy"
)

func TestEnsureAccountGrantsOnce(t *testing.T) {
	db := economyDB(t)
	clock := newClock()
	setDial(t, db, 10, 1, 3, 10, true)
	w, userID := freshWallet(t, db, clock)
	ctx := context.Background()

	first, err := w.EnsureAccount(ctx, userID, econ.CurrencyToken)
	if err != nil {
		t.Fatalf("first ensure: %v", err)
	}
	if first.Balance != 10 {
		t.Fatalf("signup grant: got %d want 10", first.Balance)
	}

	second, err := w.EnsureAccount(ctx, userID, econ.CurrencyToken)
	if err != nil {
		t.Fatalf("second ensure: %v", err)
	}
	if second.Balance != 10 {
		t.Fatalf("grant replayed: got %d want 10", second.Balance)
	}

	entries, total, err := w.Transactions(ctx, userID, econ.CurrencyToken, 50, 0)
	if err != nil {
		t.Fatalf("transactions: %v", err)
	}
	if total != 1 || len(entries) != 1 {
		t.Fatalf("welcome grant must be exactly one entry, got total=%d len=%d", total, len(entries))
	}
	if entries[0].Reason != econ.ReasonSignupGrant || entries[0].Delta != 10 {
		t.Fatalf("unexpected grant entry: %+v", entries[0])
	}
}

func TestBalanceEqualsLedgerSum(t *testing.T) {
	db := economyDB(t)
	clock := newClock()
	setDial(t, db, 0, 1, 10, 100, true)
	w, userID := freshWallet(t, db, clock)
	ctx := context.Background()

	if _, err := w.EnsureAccount(ctx, userID, econ.CurrencyToken); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		if _, err := w.Credit(ctx, econ.Movement{
			UserID: userID, Currency: econ.CurrencyToken, Amount: 7,
			Reason: econ.ReasonAdminCredit,
		}); err != nil {
			t.Fatalf("credit %d: %v", i, err)
		}
	}
	for i := 0; i < 3; i++ {
		if _, err := w.Debit(ctx, econ.Movement{
			UserID: userID, Currency: econ.CurrencyToken, Amount: 4,
			Reason: econ.ReasonAdminDebit,
		}); err != nil {
			t.Fatalf("debit %d: %v", i, err)
		}
	}

	bal, err := w.Balance(ctx, userID, econ.CurrencyToken)
	if err != nil {
		t.Fatal(err)
	}
	if want := int64(7*5 - 4*3); bal.Balance != want {
		t.Fatalf("balance: got %d want %d", bal.Balance, want)
	}

	var sum int64
	if err := db.Raw(
		`SELECT COALESCE(SUM(e.delta_minor), 0) FROM wallet_entries e
		   JOIN wallet_accounts a ON a.id = e.account_id
		  WHERE a.user_id = ?`, userID,
	).Scan(&sum).Error; err != nil {
		t.Fatal(err)
	}
	if sum != bal.Balance {
		t.Fatalf("invariant broken: ledger sum %d != balance %d", sum, bal.Balance)
	}
}

func TestDebitBelowZeroWritesNothing(t *testing.T) {
	db := economyDB(t)
	clock := newClock()
	setDial(t, db, 0, 1, 10, 100, true)
	w, userID := freshWallet(t, db, clock)
	ctx := context.Background()

	if _, err := w.EnsureAccount(ctx, userID, econ.CurrencyToken); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Credit(ctx, econ.Movement{
		UserID: userID, Currency: econ.CurrencyToken, Amount: 5,
		Reason: econ.ReasonAdminCredit,
	}); err != nil {
		t.Fatal(err)
	}

	_, err := w.Debit(ctx, econ.Movement{
		UserID: userID, Currency: econ.CurrencyToken, Amount: 6,
		Reason: econ.ReasonAdminDebit,
	})
	if !errors.Is(err, core.ErrInsufficientBalance) {
		t.Fatalf("want insufficient balance, got %v", err)
	}

	bal, _ := w.Balance(ctx, userID, econ.CurrencyToken)
	if bal.Balance != 5 {
		t.Fatalf("failed debit moved the balance: %d", bal.Balance)
	}
	var n int64
	db.Raw(`SELECT COUNT(*) FROM wallet_entries e
	          JOIN wallet_accounts a ON a.id = e.account_id
	         WHERE a.user_id = ? AND e.reason_code = ?`,
		userID, econ.ReasonAdminDebit).Scan(&n)
	if n != 0 {
		t.Fatalf("failed debit left %d ledger rows", n)
	}
}

func TestIdempotentMovementReplays(t *testing.T) {
	db := economyDB(t)
	clock := newClock()
	setDial(t, db, 0, 1, 10, 100, true)
	w, userID := freshWallet(t, db, clock)
	ctx := context.Background()

	in := econ.Movement{
		UserID: userID, Currency: econ.CurrencyToken, Amount: 9,
		Reason: econ.ReasonPurchase, IdempotencyKey: "order:abc",
	}
	a, err := w.Credit(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	b, err := w.Credit(ctx, in)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if a.ID != b.ID {
		t.Fatalf("replay made a second entry: %s vs %s", a.ID, b.ID)
	}

	bal, _ := w.Balance(ctx, userID, econ.CurrencyToken)
	if bal.Balance != 9 {
		t.Fatalf("replay double-credited: %d", bal.Balance)
	}
}

func TestLedgerIsAppendOnly(t *testing.T) {
	db := economyDB(t)
	clock := newClock()
	setDial(t, db, 0, 1, 10, 100, true)
	w, userID := freshWallet(t, db, clock)
	ctx := context.Background()

	entry, err := w.Credit(ctx, econ.Movement{
		UserID: userID, Currency: econ.CurrencyToken, Amount: 2,
		Reason: econ.ReasonAdminCredit,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`UPDATE wallet_entries SET delta_minor = 999 WHERE id = ?`, entry.ID).Error; err == nil {
		t.Fatal("UPDATE on wallet_entries succeeded; the append-only trigger is missing")
	}
	if err := db.Exec(`DELETE FROM wallet_entries WHERE id = ?`, entry.ID).Error; err == nil {
		t.Fatal("DELETE on wallet_entries succeeded; the append-only trigger is missing")
	}
}

func TestUnknownCurrencyRejected(t *testing.T) {
	db := economyDB(t)
	clock := newClock()
	w, userID := freshWallet(t, db, clock)
	if _, err := w.EnsureAccount(context.Background(), userID, "BTC"); !errors.Is(err, econ.ErrUnknownCurrency) {
		t.Fatalf("want unknown currency, got %v", err)
	}
}
