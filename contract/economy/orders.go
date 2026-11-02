package economy

import (
	"context"
	"net/http"
	"time"
)

// Product kinds (products.kind).
const (
	KindTokenPack    = "token_pack"
	KindSubscription = "subscription"
	KindRegular      = "regular"
)

// Product is one sellable catalog item.
//
// Tokens and Entitlement are lifted out of the meta jsonb so callers never
// parse it. They are only meaningful per kind: tokens for token_pack, the
// entitlement code for subscription and regular.
type Product struct {
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Kind        string    `json:"kind"`
	Currency    string    `json:"currency"`
	AmountMinor int64     `json:"amount_minor"`
	Interval    string    `json:"interval,omitempty"`
	Tokens      int64     `json:"tokens,omitempty"`
	Entitlement string    `json:"entitlement,omitempty"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Order status values (orders.status). The transitions are fixed by L2.md 3.5:
//
//	created -> pending -> paid -> fulfilled
//	                   -> failed
//	         -> expired
//	fulfilled -> refunded
//
// Every transition is a conditional UPDATE ... WHERE status = <expected>, so a
// stale writer cannot skip a step.
const (
	OrderCreated   = "created"
	OrderPending   = "pending"
	OrderPaid      = "paid"
	OrderFulfilled = "fulfilled"
	OrderFailed    = "failed"
	OrderExpired   = "expired"
	OrderRefunded  = "refunded"
)

// Checkout is the provider's answer to CreateCheckout: where to send the user
// and what reference the operator/adapter will recognise later.
type Checkout struct {
	RedirectURL   string `json:"redirect_url,omitempty"`
	ReferenceCode string `json:"reference_code,omitempty"`
	Instructions  string `json:"instructions,omitempty"`
}

// Order is one purchase attempt. AmountMinor always comes from the product on
// the server; a client-supplied amount is never read (L2.md 3.8).
type Order struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	ProductID     string    `json:"product_id"`
	ProductCode   string    `json:"product_code"`
	ProductName   string    `json:"product_name"`
	ProductKind   string    `json:"product_kind"`
	AmountMinor   int64     `json:"amount_minor"`
	Currency      string    `json:"currency"`
	Provider      string    `json:"provider"`
	Status        string    `json:"status"`
	ReferenceCode string    `json:"reference_code"`
	ExpiresAt     time.Time `json:"expires_at"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	// Checkout is populated only on creation of a fiat order.
	Checkout *Checkout `json:"checkout,omitempty"`
}

// CreateOrderInput is the client-visible part of an order request.
//
// IdempotencyKey is required: a retried request must return the original order
// rather than charge twice. Provider is optional and defaults from the product
// currency (TOKEN -> "token", fiat -> "manual").
type CreateOrderInput struct {
	UserID         string
	ProductID      string
	Provider       string
	IdempotencyKey string
}

// Entitlement is one granted right. Active means "not revoked and inside its
// window"; the module evaluates ends_at on read, so an expired subscription
// needs no job before it stops granting access.
type Entitlement struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	ProductID string     `json:"product_id,omitempty"`
	OrderID   string     `json:"order_id,omitempty"`
	Code      string     `json:"code"`
	Source    string     `json:"source"`
	StartsAt  time.Time  `json:"starts_at"`
	EndsAt    *time.Time `json:"ends_at,omitempty"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	Active    bool       `json:"active"`
	CreatedAt time.Time  `json:"created_at"`
}

// Orders is the purchase surface: catalog, the order state machine, provider
// webhooks and the admin confirm/reject actions.
type Orders interface {
	// ListProducts returns the catalog, newest first.
	ListProducts(ctx context.Context, activeOnly bool) ([]Product, error)
	// Product looks a product up by id or code.
	Product(ctx context.Context, idOrCode string) (Product, error)
	// CreateProduct and UpdateProduct are the admin catalog writes.
	CreateProduct(ctx context.Context, in ProductInput) (Product, error)
	UpdateProduct(ctx context.Context, id string, in ProductInput) (Product, error)
	// Create opens an order. A TOKEN product settles immediately inside one
	// transaction (debit + fulfilled + entitlement); a fiat product returns
	// with status pending and a Checkout. A repeated IdempotencyKey returns
	// the original order and never creates a second one.
	Create(ctx context.Context, in CreateOrderInput) (Order, error)
	// Get returns one order the caller owns; another user's order is
	// ErrOrderNotFound (404, not 403 — existence is not revealed).
	Get(ctx context.Context, userID, orderID string) (Order, error)
	// List pages the caller's own orders, newest first.
	List(ctx context.Context, userID string, limit, offset int) ([]Order, int, error)
	// ListAll is the admin view, optionally filtered by status.
	ListAll(ctx context.Context, status string, limit, offset int) ([]Order, int, error)
	// Confirm marks a pending manual order paid and fulfills it. It is the
	// operator's "the money arrived" action.
	Confirm(ctx context.Context, orderID, actorID string) (Order, error)
	// Reject closes a pending order without fulfilling it.
	Reject(ctx context.Context, orderID, actorID, reason string) (Order, error)
	// HandleWebhook verifies and applies one provider callback. It returns
	// ErrUnauthorized for a bad signature and nil for a duplicate event, so
	// the provider stops retrying.
	HandleWebhook(ctx context.Context, provider string, r *http.Request) error
}

// ProductInput is the admin catalog payload. Validation lives in the module;
// Tokens and Entitlement are written into products.meta per kind.
type ProductInput struct {
	Code        string
	Name        string
	Kind        string
	Currency    string
	AmountMinor int64
	Interval    string
	Tokens      int64
	Entitlement string
	Active      bool
}

// Entitlements is the read/manage surface for granted rights.
type Entitlements interface {
	// List returns the caller's entitlements, active ones first.
	List(ctx context.Context, userID string) ([]Entitlement, error)
	// Has answers whether the user currently holds an active entitlement.
	Has(ctx context.Context, userID, code string) (bool, error)
	// Revoke ends an entitlement now (admin/refund path).
	Revoke(ctx context.Context, id, actorID string) error
}
