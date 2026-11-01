# Faz 1 — Contracts + Bootstrap (taslak)

> Durum: **taslak çalışıyor** (2026-10-24)
> Kapsam: `contracts/` paketi + `cmd/core` referans başlatıcı.
> Core modüllerinin (auth, wallet, payments…) gerçek implementasyonu bu fazın
> devamıdır; bu belge neyin hazır, neyin eksik olduğunu net söyler.

---

## 1. Ne teslim edildi

```
contracts/
├── types.go        sentinel hatalar, User, LedgerEntry, AI/Storage/Payment tipleri
├── interfaces.go   Platform, Identity, Wallet, Entitlements, AI, Storage,
│                   Notifications, Payments, Dependencies, ConfigView, MigrationSource
├── module.go       Module arayüzü, App, doğrulama, topolojik sıralama
├── config.go       ortam değişkeni tabanlı Settings + fail-fast doğrulama
├── migrate.go      şema başına SQL migration runner
├── bootstrap.go    Run / RunWithSignals, HTTP, middleware, health, shutdown
└── *_test.go       config, sıralama, döngü, hata eşleme, HTTP zarfı

cmd/core/main.go    Core-only referans binary (modülsüz)
```

Derleme ve test:

```bash
go build ./contracts ./cmd/core
go test ./contracts -race
```

> Not: `go build ./...` hâlâ başarısız; sebep bu faz değil, Faz 0'da kurtarılacak
> eski modüllerdeki bozukluk. `contracts` ve `cmd/core` bağımsız derlenir.

---

## 2. AstroScope nasıl bağlanır (kod)

AstroScope yalnızca `core-platform/contracts` import eder.

```go
package main

import (
    "core-platform/contracts"
    "github.com/org/astroscope/internal/natal"
    "github.com/org/astroscope/migrations"
)

var version = "0.1.0"

func main() {
    err := contracts.RunWithSignals(contracts.App{
        Name:       "astroscope",
        Version:    version,
        Modules:    []contracts.Module{natal.New()},
        Migrations: []contracts.MigrationSource{migrations.Astro},
        Dependencies: contracts.Dependencies{
            Identity:     authAdapter,      // core auth modülünden gelir
            Wallet:       walletAdapter,
            Entitlements: entitlementAdapter,
            AI:           aiAdapter,
            Storage:      storageAdapter,
            Payments:     paymentsAdapter,
        },
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

Bootstrap otomatik olarak şunları yapar:

- `APP_ENV`, `DB_*`, `SERVER_*` ayarlarını okur ve doğrular.
- PostgreSQL'e bağlanır (`default_query_exec_mode=simple_protocol`, çok ifadeli
  migration'lar için).
- `App.Migrations` → modül `Migrate` sırasıyla şemayı uygular.
- Modülleri **bağımlılık sırasına** göre dizer; bilinmeyen/devre dışı bağımlılık
  ve döngüde başlamaz.
- Route'ları `/api/v1/apps/<app>` altına, `Identity.Middleware()` ile mount eder.
- `/healthz`, `/readyz`, `/version` ekler.
- `SIGINT/SIGTERM` ile graceful shutdown yapar; modülleri ters sırada durdurur.

### Modül örneği

```go
type Natal struct{}

func New() *Natal { return &Natal{} }

func (m *Natal) Name() string            { return "natal" }
func (m *Natal) Version() string         { return "1.0.0" }
func (m *Natal) Dependencies() []string  { return []string{"auth", "wallet"} }

func (m *Natal) Migrate(ctx context.Context, db *gorm.DB) error { return nil }

func (m *Natal) RegisterRoutes(g *echo.Group, d contracts.Dependencies) {
    g.POST("/natal", func(c echo.Context) error {
        user, err := d.Identity.CurrentUser(c)
        if err != nil {
            return err
        }
        // ...
        return c.JSON(http.StatusOK, result)
    })
}

func (m *Natal) Start(context.Context) error { return nil }
func (m *Natal) Stop(context.Context) error  { return nil }
```

`Dependencies()` içinde `auth`/`wallet` yazıyorsan, bootstrap'a o modüllerin de
`App.Modules` içinde verilmesi gerekir. Aksi halde "unknown module" hatası alınır
(bu bilinçli bir güvenlik kontrolüdür).

---

## 3. Migration sözleşmesi

```go
var Astro = contracts.MigrationSource{
    Schema: "astroscope",
    FS:     migrations.FS,
    Dir:    ".",
}
```

Garantiler:

- Şema adı doğrulanır (`^[a-z_][a-z0-9_]*$`), yoksa oluşturulur.
- Her şema kendi `schema_migrations` tablosunu tutar; core ve uygulama geçmişi
  karışmaz.
- `pg_advisory_lock(hashtext(schema))` ile eşzamanlı deploy engellenir.
- Her dosya kendi transaction'ında çalışır; hata → rollback.
- Uygulanmış dosyanın içeriği değişmişse **hata** verir (sessiz atlama yok).
- Dosya adı sırası uygulama sırasıdır (`0001_...`, `0002_...`).

---

## 4. Yapılandırma

Ortam değişkenleri (bu taslakta YAML henüz yok):

| Değişken | Varsayılan | Not |
|---|---|---|
| `APP_ENV` | `development` | production/test/staging |
| `SERVER_HOST` / `SERVER_PORT` | `0.0.0.0` / `8080` | tek port kaynağı |
| `DB_HOST` / `DB_PORT` | `localhost` / `5432` | |
| `DB_USER` / `DB_PASSWORD` | `postgres` / — | prod'da zorunlu |
| `DB_NAME` | `core_platform` | |
| `DB_SSLMODE` | `disable` | prod'da `disable` yasak |
| `DB_MAX_OPEN_CONNS` | `25` | |
| `CORS_ALLOWED_ORIGINS` | — | prod'da zorunlu, `*` yasak |
| `HTTP_BODY_LIMIT` | `8M` | |
| `HTTP_REQUEST_TIMEOUT` | `30s` | |
| `HTTP_SHUTDOWN_TIMEOUT` | `10s` | |
| `LOG_LEVEL` / `LOG_FORMAT` | `info` / `json` | `json` veya `console` |

Production'da eksik/tehlikeli değer **fail-fast** ile süreci durdurur.

---

## 5. HTTP davranışı

Middleware sırası:

```
RequestID → AccessLog → Recover → BodyLimit → Timeout → SecurityHeaders → CORS
```

- Hata zarfı tek biçim:

```json
{
  "success": false,
  "error": { "code": "NOT_FOUND", "message": "resource not found" },
  "request_id": "..."
}
```

- `contracts` sentinel hataları HTTP'ye tek yerde eşlenir:
  `ErrUnauthorized → 401`, `ErrForbidden → 403`, `ErrNotFound → 404`,
  `ErrValidation → 422`, `ErrInsufficientBalance → 402`,
  `ErrEntitlementRequired → 402`, `ErrRateLimited → 429`,
  `ErrProviderFailure → 502`.
- 5xx'te iç hata mesajı client'a **gitmez**, log'a gider.
- `Identity` verilmeden route mount edilirse bootstrap **başlamaz**
  (fail-closed).
- `X-Request-ID` yanıtta döner.

---

## 6. Bilinçli olarak eksik bırakılanlar

Bunlar Faz 1'in devamı veya sonraki fazlardır; taslakta yok:

1. **YAML/config dosyası okuma.** Şu an yalnızca env. `config.yaml` +
   `modules.yaml` önceliği (env > yaml > varsayılan) eklenecek.
2. **Core modüllerinin gerçek adaptörleri.** `Identity`, `Wallet`,
   `Entitlements`, `AI`, `Storage`, `Notifications`, `Payments` arayüzleri hazır;
   bunları `modules/auth`, `modules/wallet` vb. implemente edecek ve bootstrap'a
   bağlayacak. Bu taslakta `Dependencies` uygulama tarafından doldurulur.
3. **`modules.yaml` durum haritası.** `App.EnabledModules` alanı var; config'ten
   okunması bağlanmadı.
4. **OpenAPI üretimi**, rate limiter, CSRF, request body şeması doğrulama.
5. **`cmd/server`'ın yeni bootstrap'a taşınması.** Faz 0 sonrası yapılacak.
6. **Entegrasyon testleri.** Migration/DB testleri `DATABASE_URL` verildiğinde
   çalışacak şekilde eklenecek; şu an unit testler var.

---

## 7. Sıradaki 5 iş (Faz 1 devamı)

1. `core/config` → YAML + env birleştirme; `contracts.Settings`'a bağlama.
2. `modules/auth` → `contracts.Identity` implementasyonu (Bearer + cookie,
   refresh rotation); `contracts.AuthAdapter` bootstrap'a.
3. `modules/wallet` → `contracts.Wallet` (ledger + idempotency + row lock).
4. `modules/payments` → `contracts.Payments` (Stripe/PayTR/havale adapter'ları).
5. `cmd/server` → `contracts.Run` ile gerçek core modüllerini kaydet;
   `/api/v1/admin/modules` durum endpoint'i.

Her adımda kapı: `go build ./contracts ./cmd/...`, `go test ./contracts -race`,
`go vet`, ve ilgili modülün testleri.
