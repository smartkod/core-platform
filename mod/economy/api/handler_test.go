package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"core-platform/contract/core"
	econ "core-platform/contract/economy"
	"core-platform/mod/economy/api"
	"core-platform/mod/economy/migrations"
	"core-platform/mod/economy/provider"
	"core-platform/mod/economy/repository"
	"core-platform/mod/economy/service"
	"core-platform/platform/httpx"
	"core-platform/platform/migrate"
)

// noopLogger keeps 5xx paths from panicking in tests.
type noopLogger struct{}

func (noopLogger) Info(string, ...any)     {}
func (noopLogger) Warn(string, ...any)     {}
func (noopLogger) Error(string, ...any)    {}
func (noopLogger) With(...any) core.Logger { return noopLogger{} }

// stubIdentity injects a fixed principal, so the transport rules can be tested
// without the identity module or a real token.
type stubIdentity struct{ p *core.Principal }

func (s stubIdentity) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if s.p != nil {
				c.Set(core.ContextUserKey, s.p)
			}
			return next(c)
		}
	}
}

func (s stubIdentity) RequireRole(allowed ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			p := core.UserFrom(c)
			if p == nil {
				return core.ErrUnauthorized
			}
			for _, r := range allowed {
				if p.Role == r {
					return next(c)
				}
			}
			return core.ErrForbidden
		}
	}
}

func (s stubIdentity) CurrentUser(c echo.Context) (*core.Principal, error) {
	if p := core.UserFrom(c); p != nil {
		return p, nil
	}
	return nil, core.ErrUnauthorized
}

// apiDB mirrors the service test setup: skip without a DSN.
func apiDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set; skipping database-backed economy api tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                 gormlogger.Default.LogMode(gormlogger.Silent),
		SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := migrate.RunSource(context.Background(), db, migrations.Source()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// Deterministic dials: no welcome grant, generous caps.
	if err := db.Exec(
		`INSERT INTO reward_settings (id, signup_grant, ad_watch_reward, daily_reward_cap,
		     monthly_reward_cap, click_reward_enabled, updated_at)
		 VALUES (1, 0, 1, 10, 100, true, now())
		 ON CONFLICT (id) DO UPDATE SET signup_grant = 0`,
	).Error; err != nil {
		t.Fatalf("seed settings: %v", err)
	}
	return db
}

// server mounts the economy routes over real services and the stub identity.
func server(t *testing.T, db *gorm.DB, p *core.Principal) *echo.Echo {
	t.Helper()
	store := repository.NewStore(db)
	wallet := service.NewWallet(store, nil, nil)
	ads := service.NewAds(wallet, store, nil)
	providers := map[string]econ.PaymentProvider{
		"token":  provider.Token{},
		"manual": provider.Manual{Instructions: "IBAN TR00 test"},
		"test":   provider.Test{Secret: testWebhookSecret},
	}
	orders := service.NewOrders(store, wallet, providers, nil, time.Hour)
	ents := service.NewEntitlements(store, nil)
	h := api.NewHandler(api.Deps{Wallet: wallet, Ads: ads, Orders: orders, Entitlements: ents})

	e := echo.New()
	e.HTTPErrorHandler = httpx.ErrorHandler(noopLogger{})
	h.Register(e.Group("/api"), stubIdentity{p: p})
	return e
}

// testWebhookSecret signs test-provider callbacks in these tests.
const testWebhookSecret = "test-webhook-secret"

func do(t *testing.T, e *echo.Echo, method, path, body string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	var rdr *strings.Reader
	if body == "" {
		rdr = strings.NewReader("")
	} else {
		rdr = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rdr)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec, out
}

func TestRoutesRequireAuthentication(t *testing.T) {
	db := apiDB(t)
	e := server(t, db, nil)

	rec, out := do(t, e, http.MethodGet, "/api/economy/wallet", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wallet without auth: status %d", rec.Code)
	}
	if code := errorCode(out); code != "UNAUTHORIZED" {
		t.Fatalf("envelope code: %q", code)
	}
}

func TestWalletViewIsZeroForFreshUser(t *testing.T) {
	db := apiDB(t)
	user := &core.Principal{UserID: uuid.NewString(), Role: "user"}
	e := server(t, db, user)

	rec, out := do(t, e, http.MethodGet, "/api/economy/wallet", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("wallet: status %d body %s", rec.Code, rec.Body.String())
	}
	data, _ := out["data"].(map[string]any)
	if data == nil || data["balance"].(float64) != 0 {
		t.Fatalf("unexpected wallet payload: %v", out)
	}
}

func TestAdminMutationsRequireAdminRole(t *testing.T) {
	db := apiDB(t)
	user := &core.Principal{UserID: uuid.NewString(), Role: "user"}
	e := server(t, db, user)

	rec, _ := do(t, e, http.MethodGet, "/api/economy/admin/ads", "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin admin route: status %d", rec.Code)
	}
}

func TestCreateAdValidatesHTTPSAffiliate(t *testing.T) {
	db := apiDB(t)
	admin := &core.Principal{UserID: uuid.NewString(), Role: core.RoleAdmin}
	e := server(t, db, admin)

	body := `{"code":"promo","title":"Promo","media_type":"text","body_text":"hello",
	          "target_url":"http://insecure.example.com","watch_seconds":5,"watch_reward":1,"active":true}`
	rec, out := do(t, e, http.MethodPost, "/api/economy/admin/ads", body)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("http affiliate: status %d body %s", rec.Code, rec.Body.String())
	}
	if code := errorCode(out); code != "VALIDATION_ERROR" {
		t.Fatalf("envelope code: %q", code)
	}
}

func TestPickReturnsAvailableAd(t *testing.T) {
	db := apiDB(t)
	admin := &core.Principal{UserID: uuid.NewString(), Role: core.RoleAdmin}
	adminSrv := server(t, db, admin)

	code := "api-" + uuid.NewString()[:8]
	body := `{"code":"` + code + `","title":"Promo","media_type":"image",
	          "media_url":"https://cdn.example.com/a.png","target_url":"https://affiliate.example.com/x",
	          "watch_seconds":5,"watch_reward":1,"active":true}`
	rec, out := do(t, adminSrv, http.MethodPost, "/api/economy/admin/ads", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create ad: status %d body %s", rec.Code, rec.Body.String())
	}
	data, _ := out["data"].(map[string]any)
	adID, _ := data["id"].(string)
	t.Cleanup(func() { db.Exec(`DELETE FROM ads WHERE id = ?`, adID) })

	user := &core.Principal{UserID: uuid.NewString(), Role: "user"}
	userSrv := server(t, db, user)
	rec, out = do(t, userSrv, http.MethodGet, "/api/economy/ads/pick", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("pick: status %d body %s", rec.Code, rec.Body.String())
	}
	picked, _ := out["data"].(map[string]any)
	if picked["ad"] == nil {
		t.Fatalf("pick returned no ad: %v", out)
	}
}

// errorCode digs the stable envelope code out of a decoded body.
func errorCode(out map[string]any) string {
	e, _ := out["error"].(map[string]any)
	if e == nil {
		return ""
	}
	code, _ := e["code"].(string)
	return code
}

// doWithHeaders is do() plus extra request headers.
func doWithHeaders(t *testing.T, e *echo.Echo, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// TestWebhookIsPublicButSignatureChecked documents the transport contract: the
// provider has no bearer token, so the route must not require auth, and the
// only thing that authorises it is the adapter's signature check.
func TestWebhookIsPublicButSignatureChecked(t *testing.T) {
	db := apiDB(t)
	// No principal at all: the stub identity sets none, so any route behind the
	// auth middleware would answer 401.
	e := server(t, db, nil)

	rec := doWithHeaders(t, e, http.MethodPost, "/api/economy/payments/webhook/test",
		`{"event_id":"x","status":"paid"}`, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unsigned webhook: got %d want 401 (it must be signature-checked, not auth-checked)", rec.Code)
	}

	// An unknown provider is acknowledged, not retried.
	rec = doWithHeaders(t, e, http.MethodPost, "/api/economy/payments/webhook/nope", `{}`, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("unknown provider: got %d want 200", rec.Code)
	}
}

// TestOrderRequiresIdempotencyKey covers the transport half of the rule: the
// key may arrive as a header, and missing it is a 400.
func TestOrderRequiresIdempotencyKey(t *testing.T) {
	db := apiDB(t)
	admin := &core.Principal{UserID: uuid.NewString(), Role: core.RoleAdmin}
	adminSrv := server(t, db, admin)

	code := "hdr-" + uuid.NewString()[:8]
	rec, out := do(t, adminSrv, http.MethodPost, "/api/economy/admin/products",
		`{"code":"`+code+`","name":"Header Pack","kind":"token_pack","currency":"TRY",
		  "amount_minor":1000,"tokens":10,"active":true}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create product: %d %s", rec.Code, rec.Body.String())
	}
	data, _ := out["data"].(map[string]any)
	productID, _ := data["id"].(string)
	t.Cleanup(func() { db.Exec(`DELETE FROM products WHERE id = ?`, productID) })

	user := &core.Principal{UserID: uuid.NewString(), Role: "user"}
	userSrv := server(t, db, user)

	// No header, no body key: refused before any order row exists.
	rec, out = do(t, userSrv, http.MethodPost, "/api/economy/orders",
		`{"product_id":"`+productID+`"}`)
	if rec.Code != http.StatusBadRequest || errorCode(out) != "IDEMPOTENCY_KEY_REQUIRED" {
		t.Fatalf("missing key: %d %s", rec.Code, rec.Body.String())
	}

	// Header form works, and the response carries the server price.
	key := "hdr-" + uuid.NewString()
	rec = doWithHeaders(t, userSrv, http.MethodPost, "/api/economy/orders",
		`{"product_id":"`+productID+`","amount_minor":1}`,
		map[string]string{"Idempotency-Key": key})
	if rec.Code != http.StatusCreated {
		t.Fatalf("order: %d %s", rec.Code, rec.Body.String())
	}

	var amount int64
	db.Raw(`SELECT amount_minor FROM orders WHERE idempotency_key = ?`, key).Scan(&amount)
	if amount != 1000 {
		t.Fatalf("order stored amount %d; the body's amount_minor must be ignored", amount)
	}
}

// TestUserCannotSeeAnotherUsersOrder is the 404-not-403 rule at the edge.
func TestUserCannotSeeAnotherUsersOrder(t *testing.T) {
	db := apiDB(t)
	admin := &core.Principal{UserID: uuid.NewString(), Role: core.RoleAdmin}
	adminSrv := server(t, db, admin)

	code := "priv-" + uuid.NewString()[:8]
	rec, out := do(t, adminSrv, http.MethodPost, "/api/economy/admin/products",
		`{"code":"`+code+`","name":"Private","kind":"regular","currency":"TRY",
		  "amount_minor":100,"entitlement":"private.x","active":true}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create product: %d", rec.Code)
	}
	data, _ := out["data"].(map[string]any)
	productID, _ := data["id"].(string)
	t.Cleanup(func() { db.Exec(`DELETE FROM products WHERE id = ?`, productID) })

	owner := &core.Principal{UserID: uuid.NewString(), Role: "user"}
	ownerSrv := server(t, db, owner)
	key := "own-" + uuid.NewString()
	doWithHeaders(t, ownerSrv, http.MethodPost, "/api/economy/orders",
		`{"product_id":"`+productID+`"}`,
		map[string]string{"Idempotency-Key": key})

	var orderID string
	db.Raw(`SELECT id FROM orders WHERE idempotency_key = ?`, key).Scan(&orderID)
	if orderID == "" {
		t.Fatal("order was not created")
	}

	other := &core.Principal{UserID: uuid.NewString(), Role: "user"}
	otherSrv := server(t, db, other)
	rec, out = do(t, otherSrv, http.MethodGet, "/api/economy/orders/"+orderID, "")
	if rec.Code != http.StatusNotFound || errorCode(out) != "ORDER_NOT_FOUND" {
		t.Fatalf("foreign order: %d %s (existence must not be revealed)", rec.Code, rec.Body.String())
	}
}

// TestUsersCannotMintTokens is the regression guard for the most dangerous
// mistake this module could make: a user-facing credit endpoint.
func TestUsersCannotMintTokens(t *testing.T) {
	db := apiDB(t)
	user := &core.Principal{UserID: uuid.NewString(), Role: "user"}
	e := server(t, db, user)

	// The route does not exist at all.
	rec, _ := do(t, e, http.MethodPost, "/api/economy/wallet/credit", `{"amount":1000}`)
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("user credit route exists: status %d", rec.Code)
	}

	// And the admin route is role-gated.
	target := uuid.NewString()
	rec, _ = do(t, e, http.MethodPost, "/api/economy/admin/wallets/"+target+"/credit", `{"amount":1000}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin credit: status %d", rec.Code)
	}
}

func TestAdminCreditDebitEndpoint(t *testing.T) {
	db := apiDB(t)
	admin := &core.Principal{UserID: uuid.NewString(), Role: core.RoleAdmin}
	target := uuid.NewString()
	e := server(t, db, admin)

	rec, out := do(t, e, http.MethodPost, "/api/economy/admin/wallets/"+target+"/credit",
		`{"amount":25,"idempotency_key":"manual:`+target+`"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("admin credit: status %d body %s", rec.Code, rec.Body.String())
	}
	data, _ := out["data"].(map[string]any)
	if data["reason"] != econ.ReasonAdminCredit || data["delta"].(float64) != 25 {
		t.Fatalf("unexpected credit entry: %v", data)
	}

	// Replay with the same key returns the original entry, no double credit.
	rec, out = do(t, e, http.MethodPost, "/api/economy/admin/wallets/"+target+"/credit",
		`{"amount":25,"idempotency_key":"manual:`+target+`"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("replayed credit: status %d", rec.Code)
	}
	replay, _ := out["data"].(map[string]any)
	if replay["id"] != data["id"] {
		t.Fatalf("replay created a second entry: %v vs %v", replay["id"], data["id"])
	}

	rec, out = do(t, e, http.MethodPost, "/api/economy/admin/wallets/"+target+"/debit", `{"amount":5}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("admin debit: status %d body %s", rec.Code, rec.Body.String())
	}
	debit, _ := out["data"].(map[string]any)
	if debit["reason"] != econ.ReasonAdminDebit || debit["delta"].(float64) != -5 {
		t.Fatalf("unexpected debit entry: %v", debit)
	}

	// Spending more than the balance is a 402, not a negative balance.
	rec, out = do(t, e, http.MethodPost, "/api/economy/admin/wallets/"+target+"/debit", `{"amount":999}`)
	if rec.Code != http.StatusPaymentRequired || errorCode(out) != "INSUFFICIENT_BALANCE" {
		t.Fatalf("over-debit: status %d body %s", rec.Code, rec.Body.String())
	}
}
