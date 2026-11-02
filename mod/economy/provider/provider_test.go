package provider_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	econ "core-platform/contract/economy"
	"core-platform/mod/economy/provider"
)

// sign builds the header the test adapter verifies.
func sign(secret, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestAdapterVerifiesSignatureBeforeParsing(t *testing.T) {
	const secret = "s3cret"
	body := `{"event_id":"evt-1","order_id":"11111111-1111-1111-1111-111111111111",` +
		`"status":"paid","amount_minor":500,"currency":"try"}`
	adapter := provider.Test{Secret: secret}

	req := httptest.NewRequest(http.MethodPost, "/webhook/test", strings.NewReader(body))
	req.Header.Set("X-Test-Signature", sign(secret, body))

	ev, err := adapter.ParseWebhook(req)
	if err != nil {
		t.Fatalf("valid signature refused: %v", err)
	}
	if ev.EventID != "evt-1" || ev.Status != econ.OrderPaid {
		t.Fatalf("event not normalised: %+v", ev)
	}
	if ev.Currency != "TRY" {
		t.Fatalf("currency not upper-cased: %q", ev.Currency)
	}
	if ev.AmountMinor != 500 {
		t.Fatalf("amount: %d want 500", ev.AmountMinor)
	}
	if len(ev.RawPayload) == 0 {
		t.Fatal("raw payload was not kept for the audit row")
	}
}

func TestAdapterRejectsBadSignature(t *testing.T) {
	const secret = "s3cret"
	body := `{"event_id":"evt-1","status":"paid"}`
	adapter := provider.Test{Secret: secret}

	missing := httptest.NewRequest(http.MethodPost, "/webhook/test", strings.NewReader(body))

	wrong := httptest.NewRequest(http.MethodPost, "/webhook/test", strings.NewReader(body))
	wrong.Header.Set("X-Test-Signature", sign("other", body))

	tampered := httptest.NewRequest(http.MethodPost, "/webhook/test", strings.NewReader(body+" "))
	tampered.Header.Set("X-Test-Signature", sign(secret, body))

	for name, req := range map[string]*http.Request{
		"missing": missing, "wrong": wrong, "tampered": tampered,
	} {
		if _, err := adapter.ParseWebhook(req); err == nil {
			t.Fatalf("%s signature accepted", name)
		}
	}

	// An empty secret disables the adapter: it must never accept anything, even
	// when the caller signs with an empty key.
	empty := httptest.NewRequest(http.MethodPost, "/webhook/test", strings.NewReader(body))
	empty.Header.Set("X-Test-Signature", sign("", body))
	if _, err := (provider.Test{}).ParseWebhook(empty); err == nil {
		t.Fatal("adapter without a secret accepted a webhook")
	}
}

func TestAdapterRequiresEventID(t *testing.T) {
	const secret = "s3cret"
	body := `{"order_id":"11111111-1111-1111-1111-111111111111","status":"paid"}`
	req := httptest.NewRequest(http.MethodPost, "/webhook/test", bytes.NewReader([]byte(body)))
	req.Header.Set("X-Test-Signature", sign(secret, body))

	if _, err := (provider.Test{Secret: secret}).ParseWebhook(req); err == nil {
		t.Fatal("event without an id was accepted; deduplication would be impossible")
	}
}

func TestUnsupportedOperationsAreExplicit(t *testing.T) {
	ctx := context.Background()
	in := econ.CheckoutInput{OrderID: "o", AmountMinor: 100, Currency: "TRY"}

	if _, err := (provider.Token{}).CreateCheckout(ctx, in); err != econ.ErrProviderUnsupported {
		t.Fatalf("token checkout: %v", err)
	}
	if _, err := (provider.Token{}).Refund(ctx, econ.RefundInput{}); err != econ.ErrProviderUnsupported {
		t.Fatalf("token refund: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	if _, err := (provider.Manual{}).ParseWebhook(req); err != econ.ErrProviderUnsupported {
		t.Fatalf("manual webhook: %v", err)
	}

	// A manual checkout is a reference plus instructions, never a redirect.
	res, err := provider.Manual{Instructions: "IBAN TR00"}.
		CreateCheckout(ctx, econ.CheckoutInput{ReferenceCode: "ORD-1"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != econ.OrderPending || res.ReferenceCode != "ORD-1" || res.Instructions != "IBAN TR00" {
		t.Fatalf("manual checkout: %+v", res)
	}
	if res.RedirectURL != "" {
		t.Fatalf("manual checkout invented a redirect: %q", res.RedirectURL)
	}
}

func TestReferenceCodesAreUniqueAndPrefixed(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		code := provider.ReferenceCode()
		if !strings.HasPrefix(code, "ORD-") {
			t.Fatalf("reference without prefix: %q", code)
		}
		if seen[code] {
			t.Fatalf("duplicate reference: %q", code)
		}
		seen[code] = true
	}
}
