package service_test

import (
	"context"
	"errors"
	"testing"

	econ "core-platform/contract/economy"
)

// TestIdempotencyKeyCannotReplayAnotherUsersOrder closes an information leak:
// the order idempotency key is globally unique, so without an ownership check a
// second user presenting the same key would receive the first user's order
// (including its amount and status). The correct answer is 409, not a replay.
func TestIdempotencyKeyCannotReplayAnotherUsersOrder(t *testing.T) {
	db := economyDB(t)
	clock := newClock()
	setDial(t, db, 0, 1, 10, 100, true)
	s := newShop(t, db, clock)
	ctx := context.Background()

	if _, err := s.wallet.Credit(ctx, econ.Movement{
		UserID: s.userID, Currency: econ.CurrencyToken, Amount: 100,
		Reason: econ.ReasonAdminCredit,
	}); err != nil {
		t.Fatal(err)
	}
	p := s.product(t, db, econ.ProductInput{
		Kind: econ.KindRegular, Currency: econ.CurrencyToken,
		AmountMinor: 10, Entitlement: "owned.order", Active: true,
	})

	key := "shared-" + uuidStr()
	first, err := s.orders.Create(ctx, econ.CreateOrderInput{
		UserID: s.userID, ProductID: p.ID, IdempotencyKey: key,
	})
	if err != nil {
		t.Fatal(err)
	}

	other := uuidStr()
	if _, err := s.orders.Create(ctx, econ.CreateOrderInput{
		UserID: other, ProductID: p.ID, IdempotencyKey: key,
	}); !errors.Is(err, econ.ErrIdempotencyConflict) {
		t.Fatalf("foreign idempotency key: want conflict, got %v", err)
	}

	// The stranger's attempt must not have produced a second order either.
	var count int64
	db.Raw(`SELECT COUNT(*) FROM orders WHERE idempotency_key = ?`, key).Scan(&count)
	if count != 1 {
		t.Fatalf("orders with the shared key: %d want 1", count)
	}

	// And the owner still gets a normal replay.
	replay, err := s.orders.Create(ctx, econ.CreateOrderInput{
		UserID: s.userID, ProductID: p.ID, IdempotencyKey: key,
	})
	if err != nil {
		t.Fatalf("owner replay: %v", err)
	}
	if replay.ID != first.ID {
		t.Fatalf("owner replay returned a different order: %s vs %s", replay.ID, first.ID)
	}
}
