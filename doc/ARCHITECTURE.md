# Mimari

Bu belge çekirdeğin sınırlarını, veri sahipliğini ve tenant/yetki kurallarını
tanımlar. Ürün gereksinimleri için `PRD.md`, motor bağlantısı için
`doc/ENGINE_CONTRACT.md`, kararların gerekçesi için `doc/ADR.md`.

---

## 1. Katmanlar ve bağımlılık yönü

```
contract/core          ← yalnızca tipler ve arayüzler
      ▲
      │
platform/  (L0)        ← kernel; hiçbir modülü import etmez
      ▲
      │
mod/*  (L1, L2)        ← birbirini import etmez; contract/core kullanır
      ▲
      │
cmd/server             ← kompozisyon kökü
```

Kurallar `cmd/depcheck` ile zorlanır:

1. `platform/**` hiçbir `mod/**` paketini import edemez.
2. Bir modül başka bir modülü import edemez (kendi alt paketleri serbest).
3. `contract/**` ne `platform/**` ne `mod/**` import edebilir.

Bir modül ihtiyaç duyduğu servisi `core.Deps` üzerinden alır. `Deps` alanları
bootstrap tarafından doldurulur; modül, sağlayıcı modülün paketini görmez.

## 2. Modüller

| Modül | Katman | Sorumluluk | Bağımlılık |
|---|---|---|---|
| `identity` | L1 | Hesap, oturum, token, parola | – |
| `authz` | L1 | Rol, izin, kapsamlı atama | identity |
| `files` | L1 | Yükleme, sahiplik, imzalı indirme | identity |
| `notifications` | L1 | Outbox, e-posta teslimi | identity |
| `organization` | L2 | İşletme, şube, üyelik, bölge, hizmet alanı | identity, authz |
| `catalog` | L2 | Taksonomi, kategori bağı, sunum, varyant | identity, authz, organization |

`Deps` üzerinden yayılan servisler:

| Alan | Sağlayıcı arayüzü | Tüketici |
|---|---|---|
| `Identity` | `core.IdentityProvider` | files, notifications, organization, catalog, uygulama |
| `Users` | `core.UserProvider` | motorlar (salt okunur hesap projeksiyonu) |
| `Authorizer` | `core.AuthorizerProvider` | organization, catalog, motorlar |
| `Organizations` | `core.OrganizationProvider` | catalog, motorlar |
| `Catalog` | `core.CatalogProvider` | motorlar |
| `Files` / `Notifications` | `FileProvider` / `NotificationProvider` | uygulama |

## 3. İstek yaşam döngüsü

```
HTTP isteği
  → RequestID
  → kimlik middleware (Bearer önce, sonra çerez)
  → yetki middleware (Require(action, scope))
  → handler
      → servis (iş kuralları, durum makinesi)
      → repository (SQL)
  → tek hata zarfı / başarı zarfı
```

Handler katmanı ince kalır: gövdeyi bağlar, servisi çağırır, sonucu zarfa
dönüştürür. İş kuralı handler'a yazılmaz.

### 3.1 Tenant çözümleme

Tenant kimliği **hiçbir zaman** istek gövdesinden alınmaz. İki kaynak vardır:

- Rota parametresi: `/organizations/:org_id/...`
- Oturum sahibi: `/me`, `/authz/me/*`

`Require(action, scope)` middleware'i `:org_id` / `:branch_id` parametresini
okur. Parametre yoksa istek reddedilir; bu bir istemci hatası değil, rota
tanımı hatasıdır ve sessizce "hepsi"ne dönüşmesi engellenir.

Servis katmanı ayrıca yüklediği kaydın gerçekten o tenant'a ait olduğunu
doğrular (`UpdateBranch`, `DeleteVariant`, `RemoveServiceArea`…). Middleware
tek başına yeterli sayılmaz: bir kaydın başka bir tenant'a ait olması 404
döner, 403 değil — varlığı doğrulamak da bir bilgi sızıntısıdır.

## 4. Veri sahipliği

Tüm çekirdek modülleri aynı PostgreSQL şemasını (`public`) paylaşır; her modül
kendi migration kaynağını (`core.MigrationSource`) getirir. Tablo sahipliği:

| Modül | Tablolar |
|---|---|
| identity | `users`, `refresh_tokens`, `one_time_tokens` |
| authz | `roles`, `permissions`, `role_permissions`, `role_assignments` |
| files | `files` (+ depolama anahtarları) |
| notifications | `notifications`, `notification_preferences` |
| organization | `organizations`, `branches`, `memberships`, `regions`, `service_areas`, `organization_status_events` |
| catalog | `taxonomy_nodes`, `organization_categories`, `offerings`, `offering_variants` |

Kurallar:

- Bir modül başka bir modülün tablosunu **doğrudan sorgulamaz**; arayüzü çağırır.
  Tek istisna, keşif sorgularında görünürlük kısıtıdır: catalog, yayınlanmamış bir
  işletmenin sunumlarını sızdırmamak için `organizations` tablosuna **salt okunur**
  bir join yapar; yazma yetkisi yoktur.
- Yabancı anahtarlar aynı şemada olduğu için veritabanı bütünlüğü korunur.
- Bir motor kendi şemasını ve kendi `schema_migrations` geçmişini getirir; bu
  yüzden çekirdek şemasına migration yazmaz.
- Her tablo `organization_id` taşıyorsa o sütun tenant sınırıdır ve indekslidir.

### 4.1 Neden tek şema?

Ayrı şemalar daha temiz görünür ama pratikte:

- Çapraz modül sorguları (ör. sunum + organizasyon durumu) sürekli join ister.
- Migration sırası ve geri alma tek yerden yönetilemez.
- Çekirdek zaten tek binary, tek veritabanı olarak çalışıyor.

Bu yüzden modül sınırı **kodda** tutulur, veritabanı şemasında değil. Motorlar
ise kendi şemalarını getirir çünkü onlar ayrı sürümlenir ve ayrı deploy edilir.

## 5. Yetki modeli

### 5.1 Üç kavram

- **İzin** (`permissions`): `catalog.manage` gibi atomik yetenek.
- **Rol** (`roles`): izin kümesi. Hiyerarşi yoktur.
- **Atama** (`role_assignments`): `(user, role, scope, scope_id)`.

### 5.2 Kapsamlar

| Kapsam | `scope_id` | Anlam |
|---|---|---|
| `platform` | boş | Tüm platform |
| `organization` | organizasyon id | Tek işletme |
| `branch` | şube id | Tek şube |
| `self` | kullanıcı id | Kişinin kendi kaydı |

### 5.3 Çözümleme kuralı

```
etkin_izinler(user, scope, scope_id) =
    atamalar(user, scope, scope_id)
  ∪ atamalar(user, "platform", "")
```

Yani platform ataması aşağı doğru geçerlidir; organizasyon ataması yukarı
doğru geçerli değildir. Bu, platform yöneticisinin her işletmede çalışmasını
sağlarken bir işletme sahibinin platform yetkisi kazanmasını engeller.

Bunun üstünde `users.role` bir **kök yetki**dir: `admin` her kapsamda geçer,
`moderator` yalnızca inceleme ve okuma izinlerine sahiptir. Bu istisna olmadan
ilk organizasyonu yayınlayacak bir hesap yaratılamazdı. Diğer tüm kararlar
atama tablosundan gelir.

### 5.4 Varsayılan ret

Eşleşme yoksa cevap `403`'tür. "Rolü yoksa kullanıcıdır" gibi bir varsayım
yoktur. Bir uç nokta korumasız bırakılmışsa bu bir hatadır; `Require` çağrısının
eksikliği kod incelemesinde değil, testte yakalanmalıdır.

### 5.5 Atama yaşam döngüsü

- `Grant`: yeni satır; aynı `(user, role, scope, scope_id)` için aktif ikinci
  satır kısmi unique indeksle engellenir.
- `Revoke`: `revoked_at` yazılır, satır silinmez.
- `expires_at`: süreli atama; süresi geçmiş atama etkin sayılmaz.
- Sahiplik devri eski sahibin `owner` atamasını iptal eder, yenisine verir.

## 6. Durum makineleri

### 6.1 Organizasyon

```
draft ──► pending_verification ──► active ──► suspended
  │               │                   │            │
  └───────────────┴───────────────────┴────────────┴──► closed
```

- Yalnızca `organization.publish` izni geçişi tetikleyebilir (platform).
- Geçişler `UPDATE ... WHERE status = <beklenen>` ile yapılır; bayat yazıcı
  adım atlayamaz.
- Her geçiş `organization_status_events`'e yazılır.

### 6.2 Sunum

```
draft ──► published ──► archived
  ▲                        │
  └────────────────────────┘
```

- Yayınlama `catalog.publish` gerektirir.
- Yayınlanmış sunum silinemez; önce arşivlenir.
- Oluşturma her zaman `draft` ile başlar; istemci doğrudan yayınlanmış
  oluşturamaz.

## 7. Eşzamanlılık ve tutarlılık

- Durum geçişleri koşullu `UPDATE` ile yapılır; etkilenen satır 0 ise
  `409 INVALID_STATE`.
- Organizasyon oluşturma üç adımdır (organizasyon, üyelik, atama). Üyelik veya
  atama başarısız olursa istek hata döner; yarım kalan organizasyon sahipsiz
  kalır ve operatör müdahalesi gerekir. V1'de bu kabul edilen bir sınırdır
  (bkz. ADR-008).
- Atama iptali ve üyelik güncellemesi ayrı transaction'lardır; ikisi de
  idempotent olacak şekilde yazılmıştır.

## 8. Hata modeli

Tüm hatalar `contract/core` içindeki sentinel'lara sarılır ve tek yerde HTTP'ye
çevrilir (`platform/httpx`).

| Sentinel | HTTP | Kod |
|---|---|---|
| `ErrUnauthorized` | 401 | `UNAUTHORIZED` |
| `ErrForbidden` | 403 | `FORBIDDEN` |
| `ErrNotMember` | 403 | `NOT_A_MEMBER` |
| `ErrNotFound` | 404 | `NOT_FOUND` |
| `ErrValidation` | 422 | `VALIDATION_ERROR` |
| `ErrInvalidState` | 409 | `INVALID_STATE` |
| `ErrConflict` | 409 | `CONFLICT` |
| `ErrRateLimited` | 429 | `RATE_LIMITED` |
| diğer | 500 | `INTERNAL` |

5xx gövdesi asla iç mesaj taşımaz. `DomainError` alan adı ve güvenli mesaj
taşır; `errors.Is` ile sentinela çözülür.

## 9. Gözlemlenebilirlik

- `X-Request-ID` istemciden gelirse korunur, gelmezse üretilir.
- Hata zarfı aynı kimliği döner.
- Modül `Start`/`Stop` olayları yapılandırılmış log ile yazılır.
- Sağlık uçları: `/healthz`, `/readyz`, `/version`.

## 10. Test stratejisi

| Katman | Yaklaşım |
|---|---|
| Saf kurallar (durum geçişi, kodlama, kapsam çözümleme) | Tablo testleri, veritabanısız |
| Repository / migration | `TEST_DATABASE_DSN` varsa gerçek Postgres |
| HTTP | `httptest` + echo; zarf ve hata eşlemesi |
| Sınırlar | `go run ./cmd/depcheck` (CI) |

Veritabanı gerektiren testler DSN yoksa **atlanır**, sessizce geçmez.
