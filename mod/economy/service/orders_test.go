package service_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"core-platform/contract/core"
	econ "core-platform/contract/economy"
	"core-platform/mod/economy/provider"
	"core-platform/mod/economy/repository"
	"core-platform/mod/economy/service"
)

// shop bundles the order-side services for one isolated user.
type shop struct {
	orders *service.Orders
	wallet *service.Wallet
	ents   *service.Entitlements
	userID string
	admin  string
}

func newShop(t *testing.T, db *gorm.DB, clock *fakeClock) *shop {
	t.Helper()
	store := repository.NewStore(db)
	wallet := service.NewWallet(store, nil, clock)
	providers := map[string]econ.PaymentProvider{
		"token":  provider.Token{},
		"manual": provider.Manual{Instructions: "IBAN TR00 test"},
		"test":   provider.Test{Secret: "secret"},
	}
	orders := service.NewOrders(store, wallet, providers, clock.Now, time.Hour)
	ents := service.NewEntitlements(store, clock.Now)
	return &shop{
		orders: orders,
		wallet: wallet,
		ents:   ents,
		userID: uuidStr(),
		admin:  uuidStr(),
	}
}

// product creates a catalog row through the admin path and drops it after the
// test so a shared database does not accumulate products.
func (s *shop) product(t *testing.T, db *gorm.DB, in econ.ProductInput) econ.Product {
	t.Helper()
	if in.Code == "" {
		in.Code = "p-" + uuidStr()[:8]
	}
	if in.Name == "" {
		in.Name = "Test " + in.Code
	}
	p, err := s.orders.CreateProduct(context.Background(), in)
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM products WHERE id = ?`, p.ID) })
	return p
}

func TestTokenProductSpendsFromWallet(t *testing.T) {
	db := economyDB(t)
	clock := newClock()
	setDial(t, db, 0, 1, 10, 100, true)
	s := newShop(t, db, clock)
	ctx := context.Background()

	// 100 tokens to spend.
	if _, err := s.wallet.Credit(ctx, econ.Movement{
		UserID: s.userID, Currency: econ.CurrencyToken, Amount: 100,
		Reason: econ.ReasonAdminCredit,
	}); err != nil {
		t.Fatal(err)
	}

	p := s.product(t, db, econ.ProductInput{
		Kind: econ.KindRegular, Currency: econ.CurrencyToken,
		AmountMinor: 30, Entitlement: "premium.ads_off", Active: true,
	})

	order, err := s.orders.Create(ctx, econ.CreateOrderInput{
		UserID: s.userID, ProductID: p.ID, IdempotencyKey: "tok-" + uuidStr(),
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if order.Status != econ.OrderFulfilled {
		t.Fatalf("token order should settle immediately: %s", order.Status)
	}

	bal, _ := s.wallet.Balance(ctx, s.userID, econ.CurrencyToken)
	if bal.Balance != 70 {
		t.Fatalf("balance after spend: got %d want 70", bal.Balance)
	}

	active, err := s.ents.Has(ctx, s.userID, "premium.ads_off")
	if err != nil {
		t.Fatal(err)
	}
	if !active {
		t.Fatal("entitlement was not granted by the token purchase")
	}
}

func TestTokenProductInsufficientBalanceLeavesLedgerClean(t *testing.T) {
	db := economyDB(t)
	clock := newClock()
	setDial(t, db, 0, 1, 10, 100, true)
	s := newShop(t, db, clock)
	ctx := context.Background()

	if _, err := s.wallet.Credit(ctx, econ.Movement{
		UserID: s.userID, Currency: econ.CurrencyToken, Amount: 50,
		Reason: econ.ReasonAdminCredit,
	}); err != nil {
		t.Fatal(err)
	}
	p := s.product(t, db, econ.ProductInput{
		Kind: econ.KindRegular, Currency: econ.CurrencyToken,
		AmountMinor: 100, Entitlement: "expensive", Active: true,
	})

	_, err := s.orders.Create(ctx, econ.CreateOrderInput{
		UserID: s.userID, ProductID: p.ID, IdempotencyKey: "poor-" + uuidStr(),
	})
	if !errors.Is(err, core.ErrInsufficientBalance) {
		t.Fatalf("want insufficient balance, got %v", err)
	}

	bal, _ := s.wallet.Balance(ctx, s.userID, econ.CurrencyToken)
	if bal.Balance != 50 {
		t.Fatalf("failed purchase moved the balance: %d", bal.Balance)
	}
	active, _ := s.ents.Has(ctx, s.userID, "expensive")
	if active {
		t.Fatal("a failed purchase granted an entitlement")
	}

	// The attempt itself is recorded as a failed order (spec 3.5).
	var status string
	db.Raw(`SELECT status FROM orders WHERE user_id = ? AND product_id = ?`, s.userID, p.ID).Scan(&status)
	if status != econ.OrderFailed {
		t.Fatalf("failed attempt status: got %q want failed", status)
	}
}

func TestOrderIdempotencyKeyReturnsSameOrder(t *testing.T) {
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
		AmountMinor: 10, Entitlement: "idem.right", Active: true,
	})

	key := "idem-" + uuidStr()
	a, err := s.orders.Create(ctx, econ.CreateOrderInput{UserID: s.userID, ProductID: p.ID, IdempotencyKey: key})
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.orders.Create(ctx, econ.CreateOrderInput{UserID: s.userID, ProductID: p.ID, IdempotencyKey: key})
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if a.ID != b.ID {
		t.Fatalf("replay created a second order: %s vs %s", a.ID, b.ID)
	}
	bal, _ := s.wallet.Balance(ctx, s.userID, econ.CurrencyToken)
	if bal.Balance != 90 {
		t.Fatalf("replay charged twice: %d", bal.Balance)
	}
}

func TestMissingIdempotencyKeyRejected(t *testing.T) {
	db := economyDB(t)
	s := newShop(t, db, newClock())
	p := s.product(t, db, econ.ProductInput{
		Kind: econ.KindRegular, Currency: econ.CurrencyToken,
		AmountMinor: 5, Entitlement: "x", Active: true,
	})
	_, err := s.orders.Create(context.Background(), econ.CreateOrderInput{
		UserID: s.userID, ProductID: p.ID,
	})
	if !errors.Is(err, econ.ErrIdempotencyRequired) {
		t.Fatalf("want idempotency required, got %v", err)
	}
}

func TestClientAmountIsIgnored(t *testing.T) {
	db := economyDB(t)
	setDial(t, db, 0, 1, 10, 100, true)
	s := newShop(t, db, newClock())
	ctx := context.Background()

	p := s.product(t, db, econ.ProductInput{
		Kind: econ.KindTokenPack, Currency: econ.CurrencyTRY,
		AmountMinor: 5000, Tokens: 50, Active: true,
	})

	// The only amount the input can carry is in the product price; there is no
	// amount field at all, so the server price is structurally the only one.
	order, err := s.orders.Create(ctx, econ.CreateOrderInput{
		UserID: s.userID, ProductID: p.Code, IdempotencyKey: "price-" + uuidStr(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if order.AmountMinor != 5000 || order.Currency != econ.CurrencyTRY {
		t.Fatalf("order did not use the server price: %+v", order)
	}
}

func TestOtherUsersOrderIsNotFound(t *testing.T) {
	db := economyDB(t)
	setDial(t, db, 0, 1, 10, 100, true)
	s := newShop(t, db, newClock())
	ctx := context.Background()

	if _, err := s.wallet.Credit(ctx, econ.Movement{
		UserID: s.userID, Currency: econ.CurrencyToken, Amount: 100,
		Reason: econ.ReasonAdminCredit,
	}); err != nil {
		t.Fatal(err)
	}
	p := s.product(t, db, econ.ProductInput{
		Kind: econ.KindRegular, Currency: econ.CurrencyToken,
		AmountMinor: 5, Entitlement: "private", Active: true,
	})
	order, err := s.orders.Create(ctx, econ.CreateOrderInput{
		UserID: s.userID, ProductID: p.ID, IdempotencyKey: "own-" + uuidStr(),
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := s.orders.Get(ctx, uuidStr(), order.ID); !errors.Is(err, econ.ErrOrderNotFound) {
		t.Fatalf("another user's order must read as not found, got %v", err)
	}
}

func TestManualOrderConfirmFulfillsOnce(t *testing.T) {
	db := economyDB(t)
	clock := newClock()
	setDial(t, db, 0, 1, 10, 100, true)
	s := newShop(t, db, clock)
	ctx := context.Background()

	p := s.product(t, db, econ.ProductInput{
		Kind: econ.KindTokenPack, Currency: econ.CurrencyTRY,
		AmountMinor: 5000, Tokens: 50, Active: true,
	})
	order, err := s.orders.Create(ctx, econ.CreateOrderInput{
		UserID: s.userID, ProductID: p.ID, IdempotencyKey: "manual-" + uuidStr(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if order.Status != econ.OrderPending {
		t.Fatalf("fiat order should be pending: %s", order.Status)
	}
	if order.Checkout == nil || order.Checkout.ReferenceCode == "" {
		t.Fatalf("checkout missing reference: %+v", order.Checkout)
	}

	// Before confirmation: nothing granted.
	bal, _ := s.wallet.Balance(ctx, s.userID, econ.CurrencyToken)
	if bal.Balance != 0 {
		t.Fatalf("pending order granted tokens: %d", bal.Balance)
	}

	confirmed, err := s.orders.Confirm(ctx, order.ID, s.admin)
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if confirmed.Status != econ.OrderFulfilled {
		t.Fatalf("status after confirm: %s", confirmed.Status)
	}
	bal, _ = s.wallet.Balance(ctx, s.userID, econ.CurrencyToken)
	if bal.Balance != 50 {
		t.Fatalf("token pack granted %d tokens, want 50", bal.Balance)
	}

	// A second confirm is a no-op, not a second grant.
	if _, err := s.orders.Confirm(ctx, order.ID, s.admin); err != nil {
		t.Fatalf("replayed confirm: %v", err)
	}
	bal, _ = s.wallet.Balance(ctx, s.userID, econ.CurrencyToken)
	if bal.Balance != 50 {
		t.Fatalf("replayed confirm granted again: %d", bal.Balance)
	}

	var purchases int64
	db.Raw(`SELECT COUNT(*) FROM wallet_entries e
	          JOIN wallet_accounts a ON a.id = e.account_id
	         WHERE a.user_id = ? AND e.reason_code = ?`,
		s.userID, econ.ReasonPurchase).Scan(&purchases)
	if purchases != 1 {
		t.Fatalf("purchase movements: %d want 1", purchases)
	}
}

func TestWebhookSignatureRequired(t *testing.T) {
	db := economyDB(t)
	setDial(t, db, 0, 1, 10, 100, true)
	s := newShop(t, db, newClock())
	ctx := context.Background()

	p := s.product(t, db, econ.ProductInput{
		Kind: econ.KindTokenPack, Currency: econ.CurrencyTRY,
		AmountMinor: 1000, Tokens: 10, Active: true,
	})
	order, err := s.orders.Create(ctx, econ.CreateOrderInput{
		UserID: s.userID, ProductID: p.ID, Provider: "test", IdempotencyKey: "wh-" + uuidStr(),
	})
	if err != nil {
		t.Fatal(err)
	}

	body := `{"event_id":"e1","order_id":"` + order.ID + `","status":"paid","amount_minor":1000,"currency":"TRY"}`
	req := httptest.NewRequest(http.MethodPost, "/webhook/test", bytes.NewReader([]byte(body)))
	// No signature header.
	if err := s.orders.HandleWebhook(ctx, "test", req); !errors.Is(err, econ.ErrSignatureInvalid) {
		t.Fatalf("unsigned webhook: want signature invalid, got %v", err)
	}

	// A wrong signature is equally refused and writes nothing.
	req = signedRequest(t, "wrong-secret", body)
	if err := s.orders.HandleWebhook(ctx, "test", req); !errors.Is(err, econ.ErrSignatureInvalid) {
		t.Fatalf("bad signature: want signature invalid, got %v", err)
	}
	var events int64
	db.Raw(`SELECT COUNT(*) FROM payment_events WHERE order_id = ?`, order.ID).Scan(&events)
	if events != 0 {
		t.Fatalf("a rejected webhook wrote %d events", events)
	}
}

func TestWebhookFulfillsAndDeduplicates(t *testing.T) {
	db := economyDB(t)
	setDial(t, db, 0, 1, 10, 100, true)
	s := newShop(t, db, newClock())
	ctx := context.Background()

	p := s.product(t, db, econ.ProductInput{
		Kind: econ.KindTokenPack, Currency: econ.CurrencyTRY,
		AmountMinor: 1000, Tokens: 10, Active: true,
	})
	order, err := s.orders.Create(ctx, econ.CreateOrderInput{
		UserID: s.userID, ProductID: p.ID, Provider: "test", IdempotencyKey: "wh2-" + uuidStr(),
	})
	if err != nil {
		t.Fatal(err)
	}

	body := `{"event_id":"evt-` + order.ID + `","order_id":"` + order.ID +
		`","status":"paid","amount_minor":1000,"currency":"TRY"}`

	if err := s.orders.HandleWebhook(ctx, "test", signedRequest(t, "secret", body)); err != nil {
		t.Fatalf("first webhook: %v", err)
	}
	bal, _ := s.wallet.Balance(ctx, s.userID, econ.CurrencyToken)
	if bal.Balance != 10 {
		t.Fatalf("tokens after webhook: %d want 10", bal.Balance)
	}

	// The same event id again: accepted, but no second grant.
	if err := s.orders.HandleWebhook(ctx, "test", signedRequest(t, "secret", body)); err != nil {
		t.Fatalf("duplicate webhook: %v", err)
	}
	bal, _ = s.wallet.Balance(ctx, s.userID, econ.CurrencyToken)
	if bal.Balance != 10 {
		t.Fatalf("duplicate webhook granted again: %d", bal.Balance)
	}
}

func TestWebhookAmountMismatchRefused(t *testing.T) {
	db := economyDB(t)
	setDial(t, db, 0, 1, 10, 100, true)
	s := newShop(t, db, newClock())
	ctx := context.Background()

	p := s.product(t, db, econ.ProductInput{
		Kind: econ.KindTokenPack, Currency: econ.CurrencyTRY,
		AmountMinor: 1000, Tokens: 10, Active: true,
	})
	order, err := s.orders.Create(ctx, econ.CreateOrderInput{
		UserID: s.userID, ProductID: p.ID, Provider: "test", IdempotencyKey: "wh3-" + uuidStr(),
	})
	if err != nil {
		t.Fatal(err)
	}

	// A provider reporting a cheaper price must not be believed.
	body := `{"event_id":"cheap-` + order.ID + `","order_id":"` + order.ID +
		`","status":"paid","amount_minor":1,"currency":"TRY"}`
	if err := s.orders.HandleWebhook(ctx, "test", signedRequest(t, "secret", body)); !errors.Is(err, econ.ErrPaymentMismatch) {
		t.Fatalf("mismatched amount: want payment mismatch, got %v", err)
	}
	bal, _ := s.wallet.Balance(ctx, s.userID, econ.CurrencyToken)
	if bal.Balance != 0 {
		t.Fatalf("mismatched payment granted tokens: %d", bal.Balance)
	}
}

func TestSubscriptionGrantsWindowedEntitlement(t *testing.T) {
	db := economyDB(t)
	clock := newClock()
	setDial(t, db, 0, 1, 10, 100, true)
	s := newShop(t, db, clock)
	ctx := context.Background()

	p := s.product(t, db, econ.ProductInput{
		Kind: econ.KindSubscription, Currency: econ.CurrencyTRY,
		AmountMinor: 9900, Interval: "month", Entitlement: "premium.vip", Active: true,
	})
	order, err := s.orders.Create(ctx, econ.CreateOrderInput{
		UserID: s.userID, ProductID: p.ID, IdempotencyKey: "sub-" + uuidStr(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.orders.Confirm(ctx, order.ID, s.admin); err != nil {
		t.Fatalf("confirm subscription: %v", err)
	}

	active, err := s.ents.Has(ctx, s.userID, "premium.vip")
	if err != nil {
		t.Fatal(err)
	}
	if !active {
		t.Fatal("subscription entitlement is not active")
	}

	// One month later the window has closed; no job was needed for that.
	clock.Advance(32 * 24 * time.Hour)
	active, err = s.ents.Has(ctx, s.userID, "premium.vip")
	if err != nil {
		t.Fatal(err)
	}
	if active {
		t.Fatal("subscription entitlement still active after its window")
	}

	// The row is kept, not deleted: history matters for support.
	list, err := s.ents.List(ctx, s.userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("entitlement rows: %d want 1", len(list))
	}
}

func TestRevokedEntitlementStopsGrantingAccess(t *testing.T) {
	db := economyDB(t)
	setDial(t, db, 0, 1, 10, 100, true)
	s := newShop(t, db, newClock())
	ctx := context.Background()

	p := s.product(t, db, econ.ProductInput{
		Kind: econ.KindRegular, Currency: econ.CurrencyTRY,
		AmountMinor: 500, Entitlement: "revoke.me", Active: true,
	})
	order, err := s.orders.Create(ctx, econ.CreateOrderInput{
		UserID: s.userID, ProductID: p.ID, IdempotencyKey: "rev-" + uuidStr(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.orders.Confirm(ctx, order.ID, s.admin); err != nil {
		t.Fatal(err)
	}
	list, _ := s.ents.List(ctx, s.userID)
	if len(list) != 1 {
		t.Fatalf("entitlements: %d", len(list))
	}
	if err := s.ents.Revoke(ctx, list[0].ID, s.admin); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	active, _ := s.ents.Has(ctx, s.userID, "revoke.me")
	if active {
		t.Fatal("revoked entitlement still grants access")
	}
	// Revoking twice is not a success: there is nothing left to revoke.
	if err := s.ents.Revoke(ctx, list[0].ID, s.admin); !errors.Is(err, econ.ErrEntitlementNotFound) {
		t.Fatalf("second revoke: want not found, got %v", err)
	}
}

func TestProviderCurrencyMismatchRefused(t *testing.T) {
	db := economyDB(t)
	setDial(t, db, 0, 1, 10, 100, true)
	s := newShop(t, db, newClock())
	ctx := context.Background()

	// A token-priced product could be "paid" with real money for a made-up
	// exchange rate; the service must refuse the pairing.
	tokenProduct := s.product(t, db, econ.ProductInput{
		Kind: econ.KindRegular, Currency: econ.CurrencyToken,
		AmountMinor: 5, Entitlement: "wrong.way", Active: true,
	})
	if _, err := s.orders.Create(ctx, econ.CreateOrderInput{
		UserID: s.userID, ProductID: tokenProduct.ID, Provider: "manual",
		IdempotencyKey: "mismatch-" + uuidStr(),
	}); !errors.Is(err, econ.ErrProductNotAvailable) {
		t.Fatalf("token product via fiat provider: want product not available, got %v", err)
	}
}

func TestWebhookFailedClosesOrderWithoutGrant(t *testing.T) {
	db := economyDB(t)
	setDial(t, db, 0, 1, 10, 100, true)
	s := newShop(t, db, newClock())
	ctx := context.Background()

	p := s.product(t, db, econ.ProductInput{
		Kind: econ.KindTokenPack, Currency: econ.CurrencyTRY,
		AmountMinor: 1000, Tokens: 10, Active: true,
	})
	order, err := s.orders.Create(ctx, econ.CreateOrderInput{
		UserID: s.userID, ProductID: p.ID, Provider: "test", IdempotencyKey: "fail-" + uuidStr(),
	})
	if err != nil {
		t.Fatal(err)
	}

	body := `{"event_id":"fail-` + order.ID + `","order_id":"` + order.ID +
		`","status":"failed","amount_minor":1000,"currency":"TRY"}`
	if err := s.orders.HandleWebhook(ctx, "test", signedRequest(t, "secret", body)); err != nil {
		t.Fatalf("failed webhook: %v", err)
	}

	updated, err := s.orders.Get(ctx, s.userID, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != econ.OrderFailed {
		t.Fatalf("status after failed payment: %s want failed", updated.Status)
	}
	bal, _ := s.wallet.Balance(ctx, s.userID, econ.CurrencyToken)
	if bal.Balance != 0 {
		t.Fatalf("failed payment granted %d tokens", bal.Balance)
	}

	// A later "paid" event for a closed order must not resurrect it.
	paid := `{"event_id":"late-` + order.ID + `","order_id":"` + order.ID +
		`","status":"paid","amount_minor":1000,"currency":"TRY"}`
	if err := s.orders.HandleWebhook(ctx, "test", signedRequest(t, "secret", paid)); !errors.Is(err, econ.ErrOrderNotPayable) {
		t.Fatalf("paid after failure: want order not payable, got %v", err)
	}
}

func TestExpiredOrderIsNotPayable(t *testing.T) {
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
		UserID: s.userID, ProductID: p.ID, Provider: "test", IdempotencyKey: "exp-" + uuidStr(),
	})
	if err != nil {
		t.Fatal(err)
	}

	// The expiry sweep closes it; the clock is injected, so no sleeping.
	clock.Advance(2 * time.Hour)
	store := repository.NewStore(db)
	if _, err := store.ExpireOverdue(db, clock.Now()); err != nil {
		t.Fatalf("expire: %v", err)
	}

	body := `{"event_id":"exp-` + order.ID + `","order_id":"` + order.ID +
		`","status":"paid","amount_minor":1000,"currency":"TRY"}`
	if err := s.orders.HandleWebhook(ctx, "test", signedRequest(t, "secret", body)); !errors.Is(err, econ.ErrOrderNotPayable) {
		t.Fatalf("paid after expiry: want order not payable, got %v", err)
	}
	bal, _ := s.wallet.Balance(ctx, s.userID, econ.CurrencyToken)
	if bal.Balance != 0 {
		t.Fatalf("expired order granted %d tokens", bal.Balance)
	}
}

// signedRequest builds a test-provider callback with a valid HMAC.
func signedRequest(t *testing.T, secret, body string) *http.Request {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	sig := hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/webhook/test", bytes.NewReader([]byte(body)))
	req.Header.Set("X-Test-Signature", sig)
	return req
}

// uuidStr is the local uuid helper for this file.
func uuidStr() string { return uuid.NewString() }
