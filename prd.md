# PRD — Jeton Ekonomisi ve Ödüllü Reklam (core-platform L2)

| | |
|---|---|
| **Belge** | `prd.md` |
| **Üst belge** | `PRD.md` (platform seviyesi) |
| **Teknik şartname** | `references/L2.md` (rev 1.1) |
| **Tarih** | 2026-11-02 |
| **Durum** | Onaylı — kod henüz yazılmadı |

---

## Problem

Platform ücretsiz üyelikle açılıyor, ama premium özelliklerin parası
"jetonla" ödenecek. Bugün bu üç sorun var:

1. Kullanıcının jeton almasının tek yolu ödeme sayfası — jetonu bedava
   kazanabilen bir yol yok; ücretsiz kullanıcı hiçbir değişim yapamaz.
2. Para ile jeton satın almak isteyen için satış kanalı, paket fiyatı ve
   onay akışı yok.
3. Yönetici ekonomiyi yönetemiyor: reklam programını işletemiyor,
   hoş geldin jetonu ve günlük/aylık kazanma limitlerini kod yazmadan
   değiştiremiyor.

> **Ödeme notu:** Gerçek ödeme sistemi (POS/gateway entegrasyonu) ayrı bir
> prompt ile sonra ayrı bir şartnameye alınacaktır (bkz. `Odeme_mimarisi.md`).
> Bu PRD ödemeyi "sipariş + ödeme sağlayıcı arayüzü + manuel onay"
> soyutlamasıyla çözer; jeton paketleri ve abonelik ilk fazda manuel onay
> ile satılabilir.

## Çözüm

L2 seviyesinde `mod/economy` modülü gelir: kullanıcı cüzdanı (append-only
ledger), ürün kataloğu, sipariş + ödeme adaptör arayüzü, entitlement ve
**ödüllü reklam** alt paketi. Davranış:

- Ücretsiz üyelik herkes için açık; üyenin cüzdanı ilk açıldığında
  **10 jeton** hoş geldin jetonu (varsayılan) tek kez kredilenir.
- Kullanıcı reklam izleyerek (geri sayımlı süre) veya reklama tıklayarak
  **günde en fazla 3 jeton** kazanır; bir takvim ayındaki kazanma toplamı
  **10 jetonu geçemez**.
- Tüm bu limitler **admin panelden değiştirilebilir** (kod dağıtımı yok).
- Yönetici reklam yönetim sayfalarında video/resim/metin reklam
  oluşturur; her reklamın izleme süresi ve opsiyonel **affiliate URL**'i
  olur; tıklamalar affiliate adresine yönlendirilir ve sayılır.
- Kullanıcı para ile **10/50/100 jeton paketleri** satın alır veya
  **abonelik** satın alır; ilk fazda ödeme banka talimatı + admin
  onayı ile yürür, gerçek sağlayıcı sonraki fazda adaptör olarak gelir.
- Her adımda ekonomik invariantlar (bakiye ≥ 0, SUM(delta) = balance,
  tek çekilmiş ödül) korunur; tüm bakiye hareketleri tekil ve
  idempotenttir.

## Kullanıcı Hikayeleri

### Ücretsiz üye

1. Bir ücretsiz üye olarak, cüzdanımı ilk açtığımda otomatik 10 jeton
   almak istiyorum, böylece premium özellikleri hemen deneyebilirim.
2. Bir ücretsiz üye olarak, bir reklamı geri sayımla izleyerek jeton
   kazanmak istiyorum, böylece ödeme yapmadan jeton toplayabilirim.
3. Bir ücretsiz üye olarak, reklamı tıklayarak da jeton kazanmak
   istiyorum, böylece süre beklemek istemeyenler için alternatif yol olur.
4. Bir ücretsiz üye olarak, bugün ve bu ay kazanabileceğim kalan jeton
   hakkını görmek istiyorum, böylece planlarım.
5. Bir ücretsiz üye olarak, limiti doldurduğumda net bir mesaj görmek
   istiyorum, böylece ne zaman tekrar kazanabileceğimi bilirim.
6. Bir ücretsiz üye olarak, bakiyemi ve tüm jeton geçmişimi tek ekranda
   görmek istiyorum, böylece jetonun nereden gelip nereye gittiğini anlarım.
7. Bir ücretsiz üye olarak, bir reklamın bana günde en çok bir kez ödül
   vermesini istiyorum, böylece ekonomi adil kalır.

### Ödeme yapan kullanıcı

8. Bir kullanıcı olarak, 10/50/100 jeton paketlerini fiyatlarıyla
   görmek istiyorum, böylece bana uygun paketi seçebilirim.
9. Bir kullanıcı olarak, siparişimi tamamladığımda ödeme talimatını
   (referans kod, hesap bilgisi) almak istiyorum, böylece bankadan
   havale yapabilirim.
10. Bir kullanıcı olarak, satın aldığım aboneliğin bitiş tarihini görmek
    istiyorum, böylece yenilemeyi planlayabilirim.
11. Bir kullanıcı olarak, aboneliği yeniden satın alarak uzatmak
    istiyorum, böylece premium haklarım kesintiye uğramaz.
12. Bir kullanıcı olarak, aynı satın alma isteğini iki kez göndersem bile
    bir kez işlenmesini istiyorum, böylece çift tıklama paramı harcamaz.
13. Bir kullanıcı olarak, yetersiz bakiyede net bir hata görmek istiyorum,
    böylece ne yapacağımı bilirim.

### Yönetici

14. Bir reklam yöneticisi olarak, video/resim/metin reklam
    oluşturmak/düzenlemek/yayından çıkarmak istiyorum, böylece reklam
    programını kontrol edebilirim.
15. Bir reklam yöneticisi olarak, reklamda affiliate URL tanımlamak
    istiyorum, böylece tıklamalar gelire bağlanır ve iz ölçülür.
16. Bir reklam yöneticisi olarak, her reklamın izleme süresini belirlemek
    istiyorum, böylece jeton karşılığı süreyi kontrol ederim.
17. Bir yönetici olarak, hoş geldin jetonunu, izleme ödülünü ve
    günlük/aylık limitleri panelden değiştirmek istiyorum, böylece
    ekonomiyi kod yazmadan ayarlarım.
18. Bir yönetici olarak, tıklama ödülünü açıp kapatmak ve miktarını
    değiştirmek istiyorum, böylece sahte tıklama gündem olduğunda
    kod yazmadan önlem alırım.
19. Bir yönetici olarak, reklam başına gösterim/izleme/tıklama/ödül
    istatistiklerini görmek istiyorum, böylece hangi reklamın işe
    yaradığını anlarım.
20. Bir yönetici olarak, manuel ödeme akışında sipariş
    onaylayıp reddetmek istiyorum, böylece banka havale satışları
    elle işletilür.
21. Bir yönetici olarak, cüzdana admin kaynaklı jeton ekleyip çıkarırken
    sebebini kaydetmek istiyorum, böylece destek talepleri denetlenebilir
    çözülür.
22. Bir yönetici olarak, ciro, dolaşımdaki jeton ve abone sayısı özetini
    görmek istiyorum, böylece ekonominin sağlığını ölçerim.

### Mimar / güvenlik

23. Bir mimar olarak, ödül miktarının istek gövdesinden asla alınmamasını,
    yalnız sunucu-veritabanından gelmesini istiyorum, böylece manipüle
    edilemez.
24. Bir mimar olarak, izleme oturumunun sunucu saatiyle doğrulanmasını
    istiyorum, böylece istemci "30 saniyeyi 1 saniyede tamamladım" diyemesin.
25. Bir mimar olarak, ödellerin (kullanıcı, reklam, gün) bazında
    tekil olmasını istiyorum, böylece çift ödül imkânsız olsun.
26. Bir mimar olarak, bakiyenin negatife düşememesini ve defterin
    append-only olmasını istiyorum, böylece denetim izi güvenilir kalır.

## Uygulama Kararları

- **Modül yapısı:** `mod/economy` tek modül; alt paketler `wallet`,
  `orders`, `entitlements`, `ads`. Reklam ayrı modül olmaz — ödül akışı
  doğrudan ledger'a yazılır.
- **Para birimi modeli:** Jeton bir **kod** (örn. `TOKEN`); tüm tutarlar
  `int64` minor unit (1 jeton = 1). Hiçbir yerde `float` yok.
- **Bakiye güncelleme:** tek koşullu `UPDATE ... WHERE balance + delta >= 0`
  (yarışsız); her değişim aynı transaction'da `wallet_entries` satırı
  yazar; `SUM(delta) == balance` invariant'ı `corectl doctor` doğrular.
  Append-only: geçmiş satır UPDATE/DELETE kabul edilmez.
- **Hoş geldin jetonu:** kayıtta değil, cüzdanın ilk erişiminde
  (`GET /wallet`) açılır; `signup_grant` (varsayılan 10) tek kez
  `idempotency_key = signup_grant:<userID>` ile kredilenir.
- **Ödüllü reklam akışı (izleme):**
  1. `GET /ads/token` → uygun (limit aşımı yok) jetonlu aktif reklam seçilir.
  2. `POST /ads/:id/view` → sunucu damgalı izleme oturumu
     (`required_seconds`, `server_started_at`, `expires_at`).
  3. İstemci geri sayımı yerel UI'da gösterir; süre bitince
     `POST /ads/view/:sessionID/claim` → sunucu,
     `now() ≥ started + required_seconds` kontrolü yapar;
     aksi `REWARD_TOO_EARLY` (409), ödül yazılmaz.
  4. Limit kontrolü: günlük 3 (varsayılan), aylık 10 (takvim ayı);
     aşılırsa `REWARD_LIMIT_REACHED` (429), hareket yazılmaz.
  5. Ödül tek kez, `idempotency_key = ad_watch:<sessionID>`;
     oturum `rewarded` olup tekrar claim aynı sonucu döner.
- **Tıklama ödülü:** `GET /ads/:id/click` (auth şart) → gün bazlı
  unique tıklama kaydı → tık ödülü (aynı günlük/aylık havuzdan) →
  `302` affiliate URL'e redirect. Affiliate URL yalnız `https://`
  kabul edilir. Misafir tıklama stat olarak sayılır, ödül vermez.
- **Limit ayarları:** `reward_settings` tekil DB satırı; admin
  `PUT /admin/reward-settings` değiştirir (audit trail `updated_by`);
  config dosyası yalnız ilk seed.
- **Jeton paketleri ve abonelik:** `products.kind ∈ {token_pack,
  subscription, regular}`; `token_pack` → `meta = {"tokens": 50}`,
  `subscription` → `meta = {"entitlement": "premium.vip"}` + interval.
  Fulfillment, sipariş durum makinesiyle yapılır
  (`created → pending → paid → fulfilled / failed / expired`).
- **Ödeme soyutlaması:** `PaymentProvider` arayüzü; bu sürümde
  `token`, `manual` (admin onaylı banka talimatı) ve `test` adaptörleri
  var; gerçek sağlayıcı sonraki sürümde yalnız yeni adaptör olarak gelir,
  çekirdek değişmez.
- **Güvenlik:** istemciden hiçbir tutar kullanılmaz; Idempotency-Key
  siparişlerde zorunlu; başkasının cüzdanı/siparişi 404; admin
  endpoint'leri rol arkasında; webhook auth-muaftır ama imza doğrular.
- **Kötüye kullanım koruması:** reklam uçlerinde L1 rate limiter
  kullanılır; erken claim denemeleri `retry_after` ile cevaplanır.
- **Zamanlanmış işler:** süresi geçen siparişler (1 dk), biten
  abonelikler (1 saat), günlük reklam istatistiği ve ledger denetimi —
  `Clock` enjekte edilir, çok instance'da `pg_advisory_lock`.

## Test Kararları

- Test edilen şey **dışa dönük davranıştır**: HTTP API + service
  seviyesinde sonuçlar doğrulanır; iç ayrıntılar test edilmez.
- **Cüzdan (`mod/economy/wallet`):** yetersiz bakiye 402, idempotency,
  append-only yasak, `-race` altında 100 eşzamanlı harcama → bakiye
  negatif değil; `SUM(delta) == balance`.
- **Ödüllü reklam (`mod/economy/ads`):** erken claim reddi, tek ödül,
  günlük/aylık cap, gün bazlı tıklama tekili, https zorlaması, admin
  limit değişiminin sonraki ödüllerde etkili olması.
- **Siparişler (`mod/economy/orders`):** token_pack için manual onay
  uçtan uca jeton kredisi; abonelik entitlement'ı interval bitişli;
  `"?success=true"` gibi sahte dönüş siparişi ilerletemez.
- **API:** HTTP seviyesinde durum/hata kodları, auth korumaları ve
  aktör erişimi doğrulanır (kendi dışında 404, rol azsa 403).
- Referanslar: `mod/files/service/files_test.go` (fake store deseni),
  `mod/identity/api/handler_test.go` (HTTP test deseni).
- Ekonomi paket testleri gerçek PostgreSQL üzerinde koşar (`-race`
  Linux CI'da); CHECK + partial unique index gerektirir.

## Kabul Kriterleri

1. `corectl new demo --level=L2` üretilen proje elle düzenlemeden
   derlenir; `docker compose up` ile `/readyz` 200 döner.
2. Yeni üye `GET /wallet` çağırır → **tam bir kez** 10 jeton (varsayılan)
   `signup_grant` hareketi; tekrar çağrıda hareket yok.
3. Reklam izleme: sunucu saatinden 1 sn önce claim → 409 ve ödül yok;
   tam sürede claim → 1 jeton; ikinci claim → yeni hareket yazılmaz.
4. Günlük kazanma 3, aylık kazanma 10 (varsayılan) sunucu tarafında
   zorunlu; aşım girişimi 429 döner, hiçbir hareket yazılmaz.
5. İzleme ve tıklama ödülleri aynı günlük/aylık havuzdan düşer; toplam
   kazanma limiti aşılabilir.
6. Yönetici panelden reklam (video/resim/metin) CRUD yapabilir;
   https dışı affiliate URL 400 döner; tıklama 302 ile affiliate
   URL'e yönlendirilir ve ölçülür.
7. Yönetici hoş geldin jetonu, izleme ödülü, günlük/aylık limitleri
   ve tıklama ödülünü panelden değiştirebilir; değişim `updated_by`
   audit izi taşır ve sonraki ödüllere uygulanır.
8. Jeton paket satın alma: manual akışla sipariş → admin onayı →
   paketteki token sayısı tek kez `purchase` hareketiyle kredilenir.
   İstemciden gelen tutar hiçbir adımda kullanılmaz.
9. Abonelik satın alma fulfillment→ entitlement interval kadar açık;
   süresi bitince job pasif hale getirir (silmez).
10. Append-only ledger: geçmiş satır update/delete reddedilir;
    `SUM(delta) == balance` her zaman doğru (`corectl doctor`).
11. Kullanıcıya açık "istediğim kadar jeton ver" endpoint'i yok
    (route listesi testiyle kanıt).
12. `go test -race ./...` yeşil; ekonomi paketlerinde coverage ≥ %50.

## Kapsam Dışı

- **Gerçek ödeme sağlayıcı entegrasyonu (PayTR/Stripe)** ve
  **abonelik otomatik yenilemesi (recurring charge)** — ayrı şartname:
  `Odeme_mimarisi.md`; burada arayüz + manual + test adaptörüyle
  soyutlanır.
- **Üçüncü taraf reklam ağları** (AdSense, Unity Ads vb.) ve SSV
  (server-side verification) — reklamlar admin kataloglamadır.
- Kupon/indirim kodu; kullanıcılar arası jeton hediye; referral programı.
- Video transcode / CDN / thumbnail (L1 FileStore yeterli).
- Para birimi dönüşümü; fatura/vergi motoru.
- Jetonun süresi dolma (expire) mekanizması — job hazır ama
  varsayılan kapalı.

## Notlar

- Teknik detay (şema, durum makinesi, idempotency anahtarları, sebep
  kodları, API listesi, test listesi): `references/L2.md` rev 1.1.
- Ödeme mimarisi şartnamesi: `Odeme_mimarisi.md` (rezervasyon/
  pazaryeri platformu için ücretsiz mod + PayTR fazı).
- Platform taşıyıcılığı: jeton `TOKEN` varsayılanı ile gelir; uygulama
  config'te kendi kodunu tanımlayabilir (`credits`, `coins` vb.).
