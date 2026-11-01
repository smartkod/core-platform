# AstroScope ↔ Core Platform Entegrasyonu

> Durum: **KARAR — kabul edildi** (2026-10-24)
> İlgili belgeler: `doc/GELISTIRME-PLANI.md`, `doc/ARCHITECTURE.md` (Faz 8'de üretilecek)
>
> Bu belge, AstroScope'un Core Platform'u **nasıl tüketeceğini** tanımlar.
> Amaç: iki depo arasında net, test edilebilir ve sürümlenebilir bir sözleşme kurmak;
> Core'un içine uygulama mantığı, uygulamanın içine platform altyapısı sızdırmamak.

---

## 1. Karar Özeti

| Konu | Karar |
|---|---|
| Depo yapısı | **AstroScope ayrı Git deposudur.** Core bu depoda uygulama barındırmaz. |
| Tüketim biçimi | AstroScope, Core'u **Go modülü** olarak tüketir (`require core-platform vX.Y.Z`). |
| Çalışma biçimi | Tek binary. AstroScope kendi `main.go`'suyla Core'u **bootstrap eder** ve kendi modüllerini kaydeder. |
| Dağıtım | Ayrı konteyner imajı; **aynı PostgreSQL**, ayrı şema (`astroscope`). |
| Veritabanı | Core tablolarına yazmaz. Yalnızca `users`/`wallets` gibi tabloları **ID ile** referanslar. |
| Kimlik | Core'un ürettiği JWT. AstroScope kendi auth'unu yazmaz. |
| Ödeme | Core `payments` modülü. AstroScope fiyat/plan tanımlar, ödeme akışını Core yönetir. |
| AI | Core `ai_gateway` + `prompt_engine` + `context_engine` üzerinden. AstroScope doğrudan OpenAI çağırmaz. |
| API yolu | `/api/v1/apps/astroscope/...` (uygulama öneki zorunlu). |
| Sürümleme | Core SemVer; AstroScope uyumluluk matrisi tutar. |

**Neden ayrı depo?**

- `YOL HARİTASI.txt` bunu zaten söylüyor: uygulamalar ayrı depo, kendi sürümünü takip eder.
- Core'un test/build süresi uygulamadan bağımsız kalır.
- AstroScope'un efemeris/lisans bağımlılıkları Core'a bulaşmaz.
- Aynı Core üzerine EduScope/FitScope eklenirken çakışma olmaz.

**Neden tek binary (mikroservis değil)?**

- Modüler monolit kararı korunuyor. Ayrı depo ≠ ayrı servis.
- Ağ atlaması, dağıtık transaction, iki ayrı auth yok.
- İleride ayrıştırma gerekirse modül sınırları zaten hazır olur.

---

## 2. Sorumluluk Ayrımı

| Yetenek | Core Platform | AstroScope |
|---|---|---|
| Config, logger, DB bağlantısı | ✅ sağlar | tüketir |
| Modül yaşam döngüsü | ✅ sağlar | kendi modüllerini kaydeder |
| Migration altyapısı | ✅ runner | kendi migration'larını verir |
| Kayıt / giriş / refresh / RBAC | ✅ sağlar | tüketir |
| Kullanıcı profili | ✅ sağlar | genişletmez, referanslar |
| Wallet / jeton | ✅ sağlar | harcama/ödeme çağırır |
| Premium / entitlement | ✅ sağlar | `CheckEntitlement("astroscope.pro")` |
| Ödeme (Stripe/PayTR/havale) | ✅ sağlar | plan/ürün tanımlar |
| Dosya depolama | ✅ sağlar | tüketir |
| Bildirim | ✅ sağlar | olay yayınlar |
| AI gateway / kota | ✅ sağlar | tüketir |
| Prompt/context yönetimi | ✅ sağlar | şablonlarını tanımlar |
| Burç/gezegen/ev hesaplama | ❌ | ✅ |
| Efemeris, natal, transit, sinastri | ❌ | ✅ |
| Astroloji yorumları, içerik | ❌ | ✅ |
| AstroScope'a özel tablolar | ❌ | ✅ |
| AstroScope UI/API sözleşmesi | ❌ | ✅ |

**Kural:** Core'da "burç", "gezegen", "natal" kelimesi geçiyorsa yanlış yere yazılmıştır.
Tek istisna: AI prompt şablonları (veri, kod değil).

---

## 3. Depo ve Modül Stratejisi

### 3.1 Core deposu

```
core-platform/            (bu depo)
├── go.mod                module core-platform
├── core/                 platform çekirdeği
├── modules/              auth, users, wallet, premium, payments, ...
├── contracts/            uygulamaların kullanacağı arayüzler (YENİ)
├── migrations/           core şeması
└── cmd/server/           referans bootstrap (opsiyonel)
```

### 3.2 AstroScope deposu

```
astroscope/
├── go.mod                module github.com/<org>/astroscope
│                         require core-platform v1.2.0
├── cmd/astroscope/
│   └── main.go           core bootstrap + astro modülleri
├── internal/
│   ├── natal/
│   ├── transit/
│   ├── progression/
│   ├── solarreturn/
│   ├── synastry/
│   ├── ephemeris/        Swiss Ephemeris sarmalayıcı
│   └── content/
├── migrations/           astroscope şeması (0001_..., 0002_...)
├── api/openapi.yaml
└── testdata/golden/      altın haritalar
```

### 3.3 Geliştirme ve üretim bağlama

Yerel geliştirmede `go.work`:

```
go work init
go work use ../core-platform ./astroscope
```

Üretimde `go.work` **kullanılmaz**. Sürümlenmiş modül:

```
require core-platform v1.2.0
```

`replace` yalnızca geçici hata ayıklamada kullanılır ve **commit edilmez**.
CI, `go mod verify` + `replace` yok kontrolü yapar.

Core'da kırıcı değişiklik = major sürüm (`v2`). AstroScope uyumluluk matrisi:

| AstroScope | Core Platform | Durum |
|---|---|---|
| 1.0.x | 1.0.x – 1.2.x | desteklenir |
| 1.1.x | ≥ 1.3.0 | desteklenir |
| 2.0.x | ≥ 2.0.0 | desteklenir |

---

## 4. Core'un AstroScope'a Sunduğu Sözleşmeler

`contracts/` paketi **kararlı API**'dir. Core iç modülleri değişse bile bu arayüzler
kırılmaz (kırılırsa major sürüm).

```go
// contracts/contracts.go
package contracts

import (
    "context"
    "time"

    "github.com/labstack/echo/v4"
    "go.uber.org/zap"
    "gorm.io/gorm"
)

// Platform, uygulamanın Core'a eriştiği tek noktadır.
type Platform interface {
    DB() *gorm.DB                  // yalnızca kendi şeması için
    Logger() *zap.Logger
    Config() ConfigView
    Clock() Clock
}

// Identity — token doğrulama ve kullanıcı bilgisi.
type Identity interface {
    Middleware() echo.MiddlewareFunc          // Bearer + cookie
    RequireScopes(scopes ...string) echo.MiddlewareFunc
    CurrentUser(c echo.Context) (User, error)
}

type User struct {
    ID        string
    Email     string
    Locale    string
    Roles     []string
    Scopes    []string
    Status    string
}

// Wallet — jeton hareketleri. Doğrudan bakiye yazılamaz.
type Wallet interface {
    Balance(ctx context.Context, userID string) (int64, error)
    Debit(ctx context.Context, req DebitRequest) (LedgerEntry, error)
    Credit(ctx context.Context, req CreditRequest) (LedgerEntry, error)
}

type DebitRequest struct {
    UserID         string
    Amount         int64
    Reason         string   // "astroscope.natal_report"
    IdempotencyKey string   // zorunlu
    Metadata       map[string]string
}

// Entitlements — premium/plan kontrolü.
type Entitlements interface {
    Has(ctx context.Context, userID, feature string) (bool, error)
    Require(feature string) echo.MiddlewareFunc
}

// AI — sağlayıcı detayı sızdırmaz.
type AI interface {
    Complete(ctx context.Context, req AIRequest) (AIResponse, error)
}

type AIRequest struct {
    UserID      string
    Feature     string            // "astroscope.interpretation"
    Prompt      string
    System      string
    Model       string            // boşsa Core varsayılanı
    MaxTokens   int
    Temperature float64
    CacheKey    string            // opsiyonel
}

// Storage — lokal/S3 farkını gizler.
type Storage interface {
    Put(ctx context.Context, req PutRequest) (FileRef, error)
    SignedURL(ctx context.Context, fileID string, ttl time.Duration) (string, error)
}

// Notifications — olay bazlı.
type Notifications interface {
    Send(ctx context.Context, n Notification) error
}

// Payments — plan tanımı Core'da, ürün kataloğu uygulamada.
type Payments interface {
    RegisterProduct(ctx context.Context, p Product) error
    CheckoutURL(ctx context.Context, req CheckoutRequest) (string, error)
    Entitlement(ctx context.Context, userID, productID string) (bool, error)
}
```

**AstroScope yalnızca bu paketi import eder.** `core/*` veya `modules/*` internal
paketlerini import etmez. CI'da `depguard`/`go list` ile denetlenir.

### 4.1 Core'un sağladığı `App` başlatıcısı

```go
// contracts/bootstrap.go
type App struct {
    Name       string
    Version    string
    Modules    []Module          // uygulama modülleri
    Migrations []MigrationSource
    Routes     func(g *echo.Group, p Platform, d Dependencies)
}

type Dependencies struct {
    Identity      Identity
    Wallet        Wallet
    Entitlements  Entitlements
    AI            AI
    Storage       Storage
    Notifications Notifications
    Payments      Payments
}

// Run; config → logger → db → migration → core modules → app modules → http → shutdown
func Run(app App) error
```

---

## 5. AstroScope `main.go` — Tam Örnek

```go
package main

import (
    "core-platform/contracts"
    "github.com/<org>/astroscope/internal/natal"
    "github.com/<org>/astroscope/internal/synastry"
    "github.com/<org>/astroscope/internal/transit"
    "github.com/<org>/astroscope/migrations"
    "github.com/labstack/echo/v4"
)

func main() {
    err := contracts.Run(contracts.App{
        Name:       "astroscope",
        Version:    version,
        Modules: []contracts.Module{
            natal.New(),
            transit.New(),
            synastry.New(),
        },
        Migrations: migrations.All,
        Routes: func(g *echo.Group, p contracts.Platform, d contracts.Dependencies) {
            // g = /api/v1/apps/astroscope
            g.POST("/natal", natalHandler(p, d))
            g.GET("/transit", transitHandler(p, d))
            g.POST("/synastry", synastryHandler(p, d))
        },
    })
    if err != nil {
        os.Exit(1)
    }
}
```

`main.go` iş kuralı içermez. 60 satırı geçmez.

---

## 6. Modül Tanımı (AstroScope tarafı)

Core'un `contracts.Module` arayüzü:

```go
type Module interface {
    Name() string
    Version() string
    Dependencies() []string          // core modül adları: "auth", "wallet", "ai_gateway"
    Migrate(ctx context.Context, db *gorm.DB) error
    RegisterRoutes(g *echo.Group, d Dependencies)
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
}
```

AstroScope modülleri `Dependencies()` içinde **yalnızca gerçekten kullandığını**
bildirir. Örnek:

```go
func (m *Natal) Dependencies() []string {
    return []string{"auth", "wallet", "ai_gateway"}
}
```

Module manager topolojik sıralar; `ai_gateway` yoksa AstroScope başlamaz ve
anlaşılır hata verir.

---

## 7. Veri Sahipliği ve Şema

### 7.1 Şema ayrımı

| Şema | Sahibi | İçerik |
|---|---|---|
| `public` | Core | `users`, `wallets`, `wallet_transactions`, `payments`, `files`, `notifications`, `prompt_templates`, ... |
| `astroscope` | AstroScope | `birth_charts`, `transits`, `synastry_reports`, `interpretations`, ... |

Kurallar:

- AstroScope migration'ları yalnızca `astroscope` şemasında çalışır.
  İlk migration: `CREATE SCHEMA IF NOT EXISTS astroscope;`
- Core migration'ları `astroscope` şemasına dokunmaz.
- Foreign key'ler şemalar arası **yalnızca** `users(id)` için kurulabilir; başka
  core tablosuna FK kurulmaz (bağımlılık yönü aşağı).
- AstroScope, Core tablolarını `JOIN` ile okumaz. Gerekli veriyi `contracts`
  üzerinden alır. (Tek istisna: raporlama için read-only view — ADR gerekir.)

### 7.2 Migration sırası

```
1. core migrations        (schema_migrations_core)
2. astroscope migrations  (astroscope.schema_migrations)
```

Runner iki ayrı geçmiş tablosu tutar; biri başarısızsa diğeri etkilenmez.
Dağıtımda Core job'u AstroScope'tan **önce** çalışır.

### 7.3 ID stratejisi

- Tüm yeni tablolar `uuid` (UUIDv7) kullanır.
- AstroScope tabloları da aynı stratejiyi izler; `bigserial` kullanmaz.
- Dışarıya açılan ID'ler UUID'dir; sıralı ID sızdırılmaz.

---

## 8. API Yüzeyi

### 8.1 Yol kuralı

```
/api/v1/apps/astroscope/natal
/api/v1/apps/astroscope/transit
/api/v1/apps/astroscope/synastry
/api/v1/apps/astroscope/charts/{id}
```

- Uygulama öneki zorunlu; Core route'larıyla çakışmaz.
- Sürüm Core'un URL sürümüdür. AstroScope kendi iç sürümünü header'da bildirir:
  `X-App-Version: 1.4.2`.
- Ortak hata zarfı, request ID, rate limit **Core middleware'inden** gelir.
  AstroScope kendi zarfını üretmez.

### 8.2 Sağlık

AstroScope ek endpoint eklemez. `/readyz` Core'a aittir; AstroScope modülleri
`Start()` başarılıysa readiness "ready" sayılır. AstroScope'un dış bağımlılığı
varsa (efemeris dosyası) `Start()` içinde doğrular ve hata döner.

### 8.3 OpenAPI

AstroScope kendi `api/openapi.yaml` dosyasını üretir; CI'da Core'un
`/api/v1/openapi.json` çıktısıyla birleştirilir. Çakışan path/operationId build'i kırar.

---

## 9. Kimlik ve Yetkilendirme

- AstroScope **kendi login/register ekranını Core'a devreder** (veya Core
  endpoint'lerini çağırır). İkinci bir kullanıcı tablosu yoktur.
- Token doğrulama Core `Identity.Middleware()` ile yapılır.
- Kullanıcı ID'si **asla** request body/query'den alınmaz; yalnızca context'ten.
- AstroScope'a özel yetkiler scope olarak tanımlanır:
  `astroscope.read`, `astroscope.chart.create`, `astroscope.report.premium`.
- Admin işlemleri Core RBAC'ındaki `admin` rolüne bağlanır; AstroScope kendi
  admin kontrolünü yazmaz.

```go
g.POST("/natal",
    d.Identity.RequireScopes("astroscope.chart.create"),
    natalHandler(p, d))
```

Kimlik modelinin ayrıntısı (Bearer + cookie, refresh rotation) Core planındadır.

---

## 10. Wallet, Premium ve Ödeme Kullanımı

### 10.1 Jeton harcama

```go
entry, err := d.Wallet.Debit(ctx, contracts.DebitRequest{
    UserID:         user.ID,
    Amount:         20,
    Reason:         "astroscope.natal_report",
    IdempotencyKey: req.RequestID,     // istemciden gelen benzersiz anahtar
    Metadata:       map[string]string{"chart_id": chartID},
})
```

- Bakiye yetersizse `contracts.ErrInsufficientBalance` döner → HTTP 402/409.
- Aynı `IdempotencyKey` ikinci kez gelirse **aynı** ledger kaydı döner; çift düşmez.
- AstroScope `wallets` tablosuna doğrudan yazmaz.

### 10.2 Premium

```go
ok, err := d.Entitlements.Has(ctx, user.ID, "astroscope.pro")
if !ok {
    return echo.NewHTTPError(http.StatusPaymentRequired, "premium required")
}
```

### 10.3 Ödeme

AstroScope ürün kataloğunu Core'a kaydeder:

```go
d.Payments.RegisterProduct(ctx, contracts.Product{
    ID:          "astroscope.pro.monthly",
    Name:        "AstroScope Pro — Aylık",
    AmountMinor: 4999,          // 49,99 TRY
    Currency:    "TRY",
    Interval:    "month",
})
```

Ödeme akışı tamamen Core `payments` modülündedir (Stripe / PayTR / havale).
AstroScope ödeme sayfası barındırmaz, kart verisi görmez, webhook dinlemez.

---

## 11. AI Kullanımı

- AstroScope doğrudan `OPENAI_API_KEY` okumaz.
- Tüm çağrılar `d.AI.Complete` üzerinden; kota, cache, maliyet, log Core'da.
- `Feature` alanı zorunlu: kota ve faturalama bunun üzerinden.
- Prompt şablonları AstroScope'ta tanımlanır, Core `prompt_engine`'e kaydedilir.
- AI çıktısı **güvenilmeyen veri**dir: yorum metni HTML'e basılmadan önce
  escape edilir; kullanıcıya "AI üretimi" etiketi gösterilir.

```go
resp, err := d.AI.Complete(ctx, contracts.AIRequest{
    UserID:   user.ID,
    Feature:  "astroscope.interpretation",
    System:   "Sen deneyimli bir astroloji yorumcususun...",
    Prompt:   buildPrompt(chart),
    MaxTokens: 800,
    CacheKey: "natal:" + chartID + ":" + chart.UpdatedAt,
})
```

---

## 12. Config ve Ortam Değişkenleri

Core config'i okur; AstroScope yalnızca kendi ayarlarını ekler.

| Değişken | Sahibi | Örnek |
|---|---|---|
| `DATABASE_URL` / `DB_*` | Core | ortak |
| `JWT_SECRET`, `REFRESH_SECRET` | Core | ortak |
| `STRIPE_SECRET_KEY`, `PAYTR_*` | Core | ortak |
| `ASTRO_EPHEMERIS_PATH` | AstroScope | `/data/ephe` |
| `ASTRO_DEFAULT_HOUSE_SYSTEM` | AstroScope | `placidus` |
| `ASTRO_MAX_CHART_AGE_DAYS` | AstroScope | `3650` |

Kurallar:

- Aynı değişken adı iki yerde tanımlanmaz.
- AstroScope, Core'un `ConfigView`'ından okur; `os.Getenv` doğrudan kullanmaz
  (test edilebilirlik).
- Prod'da eksik zorunlu değişken → **fail-fast**.

---

## 13. Docker / Dağıtım

```yaml
services:
  postgres: { ... }

  core:
    image: registry/core-platform:1.2.0
    environment: [ ... ortak ... ]
    ports: ["8080:8080"]
    depends_on:
      postgres: { condition: service_healthy }

  astroscope:
    image: registry/astroscope:1.4.2
    environment:
      CORE_INTERNAL_URL: http://core:8080   # yalnızca gerekliyse
      ASTRO_EPHEMERIS_PATH: /data/ephe
    volumes: ["ephe:/data/ephe:ro"]
    depends_on:
      core: { condition: service_healthy }
```

**İki model vardır; biri seçilir:**

| Model | Açıklama | Ne zaman |
|---|---|---|
| **A — Gömülü (önerilen)** | AstroScope kendi binary'sinde Core'u bootstrap eder. Tek süreç. | V1 |
| **B — Ayrı servis** | Core ayrı çalışır, AstroScope HTTP ile konuşur. | Ölçek/ekip ayrımı gerekince |

V1 için **Model A**. Bu belgedeki `contracts.Run` buna göredir. Model B'ye geçiş
için `contracts` arayüzlerinin HTTP istemcisi yazılır; iş kodu değişmez.

---

## 14. Sürümleme ve Uyumluluk

- Core, `contracts` paketini SemVer ile sürümler.
- AstroScope, Core'un **minor** sürümlerinde kırılmaz; major'da uyum matrisi güncellenir.
- CI'da "contract test": AstroScope, Core'un en düşük desteklenen ve en yeni
  sürümüne karşı derlenir.
- Deprecation: `contracts` içinde en az 1 minor sürüm boyunca `// Deprecated:`
  işaretlenir, log uyarısı üretilir.

---

## 15. Test Stratejisi

| Test | Yer | Amaç |
|---|---|---|
| Altın harita testleri | AstroScope | Aynı doğum verisi → aynı sonuç |
| Efemeris doğrulama | AstroScope | Bilinen 10 referans harita (NASA/Swiss Ephemeris) |
| Contract test | AstroScope CI | `contracts` uyumu |
| Entegrasyon | AstroScope + Core | auth → wallet debit → rapor üretimi |
| Yetki testi | AstroScope | Başka kullanıcının `chart_id`'sine erişim 404/403 |
| Kota testi | Core | AI kotası aşılamıyor |
| Migration testi | İkisi | Core + app migration sırası, rollback |

Altın test kuralı: `math/rand` **yasak**. Zaman/konum bağımlılığı enjekte edilir
(`Clock` arayüzü), testte sabitlenir.

---

## 16. Yasal ve Lisans

- **Swiss Ephemeris** AGPL-3.0'dır. Ticari kapalı kaynak kullanım için ayrı lisans
  gerekir. Alternatifler (MIT/Apache efemeris kütüphaneleri) erken değerlendirilir.
- AstroScope çıktısı "eğlence ve kişisel gelişim amaçlıdır" uyarısı taşır.
- KVKK/GDPR: doğum tarihi, saati, yeri **kişisel veri**dir.
  - Saklama süresi tanımlanır, silme akışı Core üzerinden çalışır.
  - Loglara doğum verisi yazılmaz.
  - Üçüncü taraf AI'a gönderilen prompt'larda kimlik bilgisi maskelenir.
- Ödeme: kart verisi hiçbir zaman AstroScope'a girmez (Core/PCI kapsamı).

---

## 17. Mevcut `backend/astroscope` Kodunun Geçişi

Mevcut kod **referans** olarak kullanılır, taşınmaz:

| Dosya | Karar |
|---|---|
| `backend/astroscope/natal/service.go` | `math/rand` ve mock tamamen atılır; gerçek efemeris ile yeniden yazılır |
| `backend/astroscope/natal/model.go` | Şema `astroscope`ya taşınır; tag'ler onarılır; UUID |
| `backend/astroscope/natal/handler.go` | `userID = 1` kaldırılır; `contracts.Identity` |
| `backend/astroscope/natal/module.go` | `contracts.Module` arayüzüne uyarlanır; Core `module` paketi import edilmez |
| `backend/astroscope/transit/service.go` | Rastgele veri atılır; gerçek transit hesabı |
| `backend/astroscope/progression`, `solar_return` | Aynı şekilde yeniden yazılır |
| `backend/astroscope/synastry/service.go` | `rand.Intn` atılır; iki harita arası açı hesabı |

Adımlar:

1. Core'da `contracts` paketi ve `Run` bootstrap'ı hazırlanır (Faz 1).
2. AstroScope deposu açılır; iskelet + `go.work` kurulur.
3. Efemeris katmanı ve altın testler yazılır (en riskli kısım).
4. Natal → transit → synastry → progression sırasıyla.
5. Mevcut `backend/astroscope` bu depodan **silinir** (git geçmişinde kalır).

---

## 18. Yapılmaması Gerekenler (Anti-Pattern)

- AstroScope içinde `core-platform/modules/*` import etmek.
- Kendi `users` tablosunu açmak veya Core `users` tablosuna yazmak.
- JWT'yi kendi secret'ıyla üretmek.
- `wallets.balance` alanını doğrudan `UPDATE` etmek.
- OpenAI/Stripe anahtarını AstroScope env'ine koymak.
- Ödeme webhook'unu AstroScope'ta dinlemek.
- Core'un `schema_migrations` tablosuna yazmak.
- Uygulama öneki olmadan `/api/v1/natal` gibi global route açmak.
- Burç/gezegen sabitlerini Core'a taşımak.
- `math/rand` ile "hesaplama" yapmak.
- Altın test olmadan efemeris sürümünü değiştirmek.
- Core'da `if app == "astroscope"` gibi uygulama-özel dallar açmak.

---

## 19. Uygulama Sırası (Core hazır olduktan sonra)

| # | İş | Bağımlılık |
|---|---|---|
| 1 | AstroScope deposu + modül iskeleti | Core `contracts` |
| 2 | `go.work` ile yerel derleme | 1 |
| 3 | Efemeris sarmalayıcı + lisans kararı | — |
| 4 | Altın harita testleri | 3 |
| 5 | Natal hesap + API | 4 |
| 6 | Transit + API | 5 |
| 7 | Synastry + API | 5 |
| 8 | Progression / solar return | 5 |
| 9 | Wallet debit + premium gate | Core Faz 3 |
| 10 | AI yorum entegrasyonu | Core Faz 5 |
| 11 | OpenAPI + contract test | 5–8 |
| 12 | Docker imajı + staging | Core Faz 7 |

Tahmini: **5–10 iş günü** (efemeris lisansı ve altın testler kritik yol).
