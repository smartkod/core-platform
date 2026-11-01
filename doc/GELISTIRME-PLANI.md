# Core Platform — Durum Tespiti ve Geliştirme Planı

> Bu belge, deponun 2026-10-24 tarihindeki gerçek durumunu esas alır.
> `YOL HARİTASI.txt`, `_KALICI-PROJE-HAFIZASI/` ve `doc/ai-project-starter/` içindeki
> bazı bilgiler farklı bir projeye (Next.js monorepo, `13-Organizasyon - ECOSISTEM`)
> aittir. Çelişki halinde **bu belge ve gerçek kod** geçerlidir.
>
> AstroScope entegrasyonu ayrı belgede: **`doc/ASTROSCOPE-ENTEGRASYON.md`**.
> Verilmiş kararlar: D1–D5 (Bölüm 10.1).

---

## 1. Yönetici Özeti

Depo "Core Platform v1.0.0 final" olarak etiketlenmiş; ancak:

- `go build ./...` **başarısız**.
- 112 Go dosyasının **tamamında** girinti kaybolmuş (`gofmt -l` hepsini listeliyor, 0 dosyada tab var).
- 53 dosyada import yolları bozuk (`"modules/..."`, `"core-platform/modules/..."` olmalı).
- 13 dosyada struct tag'lerindeki backtick'ler silinmiş veya bozulmuş.
- `cmd/server/main.go` hiçbir modülü, router'ı, config dosyasını, migration'ı bağlamıyor.
- Hiçbir modül örneği (`NewAuthModule`, `NewWalletModule`, ...) hiçbir yerde çağrılmıyor.
- `ModuleManager` hiç kullanılmıyor; `AutoMigrate` hiç yok.
- 0 test dosyası, 0 migration, kök `README.md` yok.
- 130 route kaydı var; büyük bölümü mock veri döndürüyor veya hiç bağlanmıyor.
- Tüm handler'larda kullanıcı sabit `userID = 1`; authorization yok.
- Sekiz dil dosyasının metinleri çift-kodlama (mojibake) nedeniyle okunamaz durumda.

**Sonuç:** Mevcut durum "prototip iskeleti" bile değil; derlenmeyen bir taslak.
Doğru strateji sıfırdan her şeyi yazmak değil, **önce depoyu kurtarmak**, sonra
çekirdeği tamamlamak, sonra modülleri tek tek gerçek implementasyona çevirmektir.

**Önerilen yol:** Kurtarma + hedefli yeniden yazım (hibrit).
Tahmini efor: **1 geliştirici + AI ile 6–8 hafta** (aşağıda faz bazında).

---

## 2. Doğrulanmış Envanter

| Ölçüm | Değer |
|---|---|
| Go dosyası | 112 |
| Go satırı (yaklaşık) | ~9.000 |
| Test dosyası | 0 |
| Migration | 0 |
| Modül klasörü (`modules/`) | 15 |
| Uygulama klasörü (`backend/astroscope`) | 1 (kısmi) |
| Route kaydı | 130 |
| Bozuk import içeren dosya | 53 |
| Bozuk struct tag içeren dosya | 13 |
| Girintisi kaybolmuş dosya | 112 / 112 |
| Lokalizasyon dosyası | 8 (JSON geçerli, metinler bozuk) |
| Kök `main` fonksiyonu | 4 (`dosya.go`, `test_api.go`, `test_community.go`, `test_simple.go`) |
| Commit'lenmiş gereksiz binary | `git-filter-repo` (212 KB) |
| README | Yok (yalnızca `doc/ai-project-starter/README.md`) |

### 2.1 Derlemeyi kesin olarak durduran hatalar (P0)

1. **Girinti kaybı:** Tüm `.go` dosyalarında tab yok. `gofmt -l .` 112 dosya döndürüyor.
2. **Import yolu:** 53 dosyada `"modules/..."`; modül adı `core-platform` olduğu için
   `"core-platform/modules/..."` olmalı.
3. **Struct tag bozulması:** En az şu dosyalar:
   - `modules/ai_copilot/model/copilot.go`
   - `modules/ai_gateway/model/ai.go`
   - `modules/ai_gateway/clients/openai.go`
   - `modules/analytics/model/analytics.go`
   - `modules/community/model/community.go`
   - `modules/context_engine/model/context.go`
   - `modules/lifeweave/model/event.go`
   - `modules/notifications/model/notification.go`
   - `modules/premium/model/premium.go`
   - `modules/prompt_engine/model/prompt.go`
   - `modules/rewarded_ads/model/ad.go`
   - `modules/search/model/search.go`
   - `modules/wallet/model/wallet.go`
4. **Eksik `config.ModulesConfig`:** `core/module/manager.go` bu tipi kullanıyor,
   `core/config/config.go` içinde tanımlı değil.
5. **Eksik import/alanlar:** Örnekler:
   - `core/middleware/xss.go` → `io` importu yok.
   - `backend/astroscope/transit/handler.go` → `time` importu yok.
   - `modules/filestorage/service/file_service.go` → `net/http` importu yok.
   - `modules/rewarded_ads/service/ad_service.go` → `VerifiedAt` alanı modelde yok.
6. **Aynı pakette 4 `main`:** `go build ./...` kök pakette `main redeclared` verir.
7. **Kullanılmayan/yanlış kurulmuş paketler:** `core/security`, `core/middleware`,
   `core/localization`, `core/api` çoğunlukla hiçbir yerden çağrılmıyor.

> Not: 1–6 düzeltilmeden hiçbir mimari tartışma anlamlı değildir. Önce "yeşil build"
> kapısı açılmalıdır.

### 2.2 Yapısal boşluklar

- `main.go` yalnızca `/health` döndürüyor; `config.yaml`, `modules.yaml`, `.env`,
  `ModuleManager`, `Router`, `HealthChecker` kullanılmıyor.
- `config/config.yaml` ve `config/modules.yaml` hiç okunmuyor. `core/config` sadece
  ortam değişkeni okuyor; YAML/JSON tag'leri dekoratif.
- Port tutarsızlığı: kod `:8081`, config varsayılanı `8080`, Docker `8080`,
  `.env.example` `8081`.
- `docker-compose.yml` kaynak kodu `/app`'e mount ediyor; imaj içindeki binary ile
  çakışma riski var. `Dockerfile` `.env.example`'ı imaja kopyalıyor (secret sızma riski).
- `deploy.yaml` `--build-arg` ile secret geçiriyor; `Dockerfile` bu argümanları
  tanımlamıyor. Test/lint/migration adımı yok; sadece imaj build + push.
- Modül bağımlılıkları `DependsOn()` ile bildirilmiş ama başlatma sırası **map
  iterasyonu**; topolojik sıralama yok. Hata durumunda rollback yok.
- `ModuleInfo.Started` alanı "state enabled" ile dolduruluyor; gerçek yaşam döngüsü
  durumunu yansıtmıyor.
- Migration altyapısı yok; modüller `Models()` döndürüyor ama hiçbir yerde
  `AutoMigrate` çağrılmıyor.
- Servis katmanları somut repository'lere bağımlı; interface yok (wallet hariç).
  Test edilebilirlik ve modül izolasyonu zayıf.
- Graceful shutdown, request ID, erişim logu, panic recovery (özel), timeout,
  body limit, hata zarfı (error envelope) yok.

---

## 3. Hedef Mimari (korunacak kararlar + düzeltmeler)

Mevcut "modüler monolit" kararı doğrudur; korunur. Ancak şu sözleşmeler
netleştirilmelidir.

### 3.1 Katmanlar ve bağımlılık yönü

```
cmd/server          → bootstrap
core/               → config, logger, database, module, http, security, i18n
modules/<domain>/   → api → service → repository → model
apps/<app>/         → yalnızca core'u ve izin verilen modülleri kullanır
```

Kurallar:

- `core`, hiçbir `modules/*` paketini import etmez.
- `modules/x`, `modules/y`'yi doğrudan import etmez; yalnızca `core/contracts`
  altındaki interface'ler üzerinden konuşur (wallet, AI, notification, storage).
- `apps/*`, `modules/*`'ın internal paketlerine erişmez.
- DB erişimi yalnızca repository katmanında.
- HTTP tipleri (`echo.Context`) service katmanına sızmaz.

### 3.2 Modül sözleşmesi (yeniden yazılacak)

```go
type Module interface {
    Name() string
    Version() string
    Dependencies() []string
    Migrate(ctx context.Context, db *gorm.DB) error
    RegisterRoutes(g *echo.Group, d Dependencies) // core modülleri d'yi yok sayar
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
}
```

`Dependencies`, uygulama modüllerinin Core servislerine eriştiği arayüz kümesidir
(`contracts.Dependencies`). Core modülleri bunu kullanmaz; uygulama modülleri
(AstroScope vb.) yalnızca bu arayüzler üzerinden konuşur.
Ayrıntı: `doc/ASTROSCOPE-ENTEGRASYON.md`.

`Manager`:

- Kahn algoritması ile topolojik sıralama, cycle detection.
- `ENABLED / DISABLED / MAINTENANCE / EXPERIMENTAL / DEPRECATED` kuralları:
  - `DISABLED`: hiç yüklenmez, route yok, migration yok.
  - `MAINTENANCE`: route'lar 503 döner (read-only opsiyonel).
  - `EXPERIMENTAL`: yalnızca `APP_ENV != production`.
  - `DEPRECATED`: yüklenir, `Deprecation` header'ı ekler, log uyarısı üretir.
- Init/Start hatasında daha önce başlayanları ters sırayla `Stop` et.
- `ModuleInfo` gerçek durumu (`registered/initialized/started/failed`) taşır.

### 3.3 Bootstrap

`cmd/server/main.go` yalnızca şunu yapar:

1. Config yükle + doğrula.
2. Logger kur.
3. DB bağlan (retry + timeout).
4. Migration çalıştır (ayrı komut veya `--migrate`).
5. Modülleri config'e göre kaydet ve başlat.
6. HTTP sunucusunu başlat.
7. `SIGINT/SIGTERM` ile graceful shutdown (10 sn).

Bunun için `internal/bootstrap` (veya `core/app`) paketi oluşturulur; `main.go`
30 satırı geçmez.

### 3.4 Config

- `config/config.yaml` + `config/modules.yaml` gerçekten okunur.
- Ortam değişkenleri YAML'ı ezer (12-factor).
- `os.ExpandEnv` + `gopkg.in/yaml.v3` (gerekçeli tek yeni bağımlılık) veya
  bağımlılıksız minimal parser. Öneri: `yaml.v3`.
- Zorunlu alan doğrulaması: prod'da `JWT_SECRET`, `DB_PASSWORD` boşsa **fail-fast**.
- Tek port kaynağı: `SERVER_PORT` (varsayılan 8080). Tüm dosyalar hizalanır.
- `.env.example` gerçek değişken adlarıyla, `${...}` sözdizimi olmadan yazılır.
- `DB_NAME` tutarlı: `core_platform`.

### 3.5 HTTP katmanı

- Sürümleme: `/api/v1/...`; eski `/api/...` gerekirse 6 ay alias.
- Ortak middleware sırası:
  `RequestID → Recover → AccessLog → Timeout → BodyLimit → CORS → SecureHeaders → RateLimit → Auth (grup bazlı) → Handler`
- Hata zarfı:

```json
{ "success": false, "error": { "code": "VALIDATION_ERROR", "message": "...", "details": [] }, "request_id": "..." }
```

- `echo.HTTPError` tek noktada map'lenir; iç hata mesajları prod'da sızdırılmaz.
- `/healthz` (liveness), `/readyz` (DB + kritik bağımlılık), `/version`.
- OpenAPI 3.1: elle yazılan `api/openapi.yaml` veya kod-üretimi; CI'da validate.

### 3.6 Veritabanı ve migration

- Üretimde `AutoMigrate` **yasak** (yalnızca lokal dev'de opsiyonel).
- `migrations/` altında numaralı SQL dosyaları + `schema_migrations` tablosu.
- Basit, bağımlılıksız runner (yaklaşık 100 satır) veya `golang-migrate`.
- Her migration: ileri + geri (mümkünse), açıklama, uyumluluk notu.
- Tüm zaman damgaları UTC (`timestamptz`).
- ID stratejisi: **UUID (v7 veya `gen_random_uuid()`)**. Şu an `uint`; veri yokken
  geçiş maliyeti sıfırdır. Dışarıya sıralı ID sızdırmamak için bu karar şimdi alınmalı.
  (Alternatif: `BIGINT` + dış ID maskeleme. Tek strateji seçilip `DECISIONS.md`'ye yazılır.)
- Soft delete yalnızca gereken tablolarda; `is_deleted` + `deleted_at` + `DeletedAt`
  üçlüsü iç içe geçmiş durumda (`files` tablosu). Tek yöntem seçilir.

### 3.7 Kimlik doğrulama modeli — **ikisi de** (karar)

Tek `TokenService`, iki taşıma biçimi. Aynı kullanıcı, aynı token ailesi.

| İstemci | Taşıma | CSRF | Kullanım |
|---|---|---|---|
| Mobil / CLI / 3. parti API | `Authorization: Bearer <access>` | yok | refresh token body'de |
| Web (tarayıcı) | `access` + `refresh` **HttpOnly cookie** | var (double-submit) | SameSite=Lax |

Kurallar:

- Middleware önce `Authorization` başlığına bakar; varsa **Bearer modu**.
  Yoksa cookie'ye bakar; varsa **cookie modu**. İkisi aynı anda gelirse Bearer kazanır
  ve cookie yok sayılır (karışık mod saldırısını önler).
- **CSRF yalnızca cookie modunda** zorunludur. Bearer istekleri CSRF'den muaftır;
  böylece mevcut "CSRF varsayılan açık" hatası (S6) düzelir ve API istemcileri kırılmaz.
- Access 15 dk, refresh 7 gün (mobil) / 30 gün (web "beni hatırla").
- Refresh rotation zorunlu; her refresh eski token'ı iptal eder. Yeniden kullanım
  tespit edilirse tüm oturum ailesi iptal edilir (`token_version++`).
- Refresh token'lar DB'de **hash'lenmiş** saklanır (`refresh_tokens` tablosu:
  `id, user_id, token_hash, family_id, device, ip, expires_at, revoked_at, replaced_by`).
- Cookie: `HttpOnly`, `Secure` (prod), `SameSite=Lax`, `Path=/api/v1/auth`.
- Access token claims: `sub`, `sid`, `jti`, `role`, `scopes`, `ver`, `iat`, `exp`, `iss`, `aud`.
- `refresh` claim'i access token'da **bulunmaz**; ayrı secret ve `aud` ile ayrılır.
- Logout: cookie temizlenir + refresh ailesi iptal edilir. Bearer'da body'de refresh verilir.
- Şifre sıfırlama ve e-posta doğrulama tüm refresh ailelerini iptal eder.

### 3.8 ID stratejisi — **UUIDv7** (karar)

- Tüm yeni tablolar: `id uuid PRIMARY KEY` (PostgreSQL `uuid` tipi).
- Üretim **uygulama tarafında** UUIDv7 (`github.com/google/uuid` veya
  `gofrs/uuid`). Böylece `pgcrypto`/eklenti bağımlılığı yok ve ID, INSERT'ten önce bilinir
  (idempotency ve olay yayını için gerekli).
- UUIDv7 zaman sıralıdır → B-tree indeks lokalitesi iyi; UUIDv4'ün rastgele yazma
  cezası yok.
- Dışarıya açılan her ID UUID'dir. `bigserial` kullanılmaz. İç join tablolarında da UUID.
- Neden şimdi: veri yokken geçiş maliyeti sıfır. Sonradan geçiş, tüm FK'ler ve
  istemci sözleşmeleri nedeniyle çok pahalıdır.
- Alternatif reddedildi: `BIGINT` + dış maskeleme → iki ID sistemi, sızıntı riski.

### 3.9 Ödeme mimarisi — **Stripe + PayTR + Havale/EFT** (karar)

Üç sağlayıcı, tek `PaymentProvider` arayüzü ve tek durum makinesi.

```go
type PaymentProvider interface {
    Name() string
    Capabilities() Capabilities        // card, wallet, manual, refund, recurring
    CreateCheckout(ctx, CheckoutInput) (CheckoutResult, error)
    ParseWebhook(r *http.Request) (Event, error)   // imza/hash doğrulama burada
    Refund(ctx, RefundInput) (RefundResult, error)
}
```

| Sağlayıcı | Yöntem | Para birimi | Tekrar ödeme | Entitlement kaynağı |
|---|---|---|---|---|
| **Stripe** | Checkout Session (kart, Apple/Google Pay) | USD/EUR/TRY* | abonelik | `checkout.session.completed`, `invoice.paid` |
| **PayTR** | iframe / redirect (kart, yerel) | TRY | abonelik (tekrarlayan) | PayTR callback hash |
| **Havale/EFT** | Manuel banka transferi | TRY | yok | Admin onayı |

\* Stripe'ın TRY desteği hesap ülkesine bağlıdır; canlıya çıkmadan test edilir.
Desteklenmiyorsa TRY işlemleri PayTR/havale üzerinden yürür.

#### Ortak şema

```
products(id, code, name, amount_minor, currency, interval, app_id, active)
orders(id, user_id, product_id, amount_minor, currency, provider, status,
       idempotency_key UNIQUE, expires_at, created_at)
payments(id, order_id, provider, provider_ref, status, amount_minor, currency,
         paid_at, failure_code, raw_payload jsonb)
payment_events(id, provider, event_id UNIQUE, order_id, type, payload jsonb,
               processed_at)                    -- idempotency
entitlements(id, user_id, product_id, source, starts_at, ends_at, active)
refunds(id, payment_id, amount_minor, provider_ref, status)
```

- Tüm tutarlar **minor unit** (`int64` kuruş/cent). `float` yasak.
- Para birimi ve tutar **asla** istemciden alınmaz; `product_id` → sunucu fiyatı.
- `orders.idempotency_key` unique; aynı istek ikinci kez yeni sipariş açmaz.
- Durum makinesi:
  `created → pending → paid → fulfilled` / `failed` / `expired` / `refunded`.
  Geçişler yalnızca servis katmanında, transaction içinde.

#### Stripe akışı

1. `POST /api/v1/payments/checkout` → `{provider:"stripe", product_id}`.
2. Sunucu `orders` kaydı + Stripe Checkout Session oluşturur, `client_reference_id=order_id`.
3. Kullanıcı Stripe sayfasında öder; başarı sayfası **entitlement vermez**.
4. Webhook `POST /api/v1/payments/webhook/stripe`:
   imza (`Stripe-Signature`) → `payment_events` idempotency → transaction →
   `payments.status=paid` → `entitlements` → `order.status=fulfilled`.
5. Tekrar gelen aynı `event.id` işlenmez (200 döner).

#### PayTR akışı

1. `POST /api/v1/payments/checkout` → `{provider:"paytr", product_id}`.
2. Sunucu PayTR token isteği (merchant_id, merchant_key, merchant_salt,
   `merchant_oid=order_id`, tutar, sepet, başarılı/başarısız URL).
3. Kullanıcı PayTR iframe/redirect'e yönlendirilir.
4. PayTR **sunucudan sunucuya** callback atar: `POST /api/v1/payments/webhook/paytr`
   (form-encoded). Doğrulama:
   `hash = base64(HMAC-SHA256(merchant_oid + merchant_salt + status + total_amount,
   merchant_key))` — sabit zamanlı karşılaştırma.
5. `status=success` ise entitlement verilir. Callback gövdesi **asla** yeterli sayılmaz;
   PayTR'nin durum sorgu API'si ile de çapraz kontrol edilir.
6. Callback'e her durumda `OK` düz metni dönülür (aksi halde PayTR tekrar dener).

#### Havale/EFT akışı

1. `POST /api/v1/payments/checkout` → `{provider:"bank_transfer", product_id}`.
2. `order` `pending` + **benzersiz referans kodu** üretilir (ör. `AS-7F3K2Q`).
3. Kullanıcıya banka bilgileri, tutar, referans ve son ödeme tarihi (`expires_at`)
   gösterilir.
4. Kullanıcı dekont yükler (`POST /api/v1/payments/:id/receipt`, filestorage).
5. Admin `POST /api/v1/admin/payments/:id/confirm` ile onaylar → entitlement.
   Reddederse `failure_reason` ile `failed`.
6. `expires_at` geçince otomatik `expired`. Onay olmadan entitlement **verilmez**.
7. Mutabakat: günlük banka ekstresi ile `order` referansları eşleştirilir; eşleşmeyen
   ödemeler raporlanır.

#### Güvenlik ve uyum

- Kart verisi hiçbir noktada uygulamaya girmez (Stripe/PayTR hosted sayfa). PCI kapsamı
  en aza iner.
- Webhook route'ları auth middleware'inden **muaf**, kendi imza doğrulamasına sahip,
  ayrı rate limit ve IP logu ile çalışır.
- `provider_ref`, `raw_payload`, `event_id` saklanır; loglarda kart/PII yok.
- İade: Stripe/PayTR API üzerinden; havale iade manuel kayıt + admin onayı.
- Para birimi dönüşümü yok; ürün fiyatı tek para biriminde tanımlanır.

#### Neden üçü birden?

- Stripe: uluslararası kart, abonelik, düşük operasyon yükü.
- PayTR: Türkiye'de yerel kart/taksit ve TRY tahsilat zorunluluğu.
- Havale: kartı olmayan/kurumsal kullanıcı, düşük hacim, yüksek tutar.

Üçü de aynı `orders`/`entitlements` modeline bağlandığı için uygulama kodu
sağlayıcıdan bağımsızdır. Yeni sağlayıcı eklemek = yeni adapter + test.

---

## 4. Modül Bazında Karar Tablosu

| Modül | Mevcut durum | Karar | Gerekçe |
|---|---|---|---|
| `core/config` | Sadece env; YAML okumuyor | **Yeniden yaz** | Tüm yapılandırma temeli |
| `core/logger` | Çalışır | **Küçük düzeltme** | Zap sarmalayıcı yeterli |
| `core/database` | Bağlantı var, migration yok | **Genişlet** | Runner, retry, metrik |
| `core/module` | Derlenmiyor, kullanılmıyor | **Yeniden yaz** | Platformun kalbi |
| `core/api` | 2 ayrı router, sürüm yok | **Yeniden yaz** | Tek router + envelope |
| `core/health` | Basit | **Düzelt** | Liveness/readiness ayrımı |
| `core/security` | Kısmen var, bağlı değil | **Birleştir + düzelt** | CORS/CSRF/rate limit |
| `core/middleware` | Dağınık, biri derlenmiyor | **`core/security` içine taşı** | Tek sorumluluk |
| `core/localization` | Metinler bozuk, loader eski | **Yeniden yaz** | `embed` + yeniden çeviri |
| `core/version` | Var | **Küçük düzeltme** | ldflags ile bağla |
| `modules/auth` | Kısmen gerçek | **Yeniden yaz** | Refresh/logout/RBAC yok |
| `modules/users` | Auth ile aynı tablo/model | **Auth ile birleştir** | Çift model/repo |
| `modules/wallet` | Yarım, transaction yok | **Yeniden yaz** | Para/jeton invariant'ı |
| `modules/premium` | Mock route + gerçek servis karışık | **Yeniden yaz** | Ödeme entegrasyonu yok |
| `modules/payments` | **Yok** (config'te var, kodda yok) | **Yeni yaz** | Stripe + PayTR + havale |
| `modules/rewarded_ads` | Mock, güvensiz | **Yeniden yaz veya dondur** | SSV doğrulaması şart |
| `modules/notifications` | Gerçek CRUD, gönderim mock | **Tamamla** | Outbox + adapter |
| `modules/filestorage` | Derlenmiyor, güvenlik açıkları | **Düzelt + sertleştir** | `net/http`, SVG, authz |
| `modules/search` | Panik riski, pagination yok | **Düzelt** | Type assertion + SQL |
| `modules/community` | Tamamen mock | **Yeniden yaz** | DB yok |
| `modules/analytics` | Tamamen mock | **Yeniden yaz** | Gerçek aggregate |
| `modules/ai_gateway` | OpenAI istemcisi var | **Genişlet** | Interface, retry, kota |
| `modules/prompt_engine` | JSON parse TODO, template riski | **Düzelt** | `json` + sandbox |
| `modules/context_engine` | CRUD var, authz yok | **Düzelt** | Sahiplik kontrolü |
| `modules/ai_copilot` | Gateway'e bağımlı, mock | **Düzelt** | Interface üzerinden |
| `modules/lifeweave` | Mock kelime eşleştirme | **Core'dan çıkar** | Uygulamaya özel |
| `backend/astroscope` | Tamamen mock/rastgele | **Ayrı uygulama deposuna taşı** | Yol haritası kuralı |
| `modules/community`, `analytics`, `lifeweave` | — | **3-uygulama kuralı ile yeniden değerlendir** | Core şişmesin |

> Yol haritasındaki "bir özellik en az 3 uygulamada kullanılacaksa Core'a girer"
> kuralı uygulanmamış. `lifeweave`, `community`, `rewarded_ads` büyük olasılıkla
> uygulama seviyesine ait. Bu, core'un bakım yükünü ciddi azaltır.

---

## 5. Faz Planı

Her fazın sonunda **kapı (gate)** vardır; kapı geçilmeden sonraki faza geçilmez.
Süreler 1 geliştirici + AI varsayımıyladır.

### Faz 0 — Depo Kurtarma (1–2 gün) — CRITICAL

Amaç: `go build ./...` yeşil, `gofmt` temiz, `go vet` temiz.

- [ ] Tüm `.go` dosyalarını UTF-8'e normalize et; yorumlardaki mojibake'i temizle.
      **Karar:** kaynak kodda ASCII yorum kullan; Türkçe metinler yalnızca locale JSON'da.
- [ ] `gofmt -w .`
- [ ] 53 dosyada importları `core-platform/modules/...` yap.
- [ ] 13 dosyada struct tag'lerini elle onar; bozuk `//` yorumlarını tag dışına al.
- [ ] `core/config`'e `ModulesConfig` tipini ekle (veya manager'ı yeni yapıya çevir).
- [ ] Eksik stdlib importlarını ekle (`io`, `net/http`, `time`), eksik model alanlarını
      düzelt (`VerifiedAt`).
- [ ] Kök `main` dosyalarını sil veya `_legacy/` altına taşı:
      `dosya.go`, `test_api.go`, `test_community.go`, `test_simple.go`.
      (Git geçmişinde kalır; mock route'lar üretim kodu değildir.)
- [ ] `cmd/server/main.go.backup`, `.backup2`, `git-filter-repo`, `server.exe` sil.
- [ ] `.gitignore`: `uploads/`, `bin/`, `*.exe`, `*.backup*`, `.env.*` (örnek hariç).
- [ ] `go mod tidy`; `go.mod` ile Docker Go sürümünü **tek sürüme** sabitle.
- [ ] Kök `README.md` ekle (kurulum, çalıştırma, test, mimari özeti).
- [ ] `CHANGELOG` ve `core/version` sürümünü gerçek duruma çek (`0.1.0-dev`).

**Kapı:** `gofmt -l .` boş · `go build ./...` başarılı · `go vet ./...` başarılı ·
`go test ./...` (test yoksa `ok`/`no test files`) · commit.

---

### Faz 1 — Çekirdek Platform (3–5 gün) — HIGH

Amaç: Platform gerçekten ayağa kalksın; modül yaşam döngüsü çalışsın.

> **Başlangıç teslim edildi:** `contracts/` paketi ve `cmd/core` referans bootstrap'ı
> yazıldı (derlenir + testleri geçer). Ayrıntı: `doc/FAZ1-CONTRACTS-BOOTSTRAP.md`.
> Kalan iş: YAML config, core modül adaptörleri ve `cmd/server` geçişi.

- [ ] `config`: YAML + env override + doğrulama; `config.example.yaml` ile uyum.
- [ ] `module.Manager`: topolojik sıra, cycle detection, rollback, gerçek durumlar.
- [ ] `bootstrap`: config → logger → DB → migrate → modules → HTTP → shutdown.
- [ ] Migration runner + `migrations/0001_init.sql` (users, schema_migrations).
- [ ] `core/api`: `/api/v1`, tek router, hata zarfı, request ID, access log, timeout,
      body limit, recover.
- [ ] `core/health`: `/healthz`, `/readyz`, `/version`.
- [ ] `core/security`: CORS (açık origin listesi), güvenlik header'ları (prod'da HSTS),
      rate limiter (TTL/temizlik; çok instance için Redis opsiyonu), CSRF kararı.
- [ ] `core/localization`: `embed.FS`, eksik anahtar fallback, dil dosyalarını yeniden yaz.
- [ ] `contracts/` paketi: `Platform`, `Identity`, `Wallet`, `Entitlements`, `AI`,
      `Storage`, `Notifications`, `Payments`, `Module`, `Dependencies`, `Run`.
      AstroScope bu pakete bağlanacağı için **Faz 1 çıktısıdır**
      (bkz. `doc/ASTROSCOPE-ENTEGRASYON.md`).
- [ ] `contracts.Run`: uygulama modüllerini Core modülleriyle birlikte başlatan
      bootstrap; route öneki `/api/v1/apps/<app>`.
- [ ] `Makefile`: `run`, `build`, `test`, `lint`, `fmt`, `migrate`, `docker-*` (Windows
      uyumlu; `rmdir` yerine taşınabilir komutlar).
- [ ] Docker/compose: tek port, non-root, `.env` kopyalama yok, healthcheck,
      kaynak mount yok, `depends_on: condition: service_healthy`.
- [ ] İlk gerçek testler: config, module sıralama, health, migration runner.

**Kapı:** `docker compose up` ile Postgres + app ayağa kalkar; `/readyz` 200;
`go test ./...` yeşil; modül durumları `/api/v1/admin/modules` ile görünür.

---

### Faz 2 — Kimlik ve Yetkilendirme (3–5 gün) — HIGH

Amaç: Tek kullanıcı modeli, güvenli auth, tüm handler'larda gerçek kullanıcı.

- [ ] `users`/`auth` birleştir: tek `User` modeli, tek repository, tek servis.
- [ ] Şema: `id (uuid)`, `email citext unique`, `password_hash`, `name`, `locale`,
      `avatar_url`, `role`, `status`, `email_verified_at`, `last_login_at`,
      `token_version`, `created_at`, `updated_at`, `deleted_at`.
- [ ] Şifre: Argon2id (veya bcrypt cost 12); politika + uzunluk sınırı.
- [ ] JWT: access 15 dk, refresh 7 gün; ayrı secret; `jti`, `token_version`;
      refresh rotation + hash'lenmiş saklama + reuse detection.
- [ ] Endpoint'ler:
      `POST /auth/register`, `/login`, `/refresh`, `/logout`,
      `GET/PATCH /me`, `POST /auth/forgot-password`, `/reset-password`,
      `POST /auth/verify-email`.
- [ ] `AuthMiddleware`: token doğrula, kullanıcıyı context'e koy, `status=active` kontrol.
- [ ] `RequireRole("admin")`; `GET /users`, `DELETE /users/:id` artık korumalı.
- [ ] Tüm handler'lardan `userID = 1` kaldırılır.
- [ ] Girdi doğrulama: `validator/v10` gerçekten bağlanır; her request DTO'su doğrulanır.
- [ ] Login/register/refresh için IP + e-posta bazlı rate limit.
- [ ] E-posta gönderimi için `Mailer` interface (SMTP/Resend); dev'de MailHog.
- [ ] Testler: register/login/refresh rotation/reuse/logout/authorization.

**Kapı:** Kayıt → giriş → korumalı `/me` → refresh → logout akışı E2E geçer;
admin olmayan kullanıcı admin endpoint'inde 403 alır.

---

### Faz 3 — İş Çekirdeği: Wallet, Premium, Ödeme (8–12 gün) — HIGH/CRITICAL

**Ödeme sağlayıcıları (karar): Stripe + PayTR + Havale/EFT.** Ayrıntı: Bölüm 3.9.

- [ ] **Wallet ledger:** `wallet_transactions` append-only; bakiye ya ledger'dan
      türetilir ya da `SELECT ... FOR UPDATE` ile güncellenir. `idempotency_key` zorunlu.
- [ ] `POST /wallet/earn` **kaldırılır**; jeton yalnızca sunucu olaylarıyla
      (reklam SSV, günlük bonus, admin düzeltmesi) verilir.
- [ ] Günlük bonus: `unique(user_id, date)` ile gerçekten bir kez.
- [ ] Harcama: AI kullanımı, premium rapor; yetersiz bakiye 409.
- [ ] `payments` modülünü sıfırdan kur: `products`, `orders`, `payments`,
      `payment_events`, `entitlements`, `refunds`.
- [ ] `PaymentProvider` arayüzü + `internal/provider/{stripe,paytr,manual}`.
- [ ] **Stripe adapter:** Checkout Session, `Stripe-Signature` doğrulama,
      `checkout.session.completed` + `invoice.paid`, abonelik iptali, iade.
- [ ] **PayTR adapter:** token isteği, iframe/redirect, hash doğrulama
      (HMAC-SHA256 + sabit zamanlı), durum sorgu API'si ile çapraz kontrol,
      tekrarlayan ödeme, iade.
- [ ] **Havale/EFT adapter:** referans kodlu sipariş, son ödeme tarihi, dekont yükleme,
      admin onay/red, otomatik `expired`, günlük mutabakat raporu.
- [ ] Webhook idempotency: `payment_events.provider + event_id` unique;
      aynı olay ikinci kez işlenmez, 200 döner. Webhook route'ları auth'tan muaf,
      kendi imza doğrulaması + ayrı rate limit.
- [ ] Entitlement yalnızca doğrulanmış sunucu olayından verilir; client redirect'i
      veya "başarılı" query param'ı **asla** yeterli değil.
- [ ] `premium` modülü `entitlements` üzerine taşınır; fiyat istemciden alınmaz.
- [ ] Rewarded ads: SSV callback + imza + `unique(user_id, ad_id, provider)`;
      aksi halde modül **dondurulur** (route açılmaz).
- [ ] Testler: çift harcama, eşzamanlı harcama, webhook tekrarı, imza reddi,
      tutar uyuşmazlığı, süre dolumu, iade, iptal/yenileme.
- [ ] Stripe test kartları + PayTR test merchant + havale admin akışı için E2E.

**Kapı:** Üç sağlayıcının her biri test modunda: sipariş → ödeme → webhook →
entitlement → wallet/rapor erişimi uçtan uca çalışır. Sahte "başarılı" isteği
entitlement vermez. Yarış testinde bakiye negatife düşmez.

---

### Faz 4 — Dosya, Bildirim, Arama, Topluluk, Analitik (4–6 gün) — MEDIUM/HIGH

- [ ] **Filestorage:** `net/http` importu; SVG yasak veya sanitize; içerik doğrulama;
      kullanıcı sahipliği; erişimde authz; imzalı URL; `baseURL` config'ten;
      S3 uyumlu `Storage` interface (lokal + S3).
- [ ] **Notifications:** outbox tablosu, retry, template, tercih kontrolü;
      e-posta/push adapter; `IsSent` gerçekten güncellenir.
- [ ] **Search:** type assertion panics düzelt; `offset` gerçekten uygulanır;
      `ILIKE` yerine `pg_trgm` + GIN index; `deleted_at` tutarlılığı; sonuç skorlama.
- [ ] **Community:** mock handler'lar DB'ye taşınır; moderasyon, anonimlik,
      rate limit, sayfalama.
- [ ] **Analytics:** mock dashboard gerçek aggregate'e çevrilir; indeksler;
      PII içermeyen olay şeması.
- [ ] Testler: dosya yükleme sınırları, yetkisiz erişim, arama injection/panic.

**Kapı:** Dosya yükleme/indirme yetkili; bildirim outbox'ı yeniden deniyor;
arama 10k kayıtta < 200 ms; topluluk CRUD gerçek DB'de.

---

### Faz 5 — AI Katmanı (4–6 gün) — MEDIUM/HIGH

- [ ] `AIProvider` interface: `Chat(ctx, req) (resp, usage, error)`.
- [ ] OpenAI adapter (mevcut), Anthropic, Gemini; her biri timeout + retry (jitter) +
      circuit breaker.
- [ ] Kota: kullanıcı/gün/model bazlı; wallet entegrasyonu; aşımda 429.
- [ ] Cache: `prompt_hash + model + provider`; hata cache'lenmez; TTL config'ten.
- [ ] Maliyet tablosu + kullanım kaydı; loglarda prompt/PII maskelenir.
- [ ] Prompt engine: `json.Marshal/Unmarshal`; `missingkey=error`; şablon içinde
      keyfi fonksiyon yok; şablon sahipliği ve sürümleme.
- [ ] Context engine: oturum sahipliği, token bütçesi, özetleme, saklama süresi.
- [ ] Copilot: gateway interface'i üzerinden; mock yanıt yok.
- [ ] Testler: sağlayıcı hataları, timeout, kota, cache, şablon injection.

**Kapı:** Sağlayıcı kesintisinde uygulama 503/fallback ile ayakta; kullanıcı kotası
aşılamıyor; AI maliyeti kullanıcı bazında izlenebiliyor.

---

### Faz 6 — Uygulama: AstroScope (5–10 gün) — HIGH

**Karar: ayrı depo.** AstroScope, Core'u Go modülü olarak tüketir, kendi modüllerini
`contracts.Run` ile kaydeder, aynı PostgreSQL'de `astroscope` şemasını kullanır.
Ayrıntılı sözleşme, `main.go` örneği, şema/API/auth/ödeme/AI entegrasyonu ve geçiş
adımları: **`doc/ASTROSCOPE-ENTEGRASYON.md`**.

Core tarafında bu fazın ön koşulu:

- [ ] `contracts/` paketi: `Platform`, `Identity`, `Wallet`, `Entitlements`, `AI`,
      `Storage`, `Notifications`, `Payments`, `Module`, `Run`.
- [ ] `Run`: config → logger → DB → migration → core modülleri → app modülleri →
      HTTP → graceful shutdown.
- [ ] AstroScope'un `core/*` veya `modules/*` internal paketlerini import etmediğini
      denetleyen CI kuralı (`depguard` veya `go list` kontrolü).
- [ ] Uygulama route öneki: `/api/v1/apps/astroscope/...`.

AstroScope tarafında:

- [ ] Ayrı depo + `go.work` (lokal) / sürümlenmiş `require core-platform` (prod).
- [ ] `math/rand` tamamen kaldırılır.
- [ ] Gerçek efemeris (Swiss Ephemeris / lisansı uygun Go kütüphanesi) — lisans kontrolü.
- [ ] Doğum haritası, transit, progression, solar return, synastry gerçek hesaplama.
- [ ] Bilinen 5–10 referans harita ile altın testler (deterministik, `Clock` enjekte).
- [ ] Burç/gezegen/ev etiketleri i18n; Türkçe metinler yeniden yazılır.
- [ ] Yasal uyarı (eğlence/kişisel gelişim amaçlı) ve KVKK aydınlatma.
- [ ] Wallet debit + premium entitlement (`astroscope.pro`) entegrasyonu.
- [ ] Mock `summary` üretimi AI gateway'e taşınır (kotalı, cache'li).
- [ ] Mevcut `backend/astroscope` bu depodan silinir (git geçmişinde kalır).

**Kapı:** Aynı doğum verisi her zaman aynı sonucu verir; altın testler geçer;
rastgelelik yok; contract test Core'un desteklenen sürümlerinde geçer.

---

### Faz 7 — Üretim Hazırlığı (4–6 gün) — HIGH

- [ ] CI: `gofmt -l`, `go vet`, `staticcheck`, `govulncheck`, test (Postgres service),
      build, Docker build, Trivy scan.
- [ ] Deploy: imajı runtime env ile çalıştır (build-arg ile secret yok);
      migration job → rollout; rollback stratejisi.
- [ ] Secrets: üretimde secret manager / platform env; repoda sadece `.env.example`.
- [ ] TLS + reverse proxy; HSTS yalnızca HTTPS'te; güvenli cookie ayarları.
- [ ] Gözlemlenebilirlik: JSON log + `request_id` + `user_id`; `/metrics` (Prometheus);
      hata izleme (Sentry vb.); yavaş sorgu logu.
- [ ] Yedekleme: günlük `pg_dump`, saklama, **restore tatbikatı**.
- [ ] Performans: kritik sorgular `EXPLAIN`, indeksler, k6 smoke.
- [ ] Güvenlik: dependency taraması, rate limit, admin audit log, KVKK silme akışı.
- [ ] Runbook: deploy, rollback, DB restore, incident.

**Kapı:** Tag → CI yeşil → staging deploy → smoke → prod deploy → `/readyz` 200;
geri yükleme tatbikatı başarılı.

---

### Faz 8 — Dokümantasyon ve Hafıza Temizliği (1–2 gün) — LOW

- [ ] `_KALICI-PROJE-HAFIZASI/` bu projeye ait değil; sil veya `.ai/` altında
      yeniden üret. `FILE_INDEX.json` başka depoyu işaret ediyor.
- [ ] `doc/ai-project-starter/` Next.js/Stripe odaklı; Go platformu için
      `docs/ARCHITECTURE.md`, `docs/SECURITY.md`, `docs/DATABASE.md`,
      `docs/API.md`, `docs/DECISIONS.md`, `docs/DEVELOPMENT.md` olarak uyarla.
- [ ] `YOL HARİTASI.txt` ile gerçek durum çelişkisi giderilir; tek yol haritası.
- [ ] `CHANGELOG.md` gerçek sürümlerle güncellenir.
- [ ] Her modül için kısa `README.md` (sorumluluk, endpoint, tablo, bağımlılık).

---

## 6. Öncelikli 12 Commit (atomik)

1. `chore: normalize formatting, imports, struct tags; restore green build`
2. `chore: remove legacy mock mains, backups, binaries; fix gitignore`
3. `feat(config): YAML + env loading with fail-fast validation`
4. `feat(module): topological lifecycle manager with rollback`
5. `feat(bootstrap): wire config, db, migrations, modules, http, shutdown`
6. `feat(db): SQL migration runner + baseline schema`
7. `feat(http): v1 router, error envelope, request id, health/readiness`
8. `feat(security): cors, headers, rate limit, validation`
9. `refactor(identity): merge auth+users; jwt refresh rotation; rbac`
10. `feat(wallet): ledger with row locking and idempotency`
11. `feat(payments): provider abstraction + webhook idempotency + premium entitlement`
12. `ci: fmt, vet, staticcheck, vulncheck, tests, docker, scan`

---

## 7. Test Stratejisi

| Katman | Araç | Kapsam |
|---|---|---|
| Unit | `testing` + `testify` (opsiyonel) | servis kuralları, şablon, config |
| Repository | gerçek Postgres (compose profile) | transaction, unique, lock |
| HTTP | `httptest` + Echo | authz, validation, hata zarfı |
| E2E | `testcontainers-go` veya compose | register→login→işlem |
| Yük | k6 | health, auth, arama |
| Güvenlik | `govulncheck`, Trivy, `gosec` | bağımlılık + kod |

Kurallar:

- Yeni özellik = en az 1 başarı + 1 hata testi.
- Finansal invariant'lar için yarış testi (`-race`).
- CI'da `-race`, coverage eşiği başlangıçta %40, olgunlukta %70.
- Testler gerçek DB kullanır; SQLite'a kaçış yok (JSONB, `ILIKE`, `FOR UPDATE`).

---

## 8. Güvenlik Bulguları (öncelik sırası)

| # | Bulgu | Dosya | Etki | Önlem |
|---|---|---|---|---|
| S1 | Tüm handler'larda `userID = 1` | 15+ dosya | Herkes herkesin verisine erişir | Auth middleware |
| S2 | Admin endpoint'leri korumasız | `modules/users/api` | Kullanıcı silme/listeleme | RBAC |
| S3 | `POST /wallet/earn` istemciden jeton üretiyor | `wallet_handler.go` | Ekonomi çöker | Endpoint kaldır |
| S4 | `GET /users` tüm e-postaları döndürür | `user_handler.go` | Veri sızıntısı | Admin + DTO |
| S5 | CORS `*` + credentials | `core/security/cors.go` | Origin doğrulama yok | Açık liste |
| S6 | CSRF varsayılan açık + API | `core/security/security.go` | API istemcileri kırılır | Cookie akışında aç |
| S7 | Rate limiter sınırsız map | `rate_limiter.go` | Bellek DoS | TTL temizlik/Redis |
| S8 | XSS middleware JSON'u HTML-escape ediyor | `core/middleware/xss.go` | Veri bozulması + `io` yok | Kaldır; çıktıda escape |
| S9 | SVG upload'a izin | `file_service.go` | Stored XSS | Yasak/sanitize |
| S10 | Dosya indirmede sahiplik kontrolü yok | `file_handler.go` | IDOR | Authz |
| S11 | JWT access/refresh aynı secret | `auth_service.go` | Refresh sızarsa access üretir | Ayrı secret |
| S12 | Refresh rotation/iptal yok | `auth_service.go` | Çalınan token süresiz | Rotation + jti |
| S13 | İç hata mesajları client'a | tüm handler'lar | Bilgi sızıntısı | Merkezi map |
| S14 | DB varsayılan şifresi `postgres` | `core/config` | Prod'a sızabilir | Fail-fast |
| S15 | Docker imajına `.env.example` | `Dockerfile` | Secret alışkanlığı | Kaldır |
| S16 | Login/register rate limit yok | auth | Brute force | IP+e-posta limit |
| S17 | Prompt template keyfi fonksiyon | `prompt_service.go` | Injection | Kısıtlı şablon |
| S18 | `text/template` çıktısı HTML'e basılırsa XSS | prompt engine | XSS | `html/template`/kaçış |

---

## 9. API Sözleşmesi (hedef v1)

```
GET    /healthz
GET    /readyz
GET    /version

POST   /api/v1/auth/register
POST   /api/v1/auth/login
POST   /api/v1/auth/refresh
POST   /api/v1/auth/logout
POST   /api/v1/auth/forgot-password
POST   /api/v1/auth/reset-password
POST   /api/v1/auth/verify-email

GET    /api/v1/me
PATCH  /api/v1/me
DELETE /api/v1/me

GET    /api/v1/wallet
GET    /api/v1/wallet/transactions
POST   /api/v1/wallet/spend          (idempotency-key)
POST   /api/v1/wallet/daily-bonus

GET    /api/v1/premium/plans
GET    /api/v1/premium/me
POST   /api/v1/premium/checkout
POST   /api/v1/premium/cancel

GET    /api/v1/payments/providers                  (stripe, paytr, bank_transfer)
POST   /api/v1/payments/checkout                   (product_id, provider)
GET    /api/v1/payments/orders/:id
POST   /api/v1/payments/:id/receipt                (havale dekontu)
POST   /api/v1/payments/webhook/stripe             (imza; auth muaf)
POST   /api/v1/payments/webhook/paytr              (hash; auth muaf)
POST   /api/v1/admin/payments/:id/confirm          (havale onayı)
POST   /api/v1/admin/payments/:id/reject

GET    /api/v1/apps/astroscope/charts
POST   /api/v1/apps/astroscope/natal
GET    /api/v1/apps/astroscope/charts/:id
GET    /api/v1/apps/astroscope/transit
POST   /api/v1/apps/astroscope/synastry
POST   /api/v1/apps/astroscope/charts/:id/interpret

POST   /api/v1/files
GET    /api/v1/files
GET    /api/v1/files/:id
DELETE /api/v1/files/:id

GET    /api/v1/notifications
GET    /api/v1/notifications/unread-count
POST   /api/v1/notifications/:id/read
POST   /api/v1/notifications/read-all
GET    /api/v1/notifications/preferences
PUT    /api/v1/notifications/preferences

GET    /api/v1/search?q=&category=&limit=&offset=
GET    /api/v1/search/history
GET    /api/v1/search/popular

POST   /api/v1/ai/chat               (kota + rate limit)
GET    /api/v1/ai/providers

POST   /api/v1/prompts
GET    /api/v1/prompts
GET    /api/v1/prompts/:name
DELETE /api/v1/prompts/:name
POST   /api/v1/prompts/execute

GET    /api/v1/context/sessions
POST   /api/v1/context/sessions
GET    /api/v1/context/sessions/:id
DELETE /api/v1/context/sessions/:id
POST   /api/v1/context/sessions/:id/messages

GET    /api/v1/admin/modules
PATCH  /api/v1/admin/modules/:name
GET    /api/v1/admin/users
GET    /api/v1/admin/users/:id
DELETE /api/v1/admin/users/:id
```

Kurallar:

- Tüm liste endpoint'leri `limit` (max 100) + `cursor`/`offset` + toplam.
- Tüm yazma endpoint'leri `Idempotency-Key` destekler (ödeme, wallet, reklam).
- Hata kodları: `VALIDATION_ERROR`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`,
  `CONFLICT`, `RATE_LIMITED`, `PROVIDER_ERROR`, `INTERNAL`.

---

## 10. Kararlar

### 10.1 Verilmiş kararlar (2026-10-24)

| # | Karar | Sonuç | Ayrıntı |
|---|---|---|---|
| D1 | V1 uygulaması | **AstroScope** | Faz 6 |
| D2 | AstroScope yeri | **Ayrı depo**, Core'u Go modülü olarak tüketir | `doc/ASTROSCOPE-ENTEGRASYON.md` |
| D3 | Ödeme sağlayıcıları | **Stripe + PayTR + Havale/EFT** | Bölüm 3.9 |
| D4 | Auth modeli | **İkisi de**: Bearer (mobil/API) + HttpOnly cookie + CSRF (web) | Bölüm 3.7 |
| D5 | ID stratejisi | **UUIDv7**, uygulama tarafında üretilir | Bölüm 3.8 |

Bu beş karar `docs/DECISIONS.md` içine ADR olarak taşınacaktır (Faz 8).

### 10.2 Hâlâ açık konular

| # | Soru | Seçenekler | Etki |
|---|---|---|---|
| D6 | Para birimi/bölge | TRY / SGD / çoklu | Planlar, vergi |
| D7 | AI sağlayıcıları | OpenAI / Anthropic / Gemini | Kota ve maliyet |
| D8 | Dil önceliği | tr / en / en-SG | Çeviri yatırımı |
| D9 | Efemeris lisansı | Swiss Ephemeris / alternatif | Yasal + teknik |
| D10 | Core'a girecek modüller | community/lifeweave/ads? | Bakım yükü |
| D11 | Deploy hedefi | VPS/Docker / cloud | CI-CD tasarımı |
| D12 | KVKK/GDPR kapsamı | TR / SG / AB | Saklama, silme, log |
| D13 | Stripe TRY desteği | hesap ülkesine bağlı test edilecek | TRY akışı PayTR'ye düşebilir |
| D14 | Havale mutabakat sahibi | finans ekibi / admin paneli | operasyon |

Bu kararlar `docs/DECISIONS.md` içinde ADR olarak kaydedilmelidir.

---

## 11. Riskler

| Risk | Olasılık | Etki | Azaltma |
|---|---|---|---|
| Kurtarma sırasında davranış kaybı | Orta | Orta | Önce build; davranış değişikliği ayrı commit |
| Mock kodun "çalışıyor" sanılması | Yüksek | Yüksek | Mock route'ları sil; E2E ile kanıtla |
| Şema kararlarının geç alınması | Yüksek | Yüksek | Faz 1'de ID/soft-delete kararı |
| Ödeme entegrasyonunun gecikmesi | Orta | Yüksek | Sağlayıcı abstraction + test modu |
| Kapsam şişmesi (15 modül) | Yüksek | Yüksek | 3-uygulama kuralı; modül dondurma |
| Eski dokümanların yanıltması | Yüksek | Orta | Tek kaynak; staleleri sil |
| Lisans/regülasyon (efemeris, ödeme) | Orta | Yüksek | Erken hukuk/lisans kontrolü |

---

## 12. "V1 Tamamlandı" Tanımı

Aşağıdakilerin tamamı gerçekleştiğinde:

1. `gofmt`, `go vet`, `staticcheck`, `go test -race`, `govulncheck` yeşil.
2. Tek komutla (`docker compose up`) platform + Postgres ayağa kalkıyor.
3. Kayıt → giriş → e-posta doğrulama → refresh → logout akışı çalışıyor.
4. Kullanıcı yalnızca kendi verisine erişiyor; admin rolleri ayrı.
5. Wallet'ta çift harcama/negatif bakiye imkânsız; tüm hareketler ledger'da.
6. Test modunda gerçek ödeme → webhook → premium erişimi çalışıyor.
7. AI çağrıları kotalı, izlenebilir, hata durumunda güvenli.
8. Dosya yükleme/indirme yetkili ve doğrulanmış.
9. CI tag'de imaj üretiyor, staging'e çıkıyor, smoke geçiyor.
10. Yedekten geri dönüş tatbikatı yapılmış.
11. OpenAPI, runbook, ADR'ler ve modül README'leri güncel.
12. Repoda sır yok; `.env.example` eksiksiz.

---

## 13. Özet Efor

| Faz | Süre | Risk |
|---|---|---|
| 0 — Kurtarma | 1–2 gün | CRITICAL |
| 1 — Çekirdek | 3–5 gün | HIGH |
| 2 — Kimlik | 3–5 gün | HIGH |
| 3 — Wallet/Premium/Ödeme (Stripe+PayTR+Havale) | 8–12 gün | CRITICAL |
| 4 — Dosya/Bildirim/Arama/Topluluk/Analitik | 4–6 gün | MEDIUM |
| 5 — AI | 4–6 gün | MEDIUM |
| 6 — AstroScope (ayrı depo) | 5–10 gün | HIGH |
| 7 — Üretim | 4–6 gün | HIGH |
| 8 — Dokümantasyon | 1–2 gün | LOW |
| **Toplam** | **~33–55 iş günü** | |

Kritik yol: Faz 0 → 1 → 2 → 3 → 7. Faz 4/5/6 paralelleştirilebilir (modül sınırları
netleştikten sonra). AstroScope, Core'un `contracts` paketi hazır olduğunda
(Faz 1) paralel başlayabilir; ödeme/AI entegrasyonu Faz 3/5'i bekler.

---

## 14. Hemen Yapılacaklar (bugün)

1. Faz 0'ı tek commit'te tamamla; build'i yeşile al.
2. Mock `main` dosyalarını sil.
3. `doc/GELISTIRME-PLANI.md` ve `doc/ASTROSCOPE-ENTEGRASYON.md`'yi tek referans ilan et;
   eski çelişkili dokümanları arşivle.
4. Verilmiş kararları (`D1–D5`) `docs/DECISIONS.md`'ye ADR olarak yaz.
5. Faz 1 için `core/module` + `bootstrap` + `contracts` taslağını çıkar.
