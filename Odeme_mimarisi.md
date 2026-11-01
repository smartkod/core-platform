# Ödeme Mimarisi — Rezervasyon ve Organizasyon Pazaryeri Platformu

**Teknik Şartname v1.0 — 2026-11-02**

> **Doğrulama notu:** PayTR'e dayanan iddialar `[P]` etiketiyle kaynak
> bağlantısına işaret eder; kaynaklar PayTR geliştirici dokümanından
> (dev.paytr.com) incelenerek doğrulanmıştır. Doğrulanamayan ya da PayTR
> ile yazılı teyit gerektiren konular ⚠️ işaretlidir; doğrulanamayan
> özellikler gerçekleşmiş gibi yazılmamıştır. Varsayımlar [VARSAYIM]
> etiketlidir, karar beklenenler [KARAR] etiketlidir.
>
> **Ana mimari karar:** Önce **ücretsiz üyelik + ücretsiz mod**; jeton ve
> paket satışı başlangıç modunda da çalışır (ödeme manuel onaylı);
> **PayTR Pazaryeri entegrasyonu ikinci moddur**; ödeme sağlayıcısı
> adaptör olarak eklenir. Rezervasyon çekirdeği bağımsız kalır.

---

## 1. Kapsam, Hedefler ve Kapsam Dışı

### 1.1 Platform tanımı

Platform; farklı sektörlerdeki işletmelerin/satıcıların hizmet ve
organizasyonlarını listelediği, müşterilerin işletmeyle iletişim kurduğu,
**teklif (offer)** ve **rezervasyon (booking)** talebi oluşturduğu bir
pazaryeridir; ileride birden fazla hizmet sağlayıcısı kategorisi tek
platformda toplanır.

### 1.2 Hedefler

| # | Hedef | Öncelik |
|---|---|---|
| H1 | Ücretsiz modda müşteri–satıcı iletişimi ve teklif/rezervasyon talebi | P0 |
| H2 | Her rezervasyonda ödeme modunun oluşum anında sabitlenmesi ve sonradan değiştirilememesi | P0 |
| H3 | PayTR modunun çekirdeği değiştirmeden adaptör olarak eklenebilmesi | P0 |
| H4 | Komisyon kuralı yönetimi ve işlem anında oran sabitleme | P1 |
| H5 | Satıcı hakediş kayıtları (payment_allocations) ve transfer denetimi | P1 |
| H6 | Kısmi/tam iade ve iptal senaryolarının doğru tutar modeliyle işlenmesi | P1 |
| H7 | Yönetim paneli ekranları | P1 |

### 1.3 Kapsam Dışı (ilk sürümde geliştirilmez)

- Platform cüzdanı (müşterinin platformda bakiye tutması).
- Kart saklama (platform tarafında token vault) — gerekirse PayTR'in
  kart saklaması sağlayıcı özelliği olarak kullanılır [VARSAYIM].
- Manuel komisyon tahsilatı (satıcıdan ayrıca fatura/fatura kesmek).
- Kendi ödeme geçidimizi işletmek: PayTR'nin gerçekleştirdiği tahsilat
  ve transfer işlevleri **yeniden yazılmaz**; platform yalnızca kendi
  rezervasyon, ödeme durum eşleştirme, komisyon ve raporlama kayıtlarını
  yönetir.
- Ayrı banka transfer motoru (hakedişi kendi POS'umuzdan göndermek);
  PayTR modunda hakediş PayTR **Platform Transfer Talebi** ile yürür [P1].
- Zorunlu satıcı aboneliği (sonraki faz, P2).
- e-Fatura/ön muhasebe motoru; satıcı faturasını kendisi düzenler.
- **Ücretsiz modda müşteri ödemesi:** platform müşteri parası almaz,
  komisyon kesmez; ücretsiz rezervasyon için sahte (stub) `payments`
  kaydı oluşturulmaz (bkz. §6).

## 2. Roller ve Yetkiler

| Rol | Tanım | Yetki / Öncelik |
|---|---|---|
| Misafir | Girişsiz kullanıcı | Katalog gezer; teklif/rezervasyon talebi için kayıt gerekir [VARSAYIM]. P0 |
| Müşteri | Ücretsiz üye | Teklif/rezervasyon talebi oluşturur; kendi taleplerini görür; PayTR modunda ödeme yapar. P0 |
| İşletme/Satıcı | Hizmet sağlayan hesap | Hizmet tanımlar, takvim/uygunluk yönetir, talepleri kabul/reddeder, hakediş özetini görür. P0 |
| Platform admin | Operasyon ekibi | Satıcı onayı, komisyon kuralları, ödeme/iade müdahalesi, mod geçiş ayarı. P0 |
| Finans admin | Ayrı yetkilendirilmiş rol | Hakediş planı onayı, transfer gönderimi, iade onayı. P1 |

Yetkilendirme ilkesi: satıcı kaynaklarına erişilemezken **404** (varlık
gizleme), admin endpoint'lerine yetkisiz erişimde **403**.

## 3. Ana Kullanıcı Akışları

### 3.1 Ücretsiz mod (Faz 1)

1. Müşteri satıcı/hizmet sayfasını görüntüler.
2. Müşteri iletişim formu ile satıcıyla **platform içinde** iletişim kurar.
3. Müşteri teklif talebi (`offer_request`) veya rezervasyon talebi
   (`booking_request`) gönderir; satıcı talebi kabul veya reddeder.
4. Kabulde talep `booking`'e dönüşür; `payment_mode = 'free'` yazılır.
5. Hizmet bedeli **platform dışında**, müşteri ile satıcı arasında
   gerçekleşir; platform bu modda müşteri ödemesi almaz ve komisyon
   kesmez; platform yalnızca kayıt/rapor tutar (`payment_allocations`
   satırı nötr yazılır, bkz. §6).

### 3.2 PayTR modu (Faz 3)

1. Satıcı uygunluk/takvim ve iptal politikası tanımlar (Faz 2 hazırlığında).
2. Müşteri rezervasyon talebi oluşturur; satıcı kabul eder; booking
   oluşur ve `payment_mode = 'paytr_platform'` **sabitlenir**.
3. Müşteri ödeme başlatır (PayTR iframe / Direkt API); gerçek durum
   yalnızca **Bildirim URL** üzerinden gelir ve doğrulanır (bkz. §9).
4. `payments.status = paid` olduğunda booking `confirmed` olur; işlem
   anındaki oranla `payment_allocations` yazılır.
5. Hakediş planına göre Platform Transfer Talebi gönderilir; transfer
   callback'i kaydı günceller.
6. İptal/iade İade API ile yürütülür (kısmi iade desteklidir).

### 3.3 Jeton/paket satışı (ekonomi bağlantısı)

Ücretsiz üyelikte hoş geldin jetonu, reklam izleme/tıklamayla jeton
kazanma (günlük 3, ayda 10, admin-değişken limitler) ve jeton/abonelik
paketleri `prd.md` + `references/L2.md` (rev 1.1) kapsamındadır. Bu
şartnamede jeton ekonomisi için tek bağlantı şudur: ilk fazda paket
ödemeleri **manuel onaylı** yürür; PayTR fazında jeton/abonelik
paketleri de checkout akışına bağlanabilir (aynı `payments` modeli,
`order` referansı ile).

## 4. Ödeme Modu Ayrımı ve Modun Sabitlenmesi

- `payment_mode` değerleri: `free` | `paytr_platform`
  (ileride `vendor_paytr` gibi modlar eklenebilir).
- **Sabitlenme:** mod, `booking` satırı oluşturulduğu anda yazılır ve
  sonrasında hiçbir API/durum geçişi ile değiştirilemez; DB seviyesinde
  kolon güncellemesi trigger ve/veya updatable-view kısıtıyla engellenebilir
  (uygulama katmanındaki değişmezlik yeterli görülürse en azından testle
  kanıtlanır).
- **Mod belirleme kuralı [VARSAYIM]:** satıcı onayı ve platform fazı
  durumuna göre; admin panelde satıcı/servis başına tercih ayarlanabilir.
- **Ücretsiz mod kuralları:**
  - `bookings.payment_mode = 'free'`; tahsilat satıcı ile müşteri
    arasındadır.
  - `payments` kaydı **oluşturulmaz** (bkz. §6); sahte/başarılı
    görünümlü ödeme kaydı yasaktır.
  - Komisyon hesaplanmaz ve devredilmez.
- **PayTR mod kuralları:**
  - `payments`, `payment_events`, `payment_allocations`,
    `settlement_transfers` kayıtları yürütülür.
  - `confirmed` durumuna geçiş ancak `payments.status = paid` ile olur;
    müşteri dönüş sayfası (`merchant_ok_url`) asla durumu ilerletemez [P0].

## 5. Modül Sınırları ve Sorumlulukları

| Modül | Sorumluluk | Sorumluluğu DEĞİL |
|---|---|---|
| booking (rezervasyon çekirdeği) | Talep/kabul durumu, uygunluk/takvim, `payment_mode` sabitleme | Ödeme işlemleri |
| catalog | Satıcı/hizmet/ödeme profili kayıtları | Ödeme |
| payments | Checkout başlatma, PayTR durum eşleştirme, `payments`/`payment_events` | Rezervasyon kararları |
| commission | Kural çözümü ve işlem anında oran sabitleme | Tahsilat |
| settlement | `payment_allocations`, hakediş planı, Platform Transfer Talebi gönderimi | Banka parasını kendisi göndermek |
| refunds | İade talep/onay/durum makinesi | PayTR dışında para iadesi |
| admin-panel | Ekranlar, onaylar, raporlar | Otomatik iş kararı |
| integration/paytr (adaptör) | İstek imzası, callback doğrulaması, idempotency, hata ve yeniden deneme kuralları | İş kuralları |

**İlke:** rezervasyon çekirdeği ödeme sağlayıcısından bağımsızdır;
`booking` yalnızca `payment_mode`, tutar ve durum bilgisini tutar.
Adaptör değişimi `booking` ve `commission` tablolarını etkilemez (K8).

## 6. Önerilen İlişkisel Veri Modeli

> Konvansiyonlar: tüm **para alanları minor unit `bigint`** (kuruş);
> her yerde `float` kullanım yasak [VARSAYIM: ana para birimi TL];
> id'ler uuid [VARSAYIM]; `times` = (`created_at`, `updated_at`).

```sql
-- Satıcı (işletme)
sellers(id, user_id FK, legal_name, display_name, tax_no, tax_office,
        category, status CHECK active|pending|suspended|rejected,
        documents_verified_at, times)

-- Satıcının ödeme profili (PayTR fazında doldurulur; ücretsiz modda boş)
seller_payment_profiles(id, seller_id FK UNIQUE, paytr_submerchant_ref,
        iban, account_holder, agreement_signed_at,
        status CHECK none|pending|approved|rejected, updated_at)
        -- paytr_submerchant_ref kimlik modeli, geliştirmeden önce
        -- PayTR ile yazılı teyide bağlıdır (bkz. §9, §15.1)

-- Hizmet
services(id, seller_id FK, title, description, currency DEFAULT 'TRY',
         price_minor NULL,          -- sabit fiyat yoksa NULL (teklife açık)
         duration_minutes,
         location_type CHECK on_site|customer|online,
         active bool)

-- Teklif talebi (pazarlık akışı)
offer_requests(id, customer_id FK, service_id FK, message,
               status CHECK open|answered|accepted|rejected|expired,
               times)

-- Rezervasyon talebi (kabulden önceki talep)
booking_requests(id, customer_id FK, service_id FK, requested_slot,
                 note, status CHECK pending|accepted|rejected|expired,
                 times)

-- Rezervasyon (kabul anında oluşturulur; ödeme modu burada sabitlenir)
bookings(id, customer_id FK, service_id FK,
         booking_request_id FK NULL, offer_id FK NULL,
         slot_start, slot_end,                 -- ileri tarihli organizasyon
         payment_mode CHECK free|paytr_platform,  -- oluşumda sabit, değişmez
         total_amount_minor, currency,
         status CHECK requested|confirmed|completed|canceled|no_show,
         policy_snapshot jsonb,                -- iptal/iade politikası kopyası
         times)

-- Ödeme (yalnızca payment_mode = 'paytr_platform' iken satır oluşur)
payments(id, booking_id FK, provider, provider_ref,          -- merchant_oid
         amount_minor, currency,
         status CHECK pending|paid|failed|expired,
         paid_at, failure_code, raw_payload jsonb, times)
-- KURAL: ücretsiz rezervasyon için payment kaydı OLUŞTURULMAZ.
-- KURAL: payments.amount_minor == bookings.total_amount_minor
--        (tutar uyuşmazlığı serviste red sebebi; bkz. K5)

-- Komisyon kuralı
commission_rules(id, category NULL, seller_id NULL, service_id NULL,
                 basis CHECK percent|fixed_minor,
                 percent_bp int NULL,             -- baz puan; %8 = 800
                 fixed_minor NULL,
                 priority int,                    -- küçük sayı kazanır
                 active bool, times)

-- İşlem anında sabitlenen paylar (kural sonradan değişse kayıt değişmez)
payment_allocations(id, booking_id FK, payment_id FK NULL,
                    seller_amount_minor, platform_amount_minor,
                    commission_rate_bp, commission_basis_ref,
                    computed_at, settled bool, settled_at)
-- Ücretsiz modda: satır nötr yazılır — seller_amount NULL,
-- platform_amount = 0, commission_rate_bp = 0, payment_id NULL;
-- sahte/başarılı görünümlü değer yazılmaz.

-- İade
refunds(id, payment_id FK, amount_minor CHECK(>0),
        provider_ref, status CHECK pending|done|failed,
        reason, created_by, times)

-- İşlem/webhook kayıtları
payment_events(id, provider, event_id, booking_id, type,
               payload jsonb, processed_at,
               UNIQUE(provider, event_id))

-- Platform Transfer Talebi karşılığı [P1]
settlement_transfers(id, booking_id, provider_trans_id UNIQUE,
                     submerchant_amount_minor, total_amount_minor,
                     transfer_name, transfer_iban,
                     status CHECK planned|requested|completed|failed,
                     requested_at, result jsonb)

-- Admin denetim izi
admin_audit(id, actor_id, action, entity, entity_id,
            before jsonb, after jsonb, reason, created_at)
```

**Tutar bütünlüğü invariantları:** her PayTR ödemesi için
`seller_amount + platform_amount = paid amount`; toplam iade tutarı
ödeme tutarını aşamaz; bakiye farkı hiçbir tabloda negatif yazılamaz.

## 7. API Uç Noktaları

> Yetki kısaltmaları: `auth` = oturum, diğerleri = rol. Örnek istek/yanıt
> gövdeleri yalnızca kontrat biçimini tarif eder; teknoloji
> seçimine bağlı değildir.

### 7.1 Müşteri uçları

```
GET  /v1/services/:id                     (public) → hizmet detayı
POST /v1/offer-requests            auth   {service_id, message}
    → 201 {id, status: "open"}
POST /v1/booking-requests          auth   {service_id, requested_slot, note}
    → 201 {id, status: "pending"}
GET  /v1/my/bookings               auth   → 200 [...]
GET  /v1/bookings/:id              auth(customer|seller|admin)
POST /v1/bookings/:id/pay          customer   -- Faz 3; yalnız
                                              -- payment_mode=paytr_platform
    → 200 {checkout_uri|token, payment_id}
```

### 7.2 Satıcı uçları

```
POST /v1/offer-requests/:id/respond    seller  {answer, price_minor?}
    → 200 {status: "answered"}
POST /v1/booking-requests/:id/accept   seller
    → 201 booking {id, payment_mode, total_amount_minor}
POST /v1/booking-requests/:id/reject   seller
PUT  /v1/services/:id                  seller
GET  /v1/my/agendas                    seller   -- uygunluk/takvim
GET  /v1/my/settlements                seller   -- hakediş özeti (P1)
```

### 7.3 Platform (admin) uçları

```
POST /v1/admin/sellers/:id/approve|reject
PUT  /v1/admin/commission-rules[/:id]
GET  /v1/admin/payments?status=pending
POST /v1/admin/payments/:id/confirm|reject     -- manuel/hakediş onayı
GET  /v1/admin/settlements?plan=weekly
POST /v1/admin/settlements/:id/transfer         -- Platform Transfer Talebi
GET  /v1/admin/refunds
POST /v1/admin/refunds/:id/execute              -- İade API çağrısı
```

### 7.4 Sağlayıcı uçları (Faz 3)

```
POST /v1/payments/webhook/paytr    -- auth MUAF; hash doğrulanır;
                                   -- ayrı rate-limit; yanıt düz metin "OK"
```

## 8. Durum Makineleri

### 8.1 Rezervasyon

```
booking_request: pending → accepted | rejected | expired
accepted talep → booking oluşur (payment_mode sabitlenir)
booking: requested → confirmed → completed
                   ↘ canceled ↘ no_show
```

PayTR modunda `confirmed`, yalnız `payments.status = paid` ile olur;
`merchant_ok_url` dönüş sayfası (müşteri tarayıcısı) asla ilerletmez [P0].

### 8.2 Ödeme

```
payments: pending → paid | failed | expired
paid anında: payment_allocations (işlem anındaki oran sabitlenir)
             settlement_transfers (planned) — hakediş planına göre
```

### 8.3 İade

```
refunds: pending → done | failed
```

Kısmi iade mümkündür; toplam iade tutarı ödeme tutarını aşamaz [P3].
Hakediş transferi gönderildikten sonra iade davranışı PayTR ile
teyide bağlıdır (bkz. §15.5).

### 8.4 Satıcı ödeme profili

```
seller_payment_profiles: none → pending → approved | rejected
transfer yalnız approved profil ile planlanır (K9)
```

## 9. PayTR Adaptörü (Faz 3)

**Doğrulanmış kaynaklar:**

- [P0] Direkt API 2. Adım — Bildirim URL:
  <https://dev.paytr.com/direkt-api/direkt-api-2-adim>
- [P1] Platform Transfer Talebi:
  <https://dev.paytr.com/platform-transfer-talebi/transfer-talimatinin-verilmesi>
- [P3] İade API: <https://dev.paytr.com/iade-api>
- [P4] Durum Sorgu API: <https://dev.paytr.com/durum-sorgu>

### 9.1 Tahsilat (Bildirim URL) akışı [P0]

- Checkout PayTR iframe veya Direkt API 1. adım ile başlatılır.
- Gerçek sonuç **her zaman** panelde tanımlı **Bildirim URL**'ye POST
  ile gelir: `merchant_oid`, `status` (`success|failed`),
  `total_amount`, `hash`, `failed_reason_code/msg`, `test_mode`,
  `payment_type` (`card|eft`), `currency` (`TL|USD|EUR|GBP|RUB`),
  `payment_amount`, `installment_count`.
- **Hash doğrulama zorunlu:**
  `hash = base64(HMAC-SHA256(merchant_oid + merchant_salt + status + total_amount, merchant_key))`
  uyuşmazsa bildirim reddedilir ve hiçbir kayıt yazılmaz (K4).
- Yanıt: düz metin **`OK`**; OK almayan bildirimler tekrarlanır.
- **Idempotency:** aynı ödeme için birden fazla bildirim gelebilir;
  `merchant_oid` bazında yalnızca ilk işlenir, sonrakiler `OK` ile
  sonuçlanır ve hiçbir alan tekrar güncellenmez (K3).
- Bildirim URL kuralları: bu uçta üye girişi/erişim kısıtlaması ve
  oturum (session) **kullanılmaz**; işlemler `merchant_oid` ile
  yapılır. **`?success=1` dönüş sayfası asla ödeme/booking durumunu
  ilerletemez.**
- Başarısızlık kodları sistemde saklanır (`failure_code`,
  `failed_reason_msg`).

### 9.2 Platform Transfer Talebi (hakediş) [P1]

- Endpoint: `https://www.paytr.com/odeme/platform/transfer`; alanlar:
  `merchant_id`, `merchant_oid`, `trans_id` (≤60 alfanumerik),
  `submerchant_amount`, `total_amount` (ikisi de tutarın 100 ile
  çarpılmış hali), `transfer_name`, `transfer_iban` (26 karakter),
  `paytr_token`; token = base64 HMAC-SHA256(alanların birleşimi +
  `merchant_salt`, `merchant_key`).
- Başarılı yanıt: `{"status":"success", ..."reference":"…"}`; hata
  örneği: `"toplam transfer tutarı kalan tutardan fazla olamaz"` (010).
- **Zaman kuralı (dokümante):** talep, transferin yapılacağı gün en geç
  10:00'a kadar gönderilmeli; geç gelen istekler ertesi gün işlenir;
  **ödemeyle aynı gün transfer talebi oluşturulamaz** (en erken ertesi
  gün). Planlama mantığı bu kısıtla yazılır.
- Birden fazla alt satıcı desteklenir: aynı `merchant_oid` için birden
  fazla platform transfer talebi (birden fazla satıcılı bir
  rezervasyon/sipariş).
- Alt satıcı ödemesi **satıcı IBAN'ına** platform transferiyle yapılır;
  satıcıların kendi PayTR mağaza hesabı bulunması şart değildir
  [VARSAYIM: bu akış pazaryeri için varsayılan; alternatif model §15.1'de].
- Transfer sonucu callback: `trans_ids` JSON + hash doğrulama;
  yanıt `OK`.

### 9.3 İade [P3]

- Endpoint: `https://www.paytr.com/odeme/iade`; alanlar: `merchant_id`,
  `merchant_oid`, `return_amount` (nokta ayracı, ör. `10.25`),
  `paytr_token` (HMAC: `merchant_id + merchant_oid + return_amount +
  merchant_salt`).
- Kısmi iade desteklenir ("siparişe ait tutarın bir kısmı veya tamamı");
  toplam iade ödemeyi aşamaz (hata örneği 006).

### 9.4 Durum sorgu / mutabakat [P4]

- Belirsiz durumda kalan ödemeler için Durum Sorgu API ile
  `merchant_oid` bazında durum teyidi yapılır; mutabakat job'i
  (günlük) `payments` ile PayTR görünümünü karşılaştırır.

### 9.5 Tutar/para birimi kontrolü, tekrar deneme

- Tüm tutarlar kuruş (×100) kabul edilir ve `int64`'tir; ondalık float
  kullanılmaz.
- Para birimi: istek ve bildirimde `TL|USD|EUR|GBP|RUB` alanı; platform
  [VARSAYIM] olarak `TL` dışını ilk fazda reddeder.
- Ağ hatası ve yeniden deneme: geçici hata/5xx durumunda istekler
  idempotent anahtarla (`trans_id`/`merchant_oid`) yeniden gönderilir;
  `success` dönmüş işlemi tekrar göndermek yerine durum sorgusu yapılır.

## 10. Komisyon Önceliği ve İşlem Anında Oran Sabitleme

- **Çözüm sırası:** service > seller > category > global kuralı;
  aynı seviyede küçük `priority` kazanır; çelişen çoklu kural
  yapılandırma hatasıdır (kayıt + panel uyarısı).
- **İşlem anında sabitleme:** `payments.paid` anında hesaplanan oran
  `payment_allocations.commission_rate_bp` alanına kopyalanır; sonradan
  admin oranı değiştirse geçmiş işlem değişmez (K6); kaynak kural
  `commission_basis_ref` ile kayda yazılır.
- **PayTR işlem ücreti ile platform komisyonu ayrıdır:** PayTR'in
  mağazaya uyguladığı işlem komisyonu ayrı bir kalemdir ve platform
  komisyonuna karıştırılmaz. `payment_allocations`:
  `seller_amount` = satıcı payı, `platform_amount` = paid − seller
  payı; PayTR'in kendi işlem ücreti platform komisyonu hesabına
  girmeden, PayTR hesabıyla ayrıca yansıtılır
  [KARAR: muhasebe gösterimi mali müşavir görüşü ile netleşecek].
- **Stopaj [P1, doğrulanmış]**: 7524 sayılı Kanun ile pazaryerinde
  %1 stopaj 01.01.2025'ten itibaren uygulanır; PayTR örneğinde hakediş
  KDV ve vergiler hariç net satış tutarı üzerinden hesaplanır.
  Muhasebeleştirme detayı mali müşavir onayı ile teyit edilecek (§15).

## 11. Güvenlik, KVKK, Veri Minimizasyonu, Denetim İzi

- **Veri minimizasyonu:** log'larda tam e-posta/telefon yok; kart verisi
  platformda asla tutulmaz (PayTR'in sorumluluğundadır); satıcı
  IBAN/vergi verisi yalnız settlement gereği tutulur ve
  admin raporlarında maskelenir.
- **KVKK:** açık rıza metinleri ve veri sorumlusu tanımlaması
  hukuk görüşüne göredir; [VARSAYIM] platform veri sorumlusudur; satıcı
  belge/banka verisi sözleşme temellidir.
- **Denetim izi:** yönetimsel işlemler (`admin_audit`) `before/after`
  ve gerekçe ile kayıt olur; **ödeme/iade/transfer geçmişi append-only**
  UPDATE/DELETE yasaktır.
- **Gizli anahtar yönetimi:** `merchant_key`/`merchant_salt` yalnızca
  sunucu tarafı secret yönetiminde; repoya, geliştirme ortamı
  dosyalarına, loglara ve istemciye hiçbir biçimde yazılmaz; admin
  panelinde maskeli gösterilir.
- **Erişim kontrolü:** rol bazlı; başkasının booking kaynağına erişimde
  404;
  webhook (Bildirim URL) auth muaf, **hash doğrulama** + ayrı
  rate-limit; hash doğrulamaz tüm bildirimler yok sayılır ve olay
  güvenlik logu yazılır (K4).
- **Sahtekârlık önlemleri:** fiyat hiçbir zaman istemciden alınmaz
  (sunucuda tanımlıdır); `test_mode` üretimde kabul edilmez ve olay
  loglanır; her callback idempotent işlenir (eşzamanlı çift bildirimde
  tek `paid` — K3).
- **Log maskeleme:** webhook günlüğünde kişisel alanlar ve gizli
  anahtar alanları maskelenir; maskeleme politikası operasyon
  dokümanında tanımlanır [VARSAYIM].

## 12. Yönetim Paneli Ekranları

| Ekran | İşlev | Öncelik |
|---|---|---|
| Satıcı onay listesi | pending satıcılar, belge/durum; onay/red (gerekçe) | P0 |
| Rezervasyonlar | filtre (mod, durum, tarih); detay, iptal | P0 |
| Ödemeler | PayTR durum bilgisi, failed nedeni, mutabakat | P1 |
| Komisyon kuralları | CRUD + bir işlem için önizleme | P1 |
| Hakediş planı | planned transfers listesi; onay → Platform Transfer | P1 |
| İadeler | talep, değerlendirme, uygulama | P1 |
| Raporlar | ciro, komisyon, satıcı bazlı tahsil/hakediş, jeton ekonomisi | P1 |
| Sistem ayarları | faz durumu, Bildirim URL bilgisi (masked env) | P2 |

## 13. Test Senaryoları ve Kabul Kriterleri

**Kabul kriterleri:**

| # | Kriter |
|---|---|
| K1 | Ücretsiz modda rezervasyon talebi → kabul → booking oluşur; `payment_mode='free'`; **payments kaydı OLUŞTURULMAZ** (sahte ödeme yok); komisyon hesaplanmaz |
| K2 | PayTR modda booking yalnız `payments.status=paid` sonrası `confirmed` olur; `merchant_ok_url?success=1` dönüşü booking'i ilerletemez |
| K3 | Aynı `merchant_oid` ile tekrar bildirim → tek `paid`; ikinci bildirim `OK` döner, hiçbir alan tekrar güncellenmez |
| K4 | Geçersiz `hash` bildirimi → kayıt yazılmaz; güvenlik olayı loglanır |
| K5 | `payments.amount != bookings.total_amount` → tutar uyuşmazlığı red; booking ilerlemez |
| K6 | İşlem anı oran sabitleme: %8 kuralda ödeme kapanır; sonra kural %20'ye çekilir; eski allocation %8 kalır |
| K7 | Kısmi iade: 1000 kuruş ödemeden 400 iade → toplam iade limiti korunur; allocation negatife düşmez |
| K8 | PayTR adaptörü mock ile değiştirilir; booking/commission çekirdeği değişmez |
| K9 | `approved` olmayan satıcı için transfer planlanamaz |
| K10 | Admin olmayan admin endpoint → 403; başkasının booking → 404 |

**Seçilmiş test senaryoları:**

- T1: Ücretsiz mod uçtan uca; DB'de payments/refunds boş kalır.
- T2: Eşzamanlı çift Bildirim (network race) → tek `paid`/tek allocation.
- T3: Çoklu alt satıcı rezervasyon: tek `merchant_oid`, birden fazla
  allocation; Platform Transfer talebi satıcı sayısı kadar işler [P1].
- T4: "Aynı gün transfer talebi" girişiminde scheduler talebi ertesi
  güne erteler; aynı gün kuralı kodla zorlanır [P1].
- T5: `submerchant_amount = 0` (tümü platform) senaryosu [P1 örtük
  örnek] — platform gelir satışı için geçerli.
- T6: Kısmi iade ve sonrasındaki hakediş planının yeniden hesaplanması;
  settled=true işlemlerde iade davranışı danışma sorusu (§15.5).

## 14. Aşamalı Uygulama Planı

| Faz | İçerik | Çıkış kriteri |
|---|---|---|
| **Faz 0 — Ücretsiz MVP** | Satıcı onayı, katalog, platform içi iletişim, teklif + rezervasyon talebi; `payment_mode='free'` sabit | K1, K10 |
| **Faz 1 — Ekonomi (jeton/paket) manuel onay** | Ürün kataloğu, sipariş, manuel ödeme onayı, jeton/abonelik satışı (bkz. `prd.md`, `references/L2.md`) | L2 kabul kriterleri |
| **Faz 2 — PayTR hazırlık** | PayTR pazaryeri sözleşmesi; **alt üye işyeri/belge/banka onay akışının yazılı teyidi**; Bildirim URL altyapısı; test modu; satıcı ödeme profili akışı | K2, K3, K4, K5 |
| **Faz 3 — PayTR entegrasyonu** | Tahsil + durum makinesi, komisyon sabitleme, hakediş planı + Platform Transfer, İade | K6, K7, K9 |
| **Faz 4 — Gelir modelleri** | Satıcı aboneliği, öne çıkarma; istenirse PayTR kart saklama/tekrarlayan ödeme ile abonelik tahsilatı | — |

## 15. Riskler, Açık Kararlar ve Danışma Soruları

**⚠️ Geliştirme öncesi yazılı teyit gerektiren dış bağımlılıklar:**

1. **Alt üye işyeri akışı:** PayTR dev dokümanında otomatik alt-üye
   işyeri onboarding API'si belgelenmemiştir; dokümandaki pazaryeri
   akışı **Platform Transfer Talebi**'dir [P1]. "PayTR'nin onayladığı
   alt üye işyeri" modelinde satıcı belge/IBAN onayı nasıl yürütülür
   PayTR ile yazılı teyit gerektirir.
2. **Hizmet/rezervasyon uygunluğu:** rezervasyon/hizmet sektörünün
   PayTR kabul koşulları ve gerekli sözleşme şartları yazılı teyit.
3. **Kapora ve kalan ödeme:** "kapora = kısmi tahsil / ön provizyon"
   modelinin PayTR destek sözleşmesindeki karşılığı; Ön Provizyon
   entegrasyonu (<https://dev.paytr.com/on-provizyon>) kapora için
   uygun mu? Provizyon sonlandırma/dekontlama süreleri? (dev dokümanı
   mevcut; kapora senaryosuna birebir oturması teyit gerektirir).
4. **İleri tarihli organizasyonlarda hakediş/transfer zamanı:** Platform
   Transfer'de "ödemeyle aynı gün talep olmaz; talep 10:00'a kadar"
   kuralı dokümandadır [P1]; organizasyon tarihine göre transfer planı
   ve sözleşme koşulları yazılı teyit.
5. **Transfer sonrası iade:** transfer yapılmış/hakediş gönderilmiş
   işlemlerde kısmi iade hangi paylardan düşer (satıcı payı mı daha
   sonraki hakedişten mi) — yazılı teyit.
6. **Yurt dışı ödeme:** callback'te `USD/EUR/GBP/RUB` para birimleri
   görünür [P0]; yabancı kart kabulü, kur/döviz komisyonu ve
   settlement kuralları PayTR teyidi (ilk faz TL-only plan [VARSAYIM]).
7. **Satıcı kabul koşulları:** PayTR üye işyeri/pazaryeri kabul
   şartları (sektör, vergi kaydı, iade oranı vb.) — satıcı onboarding
   formuna yansıtılır.

**Açık kararlar [KARAR]:**

- A1: Ücretsiz modda satıcıyla anlaşılan tutar booking'e rapor amaçlı yazılsın mı? (öneri: yazılır, payments üretilmez)
- A2: PayTR işlem ücretinin muhasebesel gösterimi (mali müşavir ile).
- A3: Satıcı profil onayını platform mu (belge inceleme) yoksa PayTR mi (kabul) yürütür?
- A4: İletişim formu mesaj sayısı limitleri ve spam koruması kapsamı (P2).

**Mali müşavir/hukuk soruları:**

- Stopaj %1'in beyan edilme yükümlülüğü ve hakedişten kesinti muhasebe
  fişi [P1 dokümante ama mali onay gerekli].
- Platform komisyonunun KDV gösterimi; e-arşiv fatura modeli.
- KVKK: veri sorumlusu kimlikler, satıcı belge/banka verisinin
  saklama süresi ve hukuki temeli, mesaj kayıtlarının saklama süresi.
- Mesafeli satış (uzaktan satış) kuralları, iptal/iade hakkı,
  satıcının sorumluluğu (pazaryeri aracı sorumluluğu).

---

### Kaynakça (PayTR resmi geliştirici dokümanı, doğrulama: 2026-11-02)

- Direkt API 2. Adım (Bildirim URL alanları, hash formülü, `OK` yanıtı,
  idempotency uyarıları, hata kodları):
  <https://dev.paytr.com/direkt-api/direkt-api-2-adim>
- Platform Transfer Talebi (transfer alanları/hash, tek ve çoklu alt
  satıcı örnekleri, `submerchant_amount=0` örneği, 10:00/ertesi gün
  kuralı, %1 stopaj tablosu ve callback):
  <https://dev.paytr.com/platform-transfer-talebi/transfer-talimatinin-verilmesi>
- İade API (alanlar, kısmi iade, bakiye aşım hata örnekleri):
  <https://dev.paytr.com/iade-api>
- Geliştirici dokümanı ana sayfa (Durum Sorgu, Ön Provizyon, kart
  saklama/tekrarlayan ödeme modülleri talimatları):
  <https://dev.paytr.com/>
