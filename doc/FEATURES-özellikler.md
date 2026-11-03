# Özellikler

Bu belge çekirdeğin **ne yapabildiğini** tek listede toplar: modüller, kurallar,
uç noktalar, yapılandırma ve araçlar. Ürün kapsamı ve kabul kriterleri için
`../PRD.md`, sınırlar için `ARCHITECTURE.md`, motor bağlantısı için
`ENGINE_CONTRACT.md`.

Sürüm: 0.2.0 (Hizmet Katmanı Çekirdeği V1)

---

## 1. Genel bakış

| Katman | Modül | Ne yapar |
|---|---|---|
| L0 | `platform/` | Config, PostgreSQL, migration, HTTP zarfı, health, modül yöneticisi |
| L1 | `identity` | Hesap, oturum, token, parola |
| L1 | `authz` | Rol + izin + kapsamlı atama |
| L1 | `files` | Yükleme, sahiplik, imzalı indirme |
| L1 | `notifications` | Outbox, e-posta, tercihler |
| L2 | `organization` | İşletme, şube, üyelik, bölge, hizmet alanı |
| L2 | `catalog` | Taksonomi, kategori bağı, sunum, varyant |

**Bilinçli olarak yok:** randevu, sipariş, teklif, ders, proje, ödeme, cüzdan,
abonelik, lojistik, AI. Bunlar `ENGINE_CONTRACT.md` üzerinden ayrı depolarda
yazılır.

---

## 2. Kernel (L0) özellikleri

### 2.1 Yapılandırma
- Öncelik sırası: **ortam değişkeni > `config/config.yaml` > varsayılan**.
- Fail-fast doğrulama: prodüksiyonda eksik/hatalı sır ile süreç **başlamaz**.
- Zorunlu prodüksiyon kuralları:
  - `JWT_ACCESS_SECRET` ve `JWT_REFRESH_SECRET` en az 32 karakter ve farklı.
  - `CORS_ALLOWED_ORIGINS` boş olamaz, `*` reddedilir.
  - `cookie_secure` otomatik `true`.
- Sırlar yalnızca ortamdan okunur; YAML'a yazılmaz, imaja gömülmez.
- Modül durumları `modules:` anahtarından okunur: `ENABLED`, `DISABLED`,
  `MAINTENANCE`, `EXPERIMENTAL`, `DEPRECATED`.
- Config'te olup binary'de olmayan modül → **config drift** hatası.

### 2.2 Veritabanı ve migration
- PostgreSQL; bağlantı havuzu, yavaş sorgu eşiği (`slow_query`).
- Gömülü SQL migration'ları (`go:embed`), sürüm + checksum tablosu
  (`schema_migrations`).
- Uygulanmış bir dosya değiştirilirse migration **reddedilir** (yeni dosya yaz).
- Eşzamanlı deploy'lara karşı `pg_advisory_lock`.
- Aynı şemayı paylaşan modüllerin dosyaları tek geçmişte birleştirilir; aynı
  isimli iki dosya hata verir.
- Uygulama kendi migration kaynağını `core.MigrationSource` ile getirir.
- Boş veritabanına tüm çekirdek migration'ları uygulanır ve ikinci kez
  çalıştırılabilir (`test/migrations`).

### 2.3 HTTP
- Tek başarı/başarısızlık zarfı; hata kodları sabit ve makine okunur.
- Her yanıtta `X-Request-ID` (istemciden gelirse korunur).
- Global middleware zinciri: request id → panic recovery → timeout → gövde
  limiti → güvenlik başlıkları → CORS.
- 5xx gövdesi iç hata mesajı **sızdırmaz**.
- Sayfalama `limit`/`offset`; üst sınır 100.
- Geliştirmede `/` kökünde rota listesi; prodüksiyonda kapalı.

### 2.4 Sağlık ve sürüm
- `GET /healthz` — süreç ayakta (DB'ye bakmaz).
- `GET /readyz` — veritabanı erişilebilir; değilse 503.
- `GET /version` — ad, sürüm, commit, derleme zamanı.

### 2.5 Modül yönetimi
- Topolojik başlatma (Kahn); bilinmeyen bağımlılık → **fail-closed**.
- Döngü tespiti.
- Hata olursa başlatılanların ters sırada geri alınması.
- Sağlayıcılar her modül başladıktan sonra çözülür; “typed nil” tuzağı engelli.
- Modül `Start`/`Stop` olayları yapılandırılmış log ile yazılır.
- Graceful shutdown (SIGINT/SIGTERM), yapılandırılabilir süre.

### 2.6 Sınır denetimi
- `go run ./cmd/depcheck` üç kuralı zorlar:
  1. `platform/**` hiçbir `mod/**` import etmez.
  2. Modüller birbirini import etmez (kendi alt paketleri serbest).
  3. `contract/**` ne `platform` ne `mod` import eder.
- CI'da `gofmt`, `vet`, `depcheck`, `test -race` çalışır.

---

## 3. Kimlik (`identity`)

### 3.1 Hesap
- E-posta + parola ile kayıt; parola minimum uzunluğu yapılandırılabilir.
- Kayıt yanıtı, adresin zaten kullanımda olup olmadığını **belli etmez**.
- Profil alanları: ad, dil (`locale`), avatar, telefon.
- Telefon değişince doğrulama durumu sıfırlanır.
- Hesap durumları: `active`, `suspended`, `closed`, `deleted`.
- Kendi hesabını kapatma (soft delete + tüm oturumları geçersiz kılma).
- Avatar/dosya referansı dosya modülünün sahiplik kontrolünden geçer.

### 3.2 Oturum ve token
- Erişim token'ı JWT; yenileme token'ı **opaque** ve yalnızca SHA-256 hash'i
  saklanır.
- Yenileme token'ı aile (`family_id`) bazında izlenir; tekrar kullanım tespit
  edilirse aile komple iptal edilir.
- `token_version`: parola değişimi ve “her yerden çıkış” eski erişim
  token'larını anında geçersiz kılar.
- Yeni token üretilirken eski token `replaced_by` ile zincirlenir.
- İki taşıma biçimi:
  - **Bearer**: API/mobil istemciler; token gövdede döner.
  - **Web**: HttpOnly `access`/`refresh` çerezleri + okunabilir `csrf` çerezi.
- Bearer her zaman çerezden önce gelir; ikisi aynı anda varsa çerez yok sayılır.
- CSRF double-submit yalnızca çerezle kimlik doğrulanan yazma isteklerinde
  uygulanır; Bearer ve anonim istekler muaf.

### 3.3 Parola ve doğrulama akışları
- E-posta doğrulama, parola sıfırlama: tek kullanımlık, hash'lenmiş, süreli token.
- “Parolamı unuttum” yanıtı adresin varlığını belli etmez (her zaman 202).
- Parola ve token karşılaştırmaları sabit zamanlı.

### 3.4 Kötüye kullanım koruması
- Giriş denemeleri IP **ve** e-posta bazında sınırlanır.
- Başarılı giriş kendi bütçesini iade eder; NAT arkasındaki kullanıcı kendi
  başarılı girişleriyle kilitlenmez.
- Yenileme uç noktası ayrı sınıra sahiptir.

### 3.5 Platform rolleri
- `user`, `admin`, `moderator`.
- `admin`/`moderator` yalnızca platform görevleri içindir; işletme kapsamlı
  yetki `authz` modülündedir.
- Opsiyonel kimlik middleware'i (`OptionalMiddleware`): kimlik varsa ekler,
  yoksa anonim geçirir (herkese açık keşif için).

### 3.6 Uç noktalar

| Metot | Yol | Not |
|---|---|---|
| POST | `/api/v1/auth/register` | Hız sınırlı |
| POST | `/api/v1/auth/login` | Hız sınırlı |
| POST | `/api/v1/auth/refresh` | Ayrı hız sınırı |
| POST | `/api/v1/auth/logout` | Yenileme token'ını iptal eder |
| POST | `/api/v1/auth/forgot-password` | Hız sınırlı |
| POST | `/api/v1/auth/reset-password` | Hız sınırlı |
| POST | `/api/v1/auth/verify-email` | |
| GET | `/api/v1/me` | Oturum sahibi |
| PATCH | `/api/v1/me` | Ad, dil, avatar, telefon |
| DELETE | `/api/v1/me` | Hesabı kapatır |
| POST | `/api/v1/me/logout-all` | Tüm oturumlar |
| GET | `/api/v1/admin/users` | Yalnızca `admin` |
| DELETE | `/api/v1/admin/users/:id` | Yalnızca `admin` |

---

## 4. Yetkilendirme (`authz`)

### 4.1 Model
- **İzin**: atomik yetenek, modül ad alanına göre (`catalog.manage`).
- **Rol**: izin kümesi (hiyerarşi yok).
- **Atama**: `(kullanıcı, rol, kapsam, kapsam kimliği)`.
- Kapsamlar: `platform`, `organization`, `branch`, `self`.

### 4.2 Kurallar
- **Varsayılan ret**: eşleşen aktif atama yoksa `403`.
- Platform kapsamlı atama her kapsamda geçerlidir; organizasyon ataması platform
  kapsamına yükselmez.
- `users.role = admin` kök yetkidir (ilk organizasyonu yayınlayacak hesap);
  `moderator` yalnızca inceleme + okuma.
- `expires_at`: süreli atama; süresi geçmiş atama etkin sayılmaz.
- Aynı `(user, role, scope, scope_id)` için ikinci aktif satır kısmi unique
  indeksle engellenir.
- İptal (`revoke`) satırı silmez; `revoked_at` yazar → denetim izi korunur.

### 4.3 Tenant güvenliği
- Kapsam kimliği **istek gövdesinden alınmaz**.
- `Require(action, scope)` kapsamı rota parametresinden okur:
  `:org_id`, `:branch_id`, `self` için oturum sahibi.
- Rota parametresi yoksa istek reddedilir (sessizce “hepsi”ne dönüşmez).
- Servis ayrıca yüklediği kaydın tenant'a ait olduğunu doğrular; başka tenant'ın
  kaydı `404` döner.

### 4.4 Tohum verisi
- İzinler: `platform.admin`, `moderation.review`, `organization.read`,
  `organization.manage`, `organization.publish`, `org.branch.manage`,
  `org.member.manage`, `catalog.read`, `catalog.manage`, `catalog.publish`,
  `taxonomy.manage`, `review.respond`.
- Sistem rolleri: `platform_admin`, `moderator`, `owner`, `manager`, `staff`,
  `member`.

### 4.5 Uç noktalar

| Metot | Yol | Not |
|---|---|---|
| GET | `/api/v1/authz/me/assignments` | Etkin atamalar |
| GET | `/api/v1/authz/me/permissions?scope=&scope_id=` | Etkin izinler (UI ipucu) |
| GET | `/api/v1/authz/admin/roles` | Rol listesi |
| POST | `/api/v1/authz/admin/roles` | Rol tanımla/güncelle |
| GET | `/api/v1/authz/admin/users/:user_id/assignments` | Kullanıcı atamaları |
| POST | `/api/v1/authz/admin/grants` | Atama ver |
| DELETE | `/api/v1/authz/admin/grants/:id` | Atamayı iptal et |

---

## 5. Dosya (`files`)

- Yükleme, listeleme, meta okuma, indirme, silme.
- **Sahiplik zorunlu**: başka kullanıcının dosyası `404` (403 değil).
- İçerik doğrulama:
  - Kapalı izin listesi: `png`, `jpeg`, `webp`, `gif`, `pdf`.
  - **SVG reddedilir** (saklı XSS riski).
  - Bildirilen içerik türü ile gerçek baytlar (magic number) karşılaştırılır.
- Boyut sınırı (varsayılan 10 MB).
- Depolama sürücüsü: `local` (varsayılan). `s3` tanımlı ama **uygulanmadı**;
  bilinmeyen sürücü sessizce diske düşmez, başlatma hata verir.
- İndirme, sahiplik kontrolünden geçen `GET /files/:id/download` uç noktasından
  yapılır. Depolama arayüzünde imzalı URL yöntemi vardır; yerel sürücüde bu yol
  ayrı bir HTTP rotası olarak sunulmaz. Yetki kontrolü her zaman handler'da
  kalır.
- Meta veri ile baytlar tutarsızsa yükleme geri alınır.

### Uç noktalar

| Metot | Yol |
|---|---|
| POST | `/api/v1/files` |
| GET | `/api/v1/files` |
| GET | `/api/v1/files/:id` |
| GET | `/api/v1/files/:id/download` |
| DELETE | `/api/v1/files/:id` |

---

## 6. Bildirim (`notifications`)

- **Outbox deseni**: mesaj önce veritabanına yazılır, sonra gönderilir.
- Yeniden deneme: üstel geri çekilme, deneme sayısı ve son hata kaydı.
- Kanallar: `email`, `in_app` (saklanır, istemci okur), `push` (rezerve).
- E-posta yalnızca bir `Mailer` bağlandığında teslim edilir; bağlı değilse mesaj
  kuyrukta durur, kaybolmaz (geliştirme için doğru varsayılan).
- Kullanıcı tercihleri kanal bazında; satır yoksa varsayılan: e-posta + in-app
  açık, push kapalı.
- Şablonlar gömülü (`templates/*.tmpl`); gönderilen konu/gövde saklanır, böylece
  şablon sonradan değişse bile yeniden deneme aynı metni yollar.
- Okundu işaretleme (tek/ tümü), okunmamış sayısı.
- Arka plan teslim döngüsü; `Stop` ile güvenli kapanır.

### Uç noktalar

| Metot | Yol |
|---|---|
| GET | `/api/v1/notifications` |
| GET | `/api/v1/notifications/unread-count` |
| POST | `/api/v1/notifications/:id/read` |
| POST | `/api/v1/notifications/read-all` |
| GET | `/api/v1/notifications/preferences` |
| PUT | `/api/v1/notifications/preferences` |

---

## 7. İşletme (`organization`)

### 7.1 Organizasyon
- Oluşturan kişi otomatik `owner` olur (üyelik + atama birlikte).
- Slug: küçük harf, rakam, tire; tekil.
- Profil: ad, açıklama, logo, e-posta, telefon, web, dil.
- Durum makinesi:
  `draft → pending_verification → active → suspended → closed`
  - `active` yalnızca platform yetkisiyle (`organization.publish`).
  - `closed` terminal.
  - Geçişler koşullu `UPDATE` ile; bayat yazıcı adım atlayamaz.
  - Her geçiş `organization_status_events`'e yazılır (kim, ne zaman, neden).
- Sahiplik devri açık bir işlem; yeni sahip **aktif üye** olmalı, eski sahip
  `manager` olur.
- Sahipsiz bırakma engelli: son sahip çıkarılamaz, devir zorunlu.
- Herkese açık keşifte yalnızca `active` işletmeler görünür; `draft` slug'ı
  `404` döner.

### 7.2 Şube
- Türler: `physical`, `service_area`, `online`.
- Adres, il/ilçe, posta kodu, koordinat, telefon.
- `public=false` ise tam adres keşifte gösterilmez.
- Durumlar: `active`, `inactive`, `suspended`.

### 7.3 Üyelik ve davet
- Durumlar: `invited`, `active`, `suspended`, `removed`.
- Davet e-posta adresine yapılır; **yalnızca o adrese sahip hesap** kabul eder.
  Deneme, id tahmin edilse bile bilgi sızdırmaz.
- Üyelik rolleri: `owner`, `manager`, `staff`, `member`.
- Rol değişimi ilgili atamayı da günceller (iptal + yeni atama).
- Aynı organizasyonda ikinci aktif üyelik engelli.
- Kayıt sonrası bekleyen davetleri hesaba bağlayan bir servis yöntemi vardır
  (`AcceptPendingInvitations`); zaten üye olunan organizasyon atlanır. V1'de bu
  çağrıyı kayıt akışına bağlamak uygulamanın kararıdır (modül sınırı).

### 7.4 Bölge ve hizmet alanı
- Bölge ağacı: `country` → `city` → `district` (kod + ad + dil).
- Hizmet alanı: organizasyon + opsiyonel şube + opsiyonel bölge + mod
  (`on_site`, `online`, `at_customer`, `hybrid`).
- Şube başka organizasyona aitse hizmet alanı reddedilir.

### 7.5 Uç noktalar

| Metot | Yol | Yetki |
|---|---|---|
| GET | `/api/v1/organizations` | Herkese açık (yalnızca `active`) |
| GET | `/api/v1/organizations/by-slug/:slug` | Herkese açık |
| GET | `/api/v1/regions?parent_id=` | Herkese açık |
| POST | `/api/v1/organizations` | Oturum (sahip = oturum) |
| POST | `/api/v1/invitations/:id/accept` | Oturum + e-posta eşleşmesi |
| GET | `/api/v1/organizations/:org_id` | `organization.read` |
| PATCH | `/api/v1/organizations/:org_id` | `organization.manage` |
| POST | `/api/v1/organizations/:org_id/transfer-ownership` | `organization.manage` |
| GET | `.../branches` | `organization.read` |
| POST | `.../branches` | `org.branch.manage` |
| PUT | `.../branches/:branch_id` | `org.branch.manage` |
| DELETE | `.../branches/:branch_id` | `org.branch.manage` |
| GET | `.../members` | `organization.read` |
| POST | `.../members` | `org.member.manage` |
| PATCH | `.../members/:membership_id` | `org.member.manage` |
| DELETE | `.../members/:membership_id` | `org.member.manage` |
| GET | `.../service-areas` | `organization.read` |
| POST | `.../service-areas` | `organization.manage` |
| DELETE | `.../service-areas/:area_id` | `organization.manage` |
| POST | `/api/v1/admin/organizations/:org_id/status` | `organization.publish` (platform) |

---

## 8. Katalog (`catalog`)

### 8.1 Taksonomi
- Merkezi kategori ağacı; kategori **serbest metinle temsil edilemez**.
- Alanlar: slug, ad, açıklama, üst düğüm, sıra, yayın durumu, dil.
- `published=false` veya `inactive` düğüm herkese açık uçta `404`.
- Yol (root → düğüm) sorgulanabilir; döngüye karşı derinlik sınırı.
- `extension_schema_ref`: bir motorun kendi alan şemasına referans (çekirdek
  yorumlamaz).

### 8.2 Organizasyon kategorileri
- Bir organizasyon birden çok kategoriye bağlanabilir; çift kayıt engelli.
- Kategori listesi `PUT` ile **tümüyle değiştirilir** (idempotent).

### 8.3 Sunum (offering)
- Alanlar: ad, slug (organizasyon içinde tekil), özet, açıklama, medya, hizmet
  modu, görünürlük, dil, etiketler, bilgilendirici `price_from`/para birimi.
- **Yok**: fiyat hesaplama, vergi, indirim, stok, SKU, süre, kapasite.
- Durum makinesi: `draft → published → archived`; `archived → draft`.
  - Oluşturma her zaman `draft`; istemci doğrudan yayınlanmış yaratamaz.
  - Yayınlama `catalog.publish` gerektirir.
  - Yayınlanmış sunum silinemez; önce arşivlenir.
- Görünürlük: `public`, `unlisted`, `private`; `private` herkese açık uçta 404.
- Kategori filtresi **alt kategorileri kapsar** (özyinelemeli sorgu).
- Herkese açık listede yalnızca `published` + `public` + işletmesi `active`
  olan sunumlar görünür.

### 8.4 Varyant
- Sunuma bağlı adlandırılmış sunum varyantı; medya ve görünürlük taşır.
- Süre/kapasite/SKU gibi anlamlar motora aittir.

### 8.5 Motor sınırı
- `extension_ref`: `extension_id`, `resource_type`, `resource_id`, `label`.
- Çekirdek bu referansı **saklar, okumaz, doğrulamaz**.

### 8.6 Uç noktalar

| Metot | Yol | Yetki |
|---|---|---|
| GET | `/api/v1/taxonomy?parent_id=` | Herkese açık |
| GET | `/api/v1/taxonomy/:slug` | Herkese açık |
| GET | `/api/v1/offerings?taxonomy_id=&region_id=&q=&organization_id=` | Herkese açık |
| GET | `/api/v1/offerings/:id` | Herkese açık (yayınlanmışsa) |
| GET | `/api/v1/organizations/:org_id/offerings` | `catalog.read` |
| GET | `/api/v1/organizations/:org_id/categories` | `catalog.read` |
| PUT | `/api/v1/organizations/:org_id/categories` | `catalog.manage` |
| POST | `/api/v1/organizations/:org_id/offerings` | `catalog.manage` |
| PUT | `/api/v1/organizations/:org_id/offerings/:offering_id` | `catalog.manage` |
| DELETE | `/api/v1/organizations/:org_id/offerings/:offering_id` | `catalog.manage` |
| POST | `.../offerings/:offering_id/variants` | `catalog.manage` |
| PUT | `.../offerings/:offering_id/variants/:variant_id` | `catalog.manage` |
| DELETE | `.../offerings/:offering_id/variants/:variant_id` | `catalog.manage` |
| POST | `.../offerings/:offering_id/status` | `catalog.publish` |
| POST | `/api/v1/admin/taxonomy` | `taxonomy.manage` (platform) |
| PATCH | `/api/v1/admin/taxonomy/:id` | `taxonomy.manage` (platform) |

---

## 9. Sözleşme yüzeyi (`contract/core`)

Bir uygulama veya motor yalnızca bu paketi import eder.

| Grup | İçerik |
|---|---|
| Modül | `Module`, `Deps`, `App`, `MigrationSource`, `ConfigView`, `Logger`, `Clock` |
| Kimlik | `Identity`, `Principal`, `UserStore`, `User`, rol ve hesap sabitleri |
| Yetki | `Authorizer`, `Check`, `Role`, `Permission`, `Assignment`, `GrantInput`, `ScopeType` |
| İşletme | `Organizations`, `Organization`, `Branch`, `Membership`, `Region`, `ServiceArea` |
| Katalog | `Catalog`, `TaxonomyNode`, `Offering`, `OfferingVariant`, `ExtensionRef` |
| Dosya / bildirim | `Files`, `Storage`, `Notifications`, `Mailer` |
| Hatalar | `ErrUnauthorized`, `ErrForbidden`, `ErrNotMember`, `ErrNotFound`, `ErrValidation`, `ErrInvalidState`, `ErrConflict`, `ErrRateLimited`, `DomainError` |
| Sağlayıcılar | `IdentityProvider`, `UserProvider`, `FileProvider`, `StorageProvider`, `NotificationProvider`, `AuthorizerProvider`, `OrganizationProvider`, `CatalogProvider` |

Sürümleme: alan **eklemek** minor, alan **kaldırmak/değiştirmek** major.

---

## 10. Araçlar

### 10.1 `corectl`
| Komut | Ne yapar |
|---|---|
| `corectl new <app> --level=L0\|L1\|L2 [--only=...] [--dir=...]` | Yeni uygulama iskeleti |
| `corectl add module <name>` | Modül ekle |
| `corectl add level <LN>` | Seviye yükselt |
| `corectl doctor` | Config + migration + modül durumu kontrolü |
| `corectl upgrade` | Çekirdek sürüm farkını uygular, major'da durur |

Üretilen proje: `cmd/server/main.go`, `config/`, `Dockerfile`,
`docker-compose.yml`, `Makefile`, `.env.example`, örnek modül ve test.
Bağımlılıklar çözülür, `vendor/` üretilir.

### 10.2 `depcheck`
Modül sınırlarını denetler (bkz. §2.6). CI'da zorunlu.

### 10.3 Betikler
| Betik | Ne yapar |
|---|---|
| `scripts/run.ps1` | Boş port bulur, konteynerleri başlatır, hazır olana kadar bekler |
| `scripts/stop.ps1` | Durdurur |
| `scripts/smoke.ps1 -BaseUrl ...` | Uçtan uca duman testi |

Duman testi kapsamı: sağlık, hata zarfı, kayıt/giriş/yenileme/çıkış, parola
akışları, hız sınırı, dosya sahipliği (IDOR), bildirim tercihleri, organizasyon
oluşturma, sahiplik izni, izolasyon (403/404), şube, davet ve katalog erişimi.

### 10.4 Geliştirme komutları
```bash
make check      # gofmt + vet + depcheck + test
make test
make run
make depcheck
make corectl ARGS="doctor"
```

Test stratejisi:
- Saf kurallar → veritabanısız tablo testleri.
- Veritabanı gerektiren testler `TEST_DATABASE_DSN` yoksa **atlanır** (sessizce
  geçmez).
- `test/migrations` tüm çekirdek migration'larını boş veritabanına uygular ve
  idempotansı doğrular.

---

## 11. Yapılandırma özeti

| Anahtar | Varsayılan | Not |
|---|---|---|
| `env` | `development` | `production` katı kuralları açar |
| `server.host` / `port` | `0.0.0.0` / `8080` | |
| `db.*` | localhost/postgres | `max_open_conns`, `slow_query` |
| `http.body_limit` | `8M` | |
| `http.request_timeout` | `30s` | |
| `http.shutdown_timeout` | `10s` | |
| `log.level` / `format` | `info` / `json` | |
| `security.allowed_origins` | boş | prodüksiyonda zorunlu, `*` yasak |
| `identity.access_ttl` | `15m` | |
| `identity.refresh_ttl` | `168h` | |
| `identity.password_min_length` | `10` | |
| `identity.cookie_secure` | `false` (prod: `true`) | |
| `modules.<name>` | `ENABLED` | `DISABLED` / `MAINTENANCE` / … |

Ortam değişkenleri: `APP_ENV`, `SERVER_*`, `DB_*`, `HTTP_*`, `LOG_*`,
`CORS_ALLOWED_ORIGINS`, `JWT_ACCESS_SECRET`, `JWT_REFRESH_SECRET`,
`JWT_ACCESS_TTL`, `JWT_REFRESH_TTL`, `TEST_DATABASE_DSN` (yalnızca test).

---

## 12. Kapsam dışı (bilinçli)

- Randevu/rezervasyon/takvim/kapasite
- Sipariş/sepet/stok/SKU/kargo
- Teklif/sözleşme/onay akışı
- Ders/sınav/müfredat
- Proje/görev
- Ödeme/cüzdan/abonelik/fatura
- AI çağrısı/prompt yönetimi
- Arama motoru (yardımcı paket olabilir, çekirdek modülü değildir)
- Raporlama/BI, çeviri yönetimi
- Web/mobil arayüz (ayrı depo)

Bir motor bunları `ENGINE_CONTRACT.md` üzerinden kendi deposunda uygular.
