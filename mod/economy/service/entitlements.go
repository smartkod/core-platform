package service

// entitlements.go implements econ.Entitlements.
//
// "Active" is computed on read from starts_at/ends_at/revoked_at rather than
// stored in a column. A stored boolean would need a job to flip it and would be
// wrong between runs; evaluating the window on the server clock is always
// right, and the partial unique index still prevents a double grant.

import (
	"context"
	"time"

	"core-platform/contract/core"
	econ "core-platform/contract/economy"
	"core-platform/mod/economy/model"
	"core-platform/mod/economy/repository"
)

// Entitlements is the entitlement service.
type Entitlements struct {
	store *repository.Store
	now   func() time.Time
}

// NewEntitlements builds the service.
func NewEntitlements(store *repository.Store, now func() time.Time) *Entitlements {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Entitlements{store: store, now: now}
}

// List implements econ.Entitlements.List.
func (e *Entitlements) List(ctx context.Context, userID string) ([]econ.Entitlement, error) {
	if userID == "" {
		return nil, core.ErrValidation
	}
	rows, err := e.store.ListEntitlements(e.store.DB, userID)
	if err != nil {
		return nil, err
	}
	now := e.now()
	out := make([]econ.Entitlement, 0, len(rows))
	for _, r := range rows {
		out = append(out, entitlementOf(r, now))
	}
	return out, nil
}

// Has implements econ.Entitlements.Has.
func (e *Entitlements) Has(ctx context.Context, userID, code string) (bool, error) {
	if userID == "" || code == "" {
		return false, core.ErrValidation
	}
	return e.store.HasActiveEntitlement(e.store.DB, userID, code, e.now())
}

// Revoke implements econ.Entitlements.Revoke.
func (e *Entitlements) Revoke(ctx context.Context, id, actorID string) error {
	if id == "" || actorID == "" {
		return core.ErrValidation
	}
	revoked, err := e.store.RevokeEntitlement(e.store.DB, id, e.now())
	if err != nil {
		return err
	}
	if !revoked {
		// Missing and already-revoked are the same answer to the caller: there
		// is nothing left to revoke. 404 is honest for both.
		return econ.ErrEntitlementNotFound
	}
	return nil
}

// entitlementOf maps the row and evaluates its window against the clock.
func entitlementOf(r model.Entitlement, now time.Time) econ.Entitlement {
	out := econ.Entitlement{
		ID:        r.ID,
		UserID:    r.UserID,
		Code:      r.Code,
		Source:    r.Source,
		StartsAt:  r.StartsAt,
		EndsAt:    r.EndsAt,
		RevokedAt: r.RevokedAt,
		CreatedAt: r.CreatedAt,
	}
	if r.ProductID != nil {
		out.ProductID = *r.ProductID
	}
	if r.OrderID != nil {
		out.OrderID = *r.OrderID
	}
	if r.RevokedAt == nil && !now.Before(r.StartsAt) &&
		(r.EndsAt == nil || now.Before(*r.EndsAt)) {
		out.Active = true
	}
	return out
}
