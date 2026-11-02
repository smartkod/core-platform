package service

// jobs.go implements the periodic economy sweep (L2.md 3.9).
//
// The three tasks are deliberately cheap and idempotent:
//
//   - expire orders whose payment window closed;
//   - count the entitlements still inside their window (subscription report;
//     access itself is evaluated on read, so nothing has to be flipped);
//   - verify SUM(entries) == balance for every wallet.
//
// A transaction-scoped advisory lock means several instances can run the
// schedule without duplicating the sweep: the losers skip this tick and the
// next tick runs normally.

import (
	"context"
	"time"

	"gorm.io/gorm"

	"core-platform/mod/economy/repository"
)

// Jobs runs the scheduled economy maintenance.
type Jobs struct {
	store *repository.Store
	now   func() time.Time
}

// NewJobs builds the job runner. A nil clock means the real UTC clock.
func NewJobs(store *repository.Store, now func() time.Time) *Jobs {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Jobs{store: store, now: now}
}

// JobsReport is what one sweep did, logged by the module.
type JobsReport struct {
	// Skipped is true when another instance held the advisory lock.
	Skipped bool
	// ExpiredOrders is how many orders the sweep closed.
	ExpiredOrders int64
	// ActiveEntitlements is the subscription report count.
	ActiveEntitlements int64
	// LedgerDrift lists accounts that break the ledger invariant. Any row here
	// is an incident, not a routine statistic.
	LedgerDrift []repository.LedgerDrift
}

// RunOnce performs one sweep. It is safe to call from any number of instances.
func (j *Jobs) RunOnce(ctx context.Context) (JobsReport, error) {
	var report JobsReport

	err := j.store.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// pg_try_advisory_xact_lock returns false instead of blocking, so a
		// second instance reports "skipped" rather than piling up waiters; the
		// lock is released when the transaction ends.
		var locked bool
		if err := tx.Raw(`SELECT pg_try_advisory_xact_lock(hashtext(?))`, "economy.jobs").
			Scan(&locked).Error; err != nil {
			return err
		}
		if !locked {
			report.Skipped = true
			return nil
		}

		now := j.now()
		expired, err := j.store.ExpireOverdue(tx, now)
		if err != nil {
			return err
		}
		report.ExpiredOrders = expired

		active, err := j.store.ActiveEntitlementCount(tx, now)
		if err != nil {
			return err
		}
		report.ActiveEntitlements = active

		drift, err := j.store.LedgerDrift(tx)
		if err != nil {
			return err
		}
		report.LedgerDrift = drift
		return nil
	})
	if err != nil {
		return JobsReport{}, err
	}
	return report, nil
}
