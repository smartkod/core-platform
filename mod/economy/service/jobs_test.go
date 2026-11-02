package service_test

import (
	"context"
	"testing"
	"time"

	econ "core-platform/contract/economy"
	"core-platform/mod/economy/repository"
	"core-platform/mod/economy/service"
)

// TestJobsReportLedgerClean proves the periodic invariant check finds nothing
// on a healthy ledger. The drift query is the same one a real incident would
// surface.
func TestJobsReportLedgerClean(t *testing.T) {
	db := economyDB(t)
	clock := newClock()
	setDial(t, db, 10, 1, 3, 10, true)
	w, userID := freshWallet(t, db, clock)
	ctx := context.Background()

	if _, err := w.EnsureAccount(ctx, userID, econ.CurrencyToken); err != nil {
		t.Fatal(err)
	}

	jobs := service.NewJobs(repository.NewStore(db), clock.Now)
	report, err := jobs.RunOnce(ctx)
	if err != nil {
		t.Fatalf("run jobs: %v", err)
	}
	if report.Skipped {
		t.Fatal("first sweep was skipped; the advisory lock should be free")
	}
	if len(report.LedgerDrift) != 0 {
		t.Fatalf("healthy ledger reported drift: %+v", report.LedgerDrift)
	}
}

// TestJobsExpireOrders shows the sweep closing a stale pending order.
func TestJobsExpireOrders(t *testing.T) {
	db := economyDB(t)
	clock := newClock()
	setDial(t, db, 0, 1, 10, 100, true)
	s := newShop(t, db, clock)
	ctx := context.Background()

	p := s.product(t, db, econ.ProductInput{
		Kind: econ.KindTokenPack, Currency: econ.CurrencyTRY,
		AmountMinor: 1000, Tokens: 10, Active: true,
	})
	order, err := s.orders.Create(ctx, econ.CreateOrderInput{
		UserID: s.userID, ProductID: p.ID, IdempotencyKey: "job-" + uuidStr(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if order.Status != econ.OrderPending {
		t.Fatalf("order should be pending: %s", order.Status)
	}

	// The shop is built with a one-hour TTL; advance past it and sweep.
	clock.Advance(2 * time.Hour)
	jobs := service.NewJobs(repository.NewStore(db), clock.Now)
	report, err := jobs.RunOnce(ctx)
	if err != nil {
		t.Fatalf("run jobs: %v", err)
	}
	if report.ExpiredOrders == 0 {
		t.Fatal("the sweep did not expire the overdue order")
	}

	updated, err := s.orders.Get(ctx, s.userID, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != econ.OrderExpired {
		t.Fatalf("status after sweep: %s want expired", updated.Status)
	}
}
