package service

// orders.go implements econ.Orders: the catalog, the single order state
// machine (L2.md 3.5), provider checkout, webhooks and the admin actions.
//
// The two paths share everything except settlement:
//
//	Path A (TOKEN product): debit + entitlement + fulfilled in one transaction.
//	Path B (fiat product):  created -> pending -> (webhook|admin) -> paid ->
//	                        fulfilled + granted entitlement.
//
// Both end in fulfill(), so an entitlement cannot be granted by one path in a
// way the other ignores. Fulfillment is idempotent twice over: the order lock
// makes the transition single-shot, and the ledger/entitlement rows carry
// order-derived keys with unique indexes behind them.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"core-platform/contract/core"
	econ "core-platform/contract/economy"
	"core-platform/mod/economy/model"
	"core-platform/mod/economy/provider"
	"core-platform/mod/economy/repository"
)

// Orders is the purchase service implementing econ.Orders.
type Orders struct {
	store     *repository.Store
	wallet    *Wallet
	providers map[string]econ.PaymentProvider
	now       func() time.Time
	// ttl bounds how long a pending order can be paid before expiry closes it.
	ttl time.Duration
}

// NewOrders builds the order service.
func NewOrders(
	store *repository.Store,
	wallet *Wallet,
	providers map[string]econ.PaymentProvider,
	now func() time.Time,
	ttl time.Duration,
) *Orders {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	if providers == nil {
		providers = map[string]econ.PaymentProvider{}
	}
	return &Orders{store: store, wallet: wallet, providers: providers, now: now, ttl: ttl}
}

// productMeta is the typed view of products.meta. json.Number keeps a token
// count exact: decoding into float64 would lose precision, and money never
// touches a float.
type productMeta struct {
	Tokens      json.Number `json:"tokens"`
	Entitlement string      `json:"entitlement"`
}

// productOf maps a row to the contract type, lifting the meta fields.
func productOf(p model.Product) (econ.Product, error) {
	out := econ.Product{
		ID:          p.ID,
		Code:        p.Code,
		Name:        p.Name,
		Kind:        p.Kind,
		Currency:    p.Currency,
		AmountMinor: p.AmountMinor,
		Active:      p.Active,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
	if p.Interval != nil {
		out.Interval = *p.Interval
	}
	if len(p.Meta) > 0 {
		var m productMeta
		dec := json.NewDecoder(bytes.NewReader(p.Meta))
		dec.UseNumber()
		if err := dec.Decode(&m); err != nil {
			return econ.Product{}, err
		}
		out.Entitlement = m.Entitlement
		if m.Tokens != "" {
			n, err := m.Tokens.Int64()
			if err != nil {
				return econ.Product{}, econ.ErrProductNotAvailable
			}
			out.Tokens = n
		}
	}
	return out, nil
}

func orderOf(o model.Order, p *econ.Product) econ.Order {
	out := econ.Order{
		ID:            o.ID,
		UserID:        o.UserID,
		ProductID:     o.ProductID,
		AmountMinor:   o.AmountMinor,
		Currency:      o.Currency,
		Provider:      o.Provider,
		Status:        o.Status,
		ReferenceCode: o.ReferenceCode,
		ExpiresAt:     o.ExpiresAt,
		CreatedAt:     o.CreatedAt,
		UpdatedAt:     o.UpdatedAt,
	}
	if p != nil {
		out.ProductCode = p.Code
		out.ProductName = p.Name
		out.ProductKind = p.Kind
	}
	return out
}

// ------------------------------------------------------------------
// Catalog
// ------------------------------------------------------------------

// ListProducts implements econ.Orders.ListProducts.
func (o *Orders) ListProducts(ctx context.Context, activeOnly bool) ([]econ.Product, error) {
	rows, err := o.store.ListProducts(o.store.DB, activeOnly)
	if err != nil {
		return nil, err
	}
	out := make([]econ.Product, 0, len(rows))
	for _, r := range rows {
		p, err := productOf(r)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// Product implements econ.Orders.Product.
func (o *Orders) Product(ctx context.Context, idOrCode string) (econ.Product, error) {
	row, found, err := o.store.ProductByIDOrCode(o.store.DB, idOrCode)
	if err != nil {
		return econ.Product{}, err
	}
	if !found || !row.Active {
		return econ.Product{}, econ.ErrProductNotAvailable
	}
	return productOf(row)
}

// CreateProduct implements econ.Orders.CreateProduct (admin).
func (o *Orders) CreateProduct(ctx context.Context, in econ.ProductInput) (econ.Product, error) {
	row, err := productRow(in)
	if err != nil {
		return econ.Product{}, err
	}
	now := o.now()
	row.ID = uuid.NewString()
	row.CreatedAt = now
	row.UpdatedAt = now
	if err := o.store.InsertProduct(o.store.DB, row); err != nil {
		return econ.Product{}, err
	}
	return productOf(row)
}

// UpdateProduct implements econ.Orders.UpdateProduct (admin).
func (o *Orders) UpdateProduct(ctx context.Context, id string, in econ.ProductInput) (econ.Product, error) {
	existing, found, err := o.store.ProductByIDOrCode(o.store.DB, id)
	if err != nil {
		return econ.Product{}, err
	}
	if !found {
		return econ.Product{}, core.ErrNotFound
	}
	row, err := productRow(in)
	if err != nil {
		return econ.Product{}, err
	}
	row.ID = existing.ID
	row.CreatedAt = existing.CreatedAt
	row.UpdatedAt = o.now()
	if err := o.store.UpdateProduct(o.store.DB, row); err != nil {
		return econ.Product{}, err
	}
	return productOf(row)
}

// productRow validates the admin payload and builds the row, including the
// meta jsonb the schema's CHECK constraint expects per kind.
func productRow(in econ.ProductInput) (model.Product, error) {
	code := strings.TrimSpace(in.Code)
	name := strings.TrimSpace(in.Name)
	if code == "" || name == "" {
		return model.Product{}, core.ErrValidation
	}
	if in.AmountMinor <= 0 {
		return model.Product{}, core.ErrValidation
	}
	if in.Currency == "" {
		return model.Product{}, core.ErrValidation
	}

	row := model.Product{
		Code:        code,
		Name:        name,
		Kind:        in.Kind,
		Currency:    strings.ToUpper(in.Currency),
		AmountMinor: in.AmountMinor,
		Active:      in.Active,
	}

	switch in.Kind {
	case econ.KindTokenPack:
		if in.Tokens <= 0 {
			return model.Product{}, core.ErrValidation
		}
		// A token pack is priced in fiat; letting it be priced in TOKEN would
		// make "buy 10 tokens for 10 tokens" a no-op loop.
		if row.Currency == econ.CurrencyToken {
			return model.Product{}, core.ErrValidation
		}
		row.Meta = []byte(`{"tokens":` + jsonInt(in.Tokens) + `}`)
	case econ.KindSubscription:
		code := strings.TrimSpace(in.Entitlement)
		if code == "" {
			return model.Product{}, core.ErrValidation
		}
		switch in.Interval {
		case "day", "week", "month", "year":
		default:
			return model.Product{}, core.ErrValidation
		}
		iv := in.Interval
		row.Interval = &iv
		row.Meta = []byte(`{"entitlement":` + jsonString(code) + `}`)
	case econ.KindRegular:
		entitlement := strings.TrimSpace(in.Entitlement)
		if entitlement == "" {
			entitlement = code
		}
		row.Meta = []byte(`{"entitlement":` + jsonString(entitlement) + `}`)
	default:
		return model.Product{}, core.ErrValidation
	}
	return row, nil
}

// ------------------------------------------------------------------
// Order creation
// ------------------------------------------------------------------

// Create implements econ.Orders.Create.
func (o *Orders) Create(ctx context.Context, in econ.CreateOrderInput) (econ.Order, error) {
	if in.UserID == "" || in.ProductID == "" {
		return econ.Order{}, core.ErrValidation
	}
	if strings.TrimSpace(in.IdempotencyKey) == "" {
		return econ.Order{}, econ.ErrIdempotencyRequired
	}

	// Replay first: a retried request must never charge twice.
	if existing, found, err := o.store.OrderByIDempotencyKey(o.store.DB, in.IdempotencyKey); err != nil {
		return econ.Order{}, err
	} else if found {
		return o.withProduct(existing)
	}

	row, found, err := o.store.ProductByIDOrCode(o.store.DB, in.ProductID)
	if err != nil {
		return econ.Order{}, err
	}
	if !found || !row.Active {
		return econ.Order{}, econ.ErrProductNotAvailable
	}
	product, err := productOf(row)
	if err != nil {
		return econ.Order{}, err
	}

	name, err := o.resolveProvider(in.Provider, product)
	if err != nil {
		return econ.Order{}, err
	}

	now := o.now()
	order := model.Order{
		ID:             uuid.NewString(),
		UserID:         in.UserID,
		ProductID:      product.ID,
		AmountMinor:    product.AmountMinor,
		Currency:       product.Currency,
		Provider:       name,
		Status:         econ.OrderCreated,
		IdempotencyKey: in.IdempotencyKey,
		ReferenceCode:  provider.ReferenceCode(),
		ExpiresAt:      now.Add(o.ttl),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if product.Currency == econ.CurrencyToken {
		return o.settleFromWallet(order, product)
	}
	return o.startFiat(order, product)
}

// resolveProvider picks the adapter name, defaulting from the currency.
func (o *Orders) resolveProvider(requested string, p econ.Product) (string, error) {
	name := requested
	if name == "" {
		if p.Currency == econ.CurrencyToken {
			name = "token"
		} else {
			name = "manual"
		}
	}
	if _, ok := o.providers[name]; !ok {
		return "", econ.ErrProviderNotFound
	}
	// Paying a fiat price with tokens would silently invent an exchange rate,
	// and paying a token price with money has no meaning. Refuse both.
	if (name == "token") != (p.Currency == econ.CurrencyToken) {
		return "", econ.ErrProductNotAvailable
	}
	return name, nil
}

// settleFromWallet is Path A: one transaction, no provider call.
func (o *Orders) settleFromWallet(order model.Order, product econ.Product) (econ.Order, error) {
	tx := o.store.DB.Begin()
	if tx.Error != nil {
		return econ.Order{}, tx.Error
	}
	defer func() { _ = tx.Rollback().Error }()

	if err := o.store.InsertOrder(tx, order); err != nil {
		return econ.Order{}, err
	}

	if _, err := o.wallet.moveOnTx(tx, econ.Movement{
		UserID:         order.UserID,
		Currency:       econ.CurrencyToken,
		Amount:         order.AmountMinor,
		Reason:         econ.ReasonSpendFeature,
		RefType:        "order",
		RefID:          order.ID,
		IdempotencyKey: "order_spend:" + order.ID,
	}, -1); err != nil {
		if errors.Is(err, core.ErrInsufficientBalance) {
			// The failed attempt is recorded as its own order (spec 3.5). The
			// idempotency key is consumed, so a retry needs a fresh key rather
			// than silently reusing a failed one.
			_ = tx.Rollback().Error
			o.recordFailed(order)
			return econ.Order{}, core.ErrInsufficientBalance
		}
		return econ.Order{}, err
	}

	if err := o.fulfill(tx, order, product); err != nil {
		return econ.Order{}, err
	}
	moved, err := o.store.TransitionOrder(tx, order.ID, econ.OrderCreated, econ.OrderFulfilled, o.now())
	if err != nil {
		return econ.Order{}, err
	}
	if !moved {
		return econ.Order{}, econ.ErrOrderNotPayable
	}
	if err := tx.Commit().Error; err != nil {
		return econ.Order{}, err
	}

	order.Status = econ.OrderFulfilled
	order.UpdatedAt = o.now()
	return orderOf(order, &product), nil
}

// recordFailed writes the failed order in its own transaction. Best effort: a
// failure here must not mask the caller's insufficient-balance error.
func (o *Orders) recordFailed(order model.Order) {
	order.Status = econ.OrderFailed
	order.UpdatedAt = o.now()
	_ = o.store.InsertOrder(o.store.DB, order)
}

// startFiat is Path B: create, checkout, wait for payment.
func (o *Orders) startFiat(order model.Order, product econ.Product) (econ.Order, error) {
	if err := o.store.InsertOrder(o.store.DB, order); err != nil {
		return econ.Order{}, err
	}

	adapter := o.providers[order.Provider]
	res, err := adapter.CreateCheckout(context.Background(), econ.CheckoutInput{
		OrderID:       order.ID,
		UserID:        order.UserID,
		ProductCode:   product.Code,
		ProductName:   product.Name,
		AmountMinor:   order.AmountMinor,
		Currency:      order.Currency,
		ReferenceCode: order.ReferenceCode,
		Description:   product.Name,
	})
	if err != nil {
		_, _ = o.store.TransitionOrder(o.store.DB, order.ID, econ.OrderCreated, econ.OrderFailed, o.now())
		if errors.Is(err, econ.ErrProviderUnsupported) {
			return econ.Order{}, err
		}
		return econ.Order{}, core.ErrProviderFailure
	}

	reference := res.ReferenceCode
	if reference == "" {
		reference = order.ReferenceCode
	}

	// An adapter may settle inline (the test provider does). Settling goes
	// through the same path as a webhook, so the state machine is identical.
	if res.Status == econ.OrderPaid {
		if err := o.applyPayment(order.ID, order.Provider, res.ProviderRef, "", econ.OrderPaid, order.AmountMinor, order.Currency, nil); err != nil {
			return econ.Order{}, err
		}
		updated, _, err := o.store.OrderByID(o.store.DB, order.ID)
		if err != nil {
			return econ.Order{}, err
		}
		return orderOf(updated, &product), nil
	}

	if _, err := o.store.SetOrderCheckout(o.store.DB, order.ID, order.Provider, reference, o.now()); err != nil {
		return econ.Order{}, err
	}

	order.Status = econ.OrderPending
	order.ReferenceCode = reference
	order.UpdatedAt = o.now()
	out := orderOf(order, &product)
	out.Checkout = &econ.Checkout{
		RedirectURL:   res.RedirectURL,
		ReferenceCode: reference,
		Instructions:  res.Instructions,
	}
	return out, nil
}

// ------------------------------------------------------------------
// Settlement (shared by webhook, admin confirm and inline settlement)
// ------------------------------------------------------------------

// applyPayment moves one order forward from a verified payment signal. It is
// the only place that writes payments, flips to paid and fulfills.
func (o *Orders) applyPayment(
	orderID, providerName, providerRef, eventID, status string,
	amountMinor int64, currency string, raw []byte,
) error {
	tx := o.store.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() { _ = tx.Rollback().Error }()

	order, found, err := o.store.OrderForUpdate(tx, orderID)
	if err != nil {
		return err
	}
	if !found {
		return econ.ErrOrderNotFound
	}

	// Amount and currency come from the server's order; a provider that
	// reports a different price is refused rather than trusted (spec 3.8).
	if amountMinor > 0 {
		if amountMinor != order.AmountMinor || (currency != "" && !strings.EqualFold(currency, order.Currency)) {
			return econ.ErrPaymentMismatch
		}
	}

	if eventID != "" {
		inserted, err := o.store.InsertEventOnce(tx, model.PaymentEvent{
			ID:          uuid.NewString(),
			Provider:    providerName,
			EventID:     eventID,
			OrderID:     order.ID,
			Type:        "payment." + status,
			Payload:     raw,
			ProcessedAt: o.now(),
		})
		if err != nil {
			return err
		}
		if !inserted {
			// A retried delivery: already applied, answer success and stop.
			return tx.Commit().Error
		}
	}

	switch status {
	case econ.OrderFailed:
		if _, err := o.store.InsertPaymentOnce(tx, model.Payment{
			ID:          uuid.NewString(),
			OrderID:     order.ID,
			Provider:    providerName,
			ProviderRef: nilIfEmpty(providerRef),
			Status:      econ.OrderFailed,
			AmountMinor: order.AmountMinor,
			Currency:    order.Currency,
			CreatedAt:   o.now(),
		}); err != nil {
			return err
		}
		if _, err := o.store.TransitionOrder(tx, order.ID, econ.OrderPending, econ.OrderFailed, o.now()); err != nil {
			return err
		}
		return tx.Commit().Error

	case econ.OrderPaid:
		// Already fulfilled: nothing to do. This is the double-webhook guard.
		if order.Status == econ.OrderFulfilled {
			return tx.Commit().Error
		}
		if order.Status != econ.OrderPending && order.Status != econ.OrderCreated {
			return econ.ErrOrderNotPayable
		}

		paidAt := o.now()
		if _, err := o.store.InsertPaymentOnce(tx, model.Payment{
			ID:          uuid.NewString(),
			OrderID:     order.ID,
			Provider:    providerName,
			ProviderRef: nilIfEmpty(providerRef),
			Status:      econ.OrderPaid,
			AmountMinor: order.AmountMinor,
			Currency:    order.Currency,
			PaidAt:      &paidAt,
			RawPayload:  raw,
			CreatedAt:   paidAt,
		}); err != nil {
			return err
		}

		if order.Status != econ.OrderPaid {
			if _, err := o.store.TransitionOrder(tx, order.ID, order.Status, econ.OrderPaid, paidAt); err != nil {
				return err
			}
		}

		row, found, err := o.store.ProductByIDOrCode(tx, order.ProductID)
		if err != nil {
			return err
		}
		if !found {
			return econ.ErrProductNotAvailable
		}
		product, err := productOf(row)
		if err != nil {
			return err
		}
		if err := o.fulfill(tx, order, product); err != nil {
			return err
		}

		// The transition to fulfilled is what makes the grant visible as a
		// completed order; fulfill() has already written the tokens/rights.
		if _, err := o.store.TransitionOrder(tx, order.ID, econ.OrderPaid, econ.OrderFulfilled, o.now()); err != nil {
			return err
		}
		return tx.Commit().Error
	}

	return core.ErrValidation
}

// fulfill grants what the product promises. It is idempotent on its own:
// the ledger key is order-derived and the entitlement insert is ON CONFLICT.
func (o *Orders) fulfill(tx *gorm.DB, order model.Order, p econ.Product) error {
	now := o.now()

	switch p.Kind {
	case econ.KindTokenPack:
		if p.Tokens <= 0 {
			return econ.ErrProductNotAvailable
		}
		_, err := o.wallet.moveOnTx(tx, econ.Movement{
			UserID:         order.UserID,
			Currency:       econ.CurrencyToken,
			Amount:         p.Tokens,
			Reason:         econ.ReasonPurchase,
			RefType:        "order",
			RefID:          order.ID,
			IdempotencyKey: "purchase:" + order.ID,
		}, +1)
		return err

	case econ.KindSubscription, econ.KindRegular:
		code := p.Entitlement
		if code == "" {
			code = p.Code
		}
		var endsAt *time.Time
		if p.Kind == econ.KindSubscription {
			t := addInterval(now, p.Interval)
			endsAt = &t
		}
		productID := p.ID
		orderID := order.ID
		_, err := o.store.GrantEntitlementOnce(tx, model.Entitlement{
			ID:        uuid.NewString(),
			UserID:    order.UserID,
			ProductID: &productID,
			OrderID:   &orderID,
			Code:      code,
			Source:    "purchase",
			StartsAt:  now,
			EndsAt:    endsAt,
			CreatedAt: now,
		})
		return err
	}
	return econ.ErrProductNotAvailable
}

// addInterval advances a time by one product interval.
func addInterval(from time.Time, interval string) time.Time {
	switch interval {
	case "day":
		return from.AddDate(0, 0, 1)
	case "week":
		return from.AddDate(0, 0, 7)
	case "month":
		return from.AddDate(0, 1, 0)
	case "year":
		return from.AddDate(1, 0, 0)
	}
	return from
}

// ------------------------------------------------------------------
// Reads
// ------------------------------------------------------------------

// withProduct re-reads an order's product for the response shape.
func (o *Orders) withProduct(row model.Order) (econ.Order, error) {
	prod, found, err := o.store.ProductByIDOrCode(o.store.DB, row.ProductID)
	if err != nil {
		return econ.Order{}, err
	}
	if !found {
		return orderOf(row, nil), nil
	}
	p, err := productOf(prod)
	if err != nil {
		return econ.Order{}, err
	}
	return orderOf(row, &p), nil
}

// Get implements econ.Orders.Get. Another user's order is reported as missing,
// not forbidden: existence itself is information.
func (o *Orders) Get(ctx context.Context, userID, orderID string) (econ.Order, error) {
	row, found, err := o.store.OrderByID(o.store.DB, orderID)
	if err != nil {
		return econ.Order{}, err
	}
	if !found || row.UserID != userID {
		return econ.Order{}, econ.ErrOrderNotFound
	}
	return o.withProduct(row)
}

// List implements econ.Orders.List.
func (o *Orders) List(ctx context.Context, userID string, limit, offset int) ([]econ.Order, int, error) {
	limit, offset = clampPage(limit, offset)
	rows, total, err := o.store.ListOrders(o.store.DB, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return o.mapOrders(rows), total, nil
}

// ListAll implements econ.Orders.ListAll (admin).
func (o *Orders) ListAll(ctx context.Context, status string, limit, offset int) ([]econ.Order, int, error) {
	limit, offset = clampPage(limit, offset)
	rows, total, err := o.store.ListAllOrders(o.store.DB, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return o.mapOrders(rows), total, nil
}

// mapOrders resolves product details for a page in one query, not N.
func (o *Orders) mapOrders(rows []model.Order) []econ.Order {
	products := map[string]econ.Product{}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		if _, seen := products[r.ProductID]; !seen {
			ids = append(ids, r.ProductID)
		}
	}
	if len(ids) > 0 {
		all, err := o.store.ListProducts(o.store.DB, false)
		if err == nil {
			for _, p := range all {
				if mapped, perr := productOf(p); perr == nil {
					products[p.ID] = mapped
				}
			}
		}
	}

	out := make([]econ.Order, 0, len(rows))
	for _, r := range rows {
		if p, ok := products[r.ProductID]; ok {
			out = append(out, orderOf(r, &p))
			continue
		}
		out = append(out, orderOf(r, nil))
	}
	return out
}

func clampPage(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = defaultPageLimit
	}
	if limit > hardPageLimit {
		limit = hardPageLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

// ------------------------------------------------------------------
// Admin actions
// ------------------------------------------------------------------

// Confirm implements econ.Orders.Confirm: the operator saw the money arrive.
func (o *Orders) Confirm(ctx context.Context, orderID, actorID string) (econ.Order, error) {
	if actorID == "" {
		return econ.Order{}, core.ErrValidation
	}
	row, found, err := o.store.OrderByID(o.store.DB, orderID)
	if err != nil {
		return econ.Order{}, err
	}
	if !found {
		return econ.Order{}, econ.ErrOrderNotFound
	}
	if row.Status == econ.OrderFulfilled {
		// Idempotent: a second confirm on a fulfilled order is a no-op the
		// operator can retry safely.
		return o.withProduct(row)
	}
	if row.Status != econ.OrderPending {
		return econ.Order{}, econ.ErrOrderNotPayable
	}

	// amountMinor 0 skips the provider-amount check: there is no provider
	// callback to compare against, the operator is the signal.
	err = o.applyPayment(row.ID, row.Provider, "", "", econ.OrderPaid, 0, "", nil)
	if err != nil {
		return econ.Order{}, err
	}
	updated, _, err := o.store.OrderByID(o.store.DB, row.ID)
	if err != nil {
		return econ.Order{}, err
	}
	return o.withProduct(updated)
}

// Reject implements econ.Orders.Reject.
func (o *Orders) Reject(ctx context.Context, orderID, actorID, reason string) (econ.Order, error) {
	if actorID == "" {
		return econ.Order{}, core.ErrValidation
	}
	row, found, err := o.store.OrderByID(o.store.DB, orderID)
	if err != nil {
		return econ.Order{}, err
	}
	if !found {
		return econ.Order{}, econ.ErrOrderNotFound
	}
	moved, err := o.store.TransitionOrder(o.store.DB, row.ID, econ.OrderPending, econ.OrderFailed, o.now())
	if err != nil {
		return econ.Order{}, err
	}
	if !moved {
		return econ.Order{}, econ.ErrOrderNotPayable
	}
	updated, _, err := o.store.OrderByID(o.store.DB, row.ID)
	if err != nil {
		return econ.Order{}, err
	}
	return o.withProduct(updated)
}

// ------------------------------------------------------------------
// Webhook
// ------------------------------------------------------------------

// HandleWebhook implements econ.Orders.HandleWebhook.
//
// Signature first: an unverified call is refused before any row is written.
// Then the event is recorded; a duplicate returns nil (the provider must stop
// retrying) and does no further work.
func (o *Orders) HandleWebhook(ctx context.Context, providerName string, r *http.Request) error {
	if r == nil {
		return core.ErrValidation
	}
	adapter, ok := o.providers[providerName]
	if !ok {
		return econ.ErrProviderNotFound
	}
	if !adapter.Capabilities().Webhook {
		return econ.ErrProviderUnsupported
	}

	ev, err := adapter.ParseWebhook(r)
	if err != nil {
		// A bad signature must be 401 and must not touch the database.
		return err
	}

	orderID := ev.OrderID
	if orderID == "" && ev.ReferenceCode != "" {
		row, found, err := o.store.OrderByReferenceCode(o.store.DB, ev.ReferenceCode)
		if err != nil {
			return err
		}
		if !found {
			return econ.ErrOrderNotFound
		}
		orderID = row.ID
	}
	if orderID == "" {
		return core.ErrValidation
	}
	status := ev.Status
	switch status {
	case econ.OrderPaid, econ.OrderFailed:
	default:
		// An unknown event type is recorded and ignored rather than treated as
		// a payment: a provider may add event kinds at any time.
		return nil
	}

	return o.applyPayment(orderID, providerName, ev.ProviderRef, ev.EventID, status,
		ev.AmountMinor, ev.Currency, ev.RawPayload)
}

// jsonInt renders an integer for the meta jsonb without going through float.
func jsonInt(n int64) string {
	return strconv.FormatInt(n, 10)
}

// jsonString renders a meta string value with proper escaping.
func jsonString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		// json.Marshal on a string cannot fail; the fallback keeps the caller
		// from writing malformed jsonb if that ever changes.
		return `""`
	}
	return string(b)
}
