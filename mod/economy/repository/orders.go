package repository

// orders.go holds the catalog, order, payment, event and entitlement queries.
//
// Two rules shape every method here:
//   - order state moves only through TransitionOrder, one conditional UPDATE
//     with the expected status, so a stale writer cannot skip a step;
//   - fulfillment is idempotent in the schema (unique partial indexes on
//     entitlements.order_id and payments.order_id where paid), not only in the
//     service, so a retried webhook cannot grant twice.

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"core-platform/mod/economy/model"
)

// ------------------------------------------------------------------
// Catalog
// ------------------------------------------------------------------

// ListProducts returns the catalog, newest first.
func (s *Store) ListProducts(tx *gorm.DB, activeOnly bool) ([]model.Product, error) {
	q := tx.Model(&model.Product{})
	if activeOnly {
		q = q.Where("active = ?", true)
	}
	var out []model.Product
	err := q.Order("created_at DESC").Find(&out).Error
	return out, err
}

// ProductByIDOrCode resolves a product by uuid or by its stable code.
func (s *Store) ProductByIDOrCode(tx *gorm.DB, idOrCode string) (model.Product, bool, error) {
	var p model.Product
	// A uuid-shaped value is tried as an id first, then as a code: codes are
	// free-form, so the reverse order would be ambiguous.
	err := tx.Where("id::text = ? OR code = ?", idOrCode, idOrCode).First(&p).Error
	if err == gorm.ErrRecordNotFound {
		return model.Product{}, false, nil
	}
	if err != nil {
		return model.Product{}, false, err
	}
	return p, true, nil
}

// InsertProduct is the admin create path.
func (s *Store) InsertProduct(tx *gorm.DB, p model.Product) error {
	return tx.Create(&p).Error
}

// UpdateProduct rewrites the mutable product fields. The catalog is admin
// managed and small, so a full update keeps the surface simple.
func (s *Store) UpdateProduct(tx *gorm.DB, p model.Product) error {
	return tx.Model(&model.Product{}).Where("id = ?", p.ID).Updates(map[string]any{
		"code":         p.Code,
		"name":         p.Name,
		"kind":         p.Kind,
		"currency":     p.Currency,
		"amount_minor": p.AmountMinor,
		"interval":     p.Interval,
		"meta":         p.Meta,
		"active":       p.Active,
		"updated_at":   time.Now().UTC(),
	}).Error
}

// ------------------------------------------------------------------
// Orders
// ------------------------------------------------------------------

// OrderByIDempotencyKey is the replay lookup: the same key must return the
// original order instead of charging again.
func (s *Store) OrderByIDempotencyKey(tx *gorm.DB, key string) (model.Order, bool, error) {
	var o model.Order
	err := tx.Where("idempotency_key = ?", key).First(&o).Error
	if err == gorm.ErrRecordNotFound {
		return model.Order{}, false, nil
	}
	if err != nil {
		return model.Order{}, false, err
	}
	return o, true, nil
}

// OrderByID loads one order.
func (s *Store) OrderByID(tx *gorm.DB, id string) (model.Order, bool, error) {
	var o model.Order
	err := tx.Where("id = ?", id).First(&o).Error
	if err == gorm.ErrRecordNotFound {
		return model.Order{}, false, nil
	}
	if err != nil {
		return model.Order{}, false, err
	}
	return o, true, nil
}

// OrderForUpdate locks one order for the fulfillment transaction, so two
// concurrent confirmations cannot both see "pending".
func (s *Store) OrderForUpdate(tx *gorm.DB, id string) (model.Order, bool, error) {
	var o model.Order
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).First(&o).Error
	if err == gorm.ErrRecordNotFound {
		return model.Order{}, false, nil
	}
	if err != nil {
		return model.Order{}, false, err
	}
	return o, true, nil
}

// OrderByReferenceCode resolves a webhook's order reference without trusting
// the provider to echo a uuid.
func (s *Store) OrderByReferenceCode(tx *gorm.DB, code string) (model.Order, bool, error) {
	var o model.Order
	err := tx.Where("reference_code = ?", code).First(&o).Error
	if err == gorm.ErrRecordNotFound {
		return model.Order{}, false, nil
	}
	if err != nil {
		return model.Order{}, false, err
	}
	return o, true, nil
}

// InsertOrder creates the order row.
func (s *Store) InsertOrder(tx *gorm.DB, o model.Order) error {
	return tx.Create(&o).Error
}

// TransitionOrder moves status from -> to and reports whether it moved.
// A false result means somebody else already transitioned it; the caller
// re-reads and decides, rather than overwriting.
func (s *Store) TransitionOrder(tx *gorm.DB, id, from, to string, now time.Time) (bool, error) {
	res := tx.Model(&model.Order{}).
		Where("id = ? AND status = ?", id, from).
		Updates(map[string]any{"status": to, "updated_at": now})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// SetOrderCheckout records the provider and its reference and moves the order
// created -> pending in the same statement.
func (s *Store) SetOrderCheckout(tx *gorm.DB, id, provider, reference string, now time.Time) (bool, error) {
	res := tx.Model(&model.Order{}).
		Where("id = ? AND status = ?", id, "created").
		Updates(map[string]any{
			"provider":       provider,
			"reference_code": reference,
			"status":         "pending",
			"updated_at":     now,
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// ListOrders pages one user's orders.
func (s *Store) ListOrders(tx *gorm.DB, userID string, limit, offset int) ([]model.Order, int, error) {
	q := tx.Model(&model.Order{}).Where("user_id = ?", userID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []model.Order
	err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&out).Error
	return out, int(total), err
}

// ListAllOrders is the admin list, optionally filtered by status.
func (s *Store) ListAllOrders(tx *gorm.DB, status string, limit, offset int) ([]model.Order, int, error) {
	q := tx.Model(&model.Order{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []model.Order
	err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&out).Error
	return out, int(total), err
}

// ExpireOverdue moves pending/created orders past expires_at to expired and
// returns how many moved. The scheduled job calls it; the order service also
// calls it lazily before a state-dependent action.
func (s *Store) ExpireOverdue(tx *gorm.DB, now time.Time) (int64, error) {
	res := tx.Exec(
		`UPDATE orders SET status = 'expired', updated_at = ?
		  WHERE status IN ('created', 'pending') AND expires_at < ?`,
		now, now,
	)
	return res.RowsAffected, res.Error
}

// ------------------------------------------------------------------
// Payments and webhook events
// ------------------------------------------------------------------

// InsertPaymentOnce appends a payment row unless a paid row already exists for
// the order. inserted reports whether this call wrote it, which is what gates
// fulfillment.
func (s *Store) InsertPaymentOnce(tx *gorm.DB, p model.Payment) (bool, error) {
	res := tx.Exec(
		`INSERT INTO payments (id, order_id, provider, provider_ref, status,
		                       amount_minor, currency, paid_at, failure_code,
		                       raw_payload, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (order_id) WHERE status = 'paid' DO NOTHING`,
		p.ID, p.OrderID, p.Provider, p.ProviderRef, p.Status,
		p.AmountMinor, p.Currency, p.PaidAt, p.FailureCode,
		p.RawPayload, p.CreatedAt,
	)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// InsertEventOnce records a webhook delivery. A duplicate (provider, event_id)
// returns false: the caller answers 200 and does no further work.
func (s *Store) InsertEventOnce(tx *gorm.DB, e model.PaymentEvent) (bool, error) {
	res := tx.Exec(
		`INSERT INTO payment_events (id, provider, event_id, order_id, type, payload, processed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (provider, event_id) DO NOTHING`,
		e.ID, e.Provider, e.EventID, e.OrderID, e.Type, e.Payload, e.ProcessedAt,
	)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// ------------------------------------------------------------------
// Entitlements
// ------------------------------------------------------------------

// GrantEntitlementOnce inserts the entitlement unless the order already has an
// active one. granted reports whether this call wrote it.
func (s *Store) GrantEntitlementOnce(tx *gorm.DB, e model.Entitlement) (bool, error) {
	res := tx.Exec(
		`INSERT INTO entitlements (id, user_id, product_id, order_id, code, source,
		                           starts_at, ends_at, revoked_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, NULL, ?)
		 ON CONFLICT (order_id) WHERE order_id IS NOT NULL AND revoked_at IS NULL
		 DO NOTHING`,
		e.ID, e.UserID, e.ProductID, e.OrderID, e.Code, e.Source,
		e.StartsAt, e.EndsAt, e.CreatedAt,
	)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// ListEntitlements returns the user's rows, active window first.
func (s *Store) ListEntitlements(tx *gorm.DB, userID string) ([]model.Entitlement, error) {
	var out []model.Entitlement
	err := tx.Where("user_id = ?", userID).
		Order("revoked_at NULLS FIRST, ends_at NULLS LAST, created_at DESC").
		Find(&out).Error
	return out, err
}

// HasActiveEntitlement answers the authorization question: not revoked and
// inside the window, evaluated on the server clock.
func (s *Store) HasActiveEntitlement(tx *gorm.DB, userID, code string, now time.Time) (bool, error) {
	var n int64
	err := tx.Model(&model.Entitlement{}).
		Where(`user_id = ? AND code = ? AND revoked_at IS NULL
		       AND starts_at <= ? AND (ends_at IS NULL OR ends_at > ?)`,
			userID, code, now, now).
		Count(&n).Error
	return n > 0, err
}

// RevokeEntitlement ends one entitlement now. It reports false when the row is
// missing or already revoked, so the caller can answer 404 instead of
// silently succeeding.
func (s *Store) RevokeEntitlement(tx *gorm.DB, id string, now time.Time) (bool, error) {
	res := tx.Model(&model.Entitlement{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Updates(map[string]any{"revoked_at": now})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// EntitlementByID loads one row (used for ownership checks).
func (s *Store) EntitlementByID(tx *gorm.DB, id string) (model.Entitlement, bool, error) {
	var e model.Entitlement
	err := tx.Where("id = ?", id).First(&e).Error
	if err == gorm.ErrRecordNotFound {
		return model.Entitlement{}, false, nil
	}
	if err != nil {
		return model.Entitlement{}, false, err
	}
	return e, true, nil
}

// ExpireEntitlements is the subscription job's body: nothing to update (the
// window is evaluated on read), so it only exists to be callable from a job
// that wants a count for reporting.
func (s *Store) ActiveEntitlementCount(tx *gorm.DB, now time.Time) (int64, error) {
	var n int64
	err := tx.Model(&model.Entitlement{}).
		Where(`revoked_at IS NULL AND starts_at <= ?
		       AND (ends_at IS NULL OR ends_at > ?)`, now, now).
		Count(&n).Error
	return n, err
}

// newID is the shared uuid helper for this file.
func newID() string { return uuid.NewString() }
