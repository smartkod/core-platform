// Package provider holds the payment adapters L2 ships with.
//
// L2 deliberately has no real gateway: a PayTR/Stripe adapter is a separate
// package plus one registration line, and it must not touch the wallet, the
// order state machine or entitlements (L2.md 3.6). These three prove that.
package provider

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	econ "core-platform/contract/economy"
)

// Token settles an order from the user's own token balance. It has no
// checkout page and no webhook: the order service debits the wallet and
// fulfills inside one transaction.
type Token struct{}

// Name implements econ.PaymentProvider.
func (Token) Name() string { return "token" }

// Capabilities implements econ.PaymentProvider.
func (Token) Capabilities() econ.ProviderCapabilities {
	return econ.ProviderCapabilities{SettlesImmediately: true}
}

// CreateCheckout implements econ.PaymentProvider. The token path never calls
// it; refusing loudly beats returning a fake checkout URL.
func (Token) CreateCheckout(context.Context, econ.CheckoutInput) (econ.CheckoutResult, error) {
	return econ.CheckoutResult{}, econ.ErrProviderUnsupported
}

// ParseWebhook implements econ.PaymentProvider.
func (Token) ParseWebhook(*http.Request) (econ.ProviderEvent, error) {
	return econ.ProviderEvent{}, econ.ErrProviderUnsupported
}

// Refund implements econ.PaymentProvider. Refunding a token purchase is a
// wallet credit performed by the refund service, not a provider call.
func (Token) Refund(context.Context, econ.RefundInput) (econ.RefundResult, error) {
	return econ.RefundResult{}, econ.ErrProviderUnsupported
}

// Manual is the bank-transfer adapter: the operator reads the reference from
// the bank statement and confirms the order in the admin panel.
type Manual struct {
	// Instructions is shown to the user on the checkout screen. Configured,
	// not hard-coded, because the account details differ per deployment.
	Instructions string
}

// Name implements econ.PaymentProvider.
func (Manual) Name() string { return "manual" }

// Capabilities implements econ.PaymentProvider.
func (Manual) Capabilities() econ.ProviderCapabilities {
	return econ.ProviderCapabilities{}
}

// CreateCheckout implements econ.PaymentProvider. It returns a reference code
// and payment instructions; the money moves outside the system.
func (m Manual) CreateCheckout(_ context.Context, in econ.CheckoutInput) (econ.CheckoutResult, error) {
	return econ.CheckoutResult{
		ReferenceCode: in.ReferenceCode,
		Instructions:  m.Instructions,
		Status:        econ.OrderPending,
	}, nil
}

// ParseWebhook implements econ.PaymentProvider: a manual transfer has no
// callback. Confirmation is the admin action.
func (Manual) ParseWebhook(*http.Request) (econ.ProviderEvent, error) {
	return econ.ProviderEvent{}, econ.ErrProviderUnsupported
}

// Refund implements econ.PaymentProvider.
func (Manual) Refund(context.Context, econ.RefundInput) (econ.RefundResult, error) {
	return econ.RefundResult{}, econ.ErrProviderUnsupported
}

// Test is the development adapter. It exists to prove the extensibility claim
// end to end: a webhook with a real HMAC check, no gateway account, and a
// registration row. It must never be registered in production (the module
// refuses to).
type Test struct {
	// Secret signs webhook bodies: hex(HMAC-SHA256(rawBody, Secret)).
	Secret string
}

// Name implements econ.PaymentProvider.
func (Test) Name() string { return "test" }

// Capabilities implements econ.PaymentProvider.
func (Test) Capabilities() econ.ProviderCapabilities {
	return econ.ProviderCapabilities{Webhook: true, Refund: true}
}

// CreateCheckout implements econ.PaymentProvider. It hands back a redirect the
// caller can ignore; the order stays pending until a signed webhook arrives,
// so the intermediate state is genuinely exercised.
func (Test) CreateCheckout(_ context.Context, in econ.CheckoutInput) (econ.CheckoutResult, error) {
	return econ.CheckoutResult{
		ProviderRef:   "test-" + in.OrderID,
		ReferenceCode: in.ReferenceCode,
		RedirectURL:   "https://test.invalid/pay/" + in.OrderID,
		Status:        econ.OrderPending,
	}, nil
}

// ParseWebhook implements econ.PaymentProvider.
//
// The signature is verified before the body is parsed: a forged call must not
// reach the JSON decoder at all.
func (t Test) ParseWebhook(r *http.Request) (econ.ProviderEvent, error) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return econ.ProviderEvent{}, econ.ErrSignatureInvalid
	}

	got := strings.TrimSpace(r.Header.Get("X-Test-Signature"))
	if t.Secret == "" || got == "" || !validSignature(t.Secret, body, got) {
		return econ.ProviderEvent{}, econ.ErrSignatureInvalid
	}

	var in struct {
		EventID     string `json:"event_id"`
		OrderID     string `json:"order_id"`
		Reference   string `json:"reference_code"`
		Status      string `json:"status"`
		AmountMinor int64  `json:"amount_minor"`
		Currency    string `json:"currency"`
	}
	if err := json.Unmarshal(body, &in); err != nil {
		return econ.ProviderEvent{}, econ.ErrSignatureInvalid
	}
	if in.EventID == "" {
		return econ.ProviderEvent{}, econ.ErrSignatureInvalid
	}

	return econ.ProviderEvent{
		EventID:       in.EventID,
		Type:          "payment." + in.Status,
		OrderID:       in.OrderID,
		ReferenceCode: in.Reference,
		Status:        in.Status,
		AmountMinor:   in.AmountMinor,
		Currency:      strings.ToUpper(in.Currency),
		RawPayload:    body,
	}, nil
}

// Refund implements econ.PaymentProvider.
func (Test) Refund(_ context.Context, in econ.RefundInput) (econ.RefundResult, error) {
	return econ.RefundResult{ProviderRef: "test-refund-" + in.PaymentID, Status: "done"}, nil
}

// validSignature compares in constant time, so a wrong signature cannot be
// guessed byte by byte.
func validSignature(secret string, body []byte, got string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	want := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(want), []byte(got))
}

// ReferenceCode builds the human-quotable order reference.
//
// It is random rather than sequential so a customer cannot infer sales volume
// from their own order id, and long enough that guessing another live order is
// impractical.
func ReferenceCode() string {
	var b [10]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand failure is fatal for a payment reference; continuing
		// with a predictable code would be worse than stopping.
		panic("economy: crypto/rand failed: " + err.Error())
	}
	return "ORD-" + hex.EncodeToString(b[:])
}
