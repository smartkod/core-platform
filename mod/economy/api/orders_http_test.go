package api_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"core-platform/contract/core"
	econ "core-platform/contract/economy"
)

// createProduct is the admin catalog call used by the order tests.
func createProduct(t *testing.T, e *echo.Echo, body string) string {
	t.Helper()
	rec, out := do(t, e, http.MethodPost, "/api/economy/admin/products", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create product: status %d body %s", rec.Code, rec.Body.String())
	}
	data, _ := out["data"].(map[string]any)
	id, _ := data["id"].(string)
	if id == "" {
		t.Fatalf("product without id: %v", out)
	}
	return id
}

// tokenPackBody builds an active token_pack product payload.
func tokenPackBody(amount, tokens int) string {
	return fmt.Sprintf(
		`{"code":"pack-%s","name":"Pack","kind":"token_pack","currency":"TRY","amount_minor":%d,"tokens":%d,"active":true}`,
		uuid.NewString()[:8], amount, tokens)
}

// TestManualOrderLifecycleOverHTTP proves the operator flow end to end at the
// transport layer: the price comes from the server, the admin confirms, and the
// token pack lands exactly once.
func TestManualOrderLifecycleOverHTTP(t *testing.T) {
	db := apiDB(t)
	admin := &core.Principal{UserID: uuid.NewString(), Role: core.RoleAdmin}
	adminSrv := server(t, db, admin)

	productID := createProduct(t, adminSrv, tokenPackBody(5000, 50))

	user := &core.Principal{UserID: uuid.NewString(), Role: "user"}
	userSrv := server(t, db, user)

	// The client tries to pay 1 minor unit; the server price must win.
	rec, out := do(t, userSrv, http.MethodPost, "/api/economy/orders",
		`{"product_id":"`+productID+`","amount_minor":1,"idempotency_key":"http-`+uuid.NewString()+`"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create order: status %d body %s", rec.Code, rec.Body.String())
	}
	order, _ := out["data"].(map[string]any)
	orderID, _ := order["id"].(string)
	if order["amount_minor"].(float64) != 5000 {
		t.Fatalf("client amount was used: %v", order)
	}
	if order["status"] != econ.OrderPending {
		t.Fatalf("manual order should be pending: %v", order["status"])
	}

	// Confirm is admin-only.
	rec, _ = do(t, userSrv, http.MethodPost, "/api/economy/admin/orders/"+orderID+"/confirm", "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin confirm: status %d", rec.Code)
	}

	rec, out = do(t, adminSrv, http.MethodPost, "/api/economy/admin/orders/"+orderID+"/confirm", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("admin confirm: status %d body %s", rec.Code, rec.Body.String())
	}
	confirmed, _ := out["data"].(map[string]any)
	if confirmed["status"] != econ.OrderFulfilled {
		t.Fatalf("status after confirm: %v", confirmed["status"])
	}

	// Reconfirming is harmless and must not credit twice.
	rec, _ = do(t, adminSrv, http.MethodPost, "/api/economy/admin/orders/"+orderID+"/confirm", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("replayed confirm: status %d", rec.Code)
	}

	rec, out = do(t, userSrv, http.MethodGet, "/api/economy/wallet", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("wallet: status %d", rec.Code)
	}
	wallet, _ := out["data"].(map[string]any)
	if wallet["balance"].(float64) != 50 {
		t.Fatalf("token pack balance: %v want 50", wallet["balance"])
	}

	// The catalog lists the pack, so a client can discover it.
	rec, out = do(t, userSrv, http.MethodGet, "/api/economy/products", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("catalog: status %d", rec.Code)
	}
	catalog, _ := out["data"].(map[string]any)
	if catalog["total"].(float64) < 1 {
		t.Fatalf("catalog returned no products: %v", catalog)
	}

	// A token pack grants no entitlement; the endpoint still answers.
	rec, _ = do(t, userSrv, http.MethodGet, "/api/economy/entitlements", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("entitlements: status %d", rec.Code)
	}
}

// TestRejectedOrderStaysUnfulfilled covers the operator's other ending: a
// transfer that never arrived is closed, not fulfilled, and grants nothing.
func TestRejectedOrderStaysUnfulfilled(t *testing.T) {
	db := apiDB(t)
	admin := &core.Principal{UserID: uuid.NewString(), Role: core.RoleAdmin}
	adminSrv := server(t, db, admin)

	productID := createProduct(t, adminSrv, tokenPackBody(1000, 10))
	user := &core.Principal{UserID: uuid.NewString(), Role: "user"}
	userSrv := server(t, db, user)

	rec, out := do(t, userSrv, http.MethodPost, "/api/economy/orders",
		`{"product_id":"`+productID+`","idempotency_key":"rej-`+uuid.NewString()+`"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create order: %d %s", rec.Code, rec.Body.String())
	}
	order, _ := out["data"].(map[string]any)
	orderID, _ := order["id"].(string)

	rec, out = do(t, adminSrv, http.MethodPost, "/api/economy/admin/orders/"+orderID+"/reject",
		`{"reason":"transfer never arrived"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("reject: %d %s", rec.Code, rec.Body.String())
	}
	rejected, _ := out["data"].(map[string]any)
	if rejected["status"] == econ.OrderFulfilled {
		t.Fatalf("rejected order was fulfilled: %v", rejected)
	}

	rec, out = do(t, userSrv, http.MethodGet, "/api/economy/wallet", "")
	wallet, _ := out["data"].(map[string]any)
	if wallet["balance"].(float64) != 0 {
		t.Fatalf("rejected order granted tokens: %v", wallet["balance"])
	}

	// Confirming after a reject must not resurrect the order.
	rec, out = do(t, adminSrv, http.MethodPost, "/api/economy/admin/orders/"+orderID+"/confirm", "")
	if rec.Code == http.StatusOK {
		t.Fatalf("confirm after reject succeeded: %s", rec.Body.String())
	}
}
