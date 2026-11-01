# Core Platform — Şemsiye PRD

| | |
|---|---|
| **Sürüm** | 2.0 |
| **Tarih** | 2026-11-01 |
| **Durum** | Onaylandı |
| **Rol** | Şemsiye belge — seviye belgelerinin üstünde |
| **Seviye belgeleri** | `L0.md` · `L1.md` · `L2.md` · `L3.md` |

> Bu belge **ne inşa edileceğini** ve **neyin nereye ait olduğunu** tanımlar.
> Ayrıntılar seviye belgelerindedir. Bir çelişki olursa seviye belgesi geçerlidir.

---

## 1. Problem

Sekiz ayrı konu uygulaması yapıyorum: CodeScope, Mimariscope, webscope,
maketscope, astroscope, eduscope, fitscope, financescope. Hepsi aynı altyapıya
ihtiyaç duyuyor: kayıt/giriş, jeton cüzdanı, ödeme, dosya, bildirim, arama, AI.

Her yeni uygulamada bu listeyi sıfırdan yazıyorum. Sonuç: her uygulamada farklı
hatalar — birinde CSRF açık, diğerinde kullanıcı ID'si sabit, üçüncüsünde kullanıcı
kendi jetonunu üretebiliyor.

**Asıl problem tekrar etmek. Dokümantasyon eksikliği değil.**

---

## 2. Çözüm

### 2.1 Tek cümle

Ortak altyapıyı **bir kez** yaz; her uygulama onu sürümlenmiş bir Go kütüphanesi
olarak tüketir ve tek komutla çalışan bir iskeletle başlar.

### 2.2 Üç fikir

**Fikir 1 — Çekirdek bir kütüphanedir, şablon değil.**
`core-platform` ayrı bir git deposu ve ayrı bir Go modülüdür. Uygulamalar `go get`
ile tüketir. Kopyalama yok → kopyaların birbirinden uzaklaşması (drift) yok.

**Fikir 2 — Ayrıştırma günü yoktur.**
Ayrım ilk günde kurulur: ayrı `go.mod`, ayrı depo, tek dar arayüz (`contract`).
Geliştirme sırasında `go.work` ile çekirdek serbestçe değiştirilir; yayına çıkarken
`go.work` silinir ve sürüm sabitlenir.

**Fikir 3 — Tekrarı `corectl` ortadan kaldırır.**
Yeni uygulama `corectl new` ile başlar. Belgeler bu komutun **çıktısını** tarif eder;
modül iç tasarımını değil.

### 2.3 Başarı nasıl görünür

```bash
corectl new fitscope --level=L1 --only=identity,files
cd fitscope && docker compose up
# 30 saniye içinde /readyz 200, kayıt → giriş → /me çalışıyor
```

Sen `internal/` altına Fitscope'un iş mantığını yazmaya başlarsın.
Bir yıl sonra Eduscope için aynı komut, farklı `--only`, **sıfır tekrar.**

---

## 3. Seviyeler

### 3.1 Yapı — kümülatif, doğrusal değil

```
L0  Çekirdek kernel                    (saf altyapı, domain yok)
L1  L0 + kimlik + ortak servisler
L2  L1 + ekonomi
L3  L1 + AI        (L2 opsiyonel)
```

- **L2 ve L3 kardeştir**, birbirini gerektirmez.
- Bir uygulama **tek bir seviye** seçer; o seviye alt seviyeleri içerir.
- Seviye, çekirdeğin değil **uygulamanın** özelliğidir. Çekirdek tek sürümdür.

### 3.2 Kapsam matrisi

| Modül | L0 | L1 | L2 | L3 |
|---|:--:|:--:|:--:|:--:|
| config · logger · db · migration · module · http · health | ✅ | ✅ | ✅ | ✅ |
| identity (users + auth) | | ✅ | ✅ | ✅ |
| files | | ✅ | ✅ | ✅ |
| notifications | | ✅ | ✅ | ✅ |
| economy (wallet + orders + entitlements) | | | ✅ | |
| ai (gateway + kota + cache + prompt) | | | | ✅ |
| `contract/core` | ✅ | ✅ | ✅ | ✅ |
| `contract/economy` | | | ✅ | |
| `contract/ai` | | | | ✅ |

**Arama çekirdekte değil.** `helper/search` altında yardımcı paket olarak verilir;
aranacak alanlar, skorlama ve filtreler her uygulamada farklıdır.

### 3.3 `--only` — seçmeli üretim

```bash
corectl new xscope --level=L1 --only=identity,files
```

Bu bir çalışma zamanı anahtarı **değildir**. Sadece üretilen kodu belirler:
seçilmeyen modülün dosyası, migration'ı, route'u hiç oluşmaz.

Tak/çıkar mekanizması (tier registry, capability) **yoktur ve olmayacaktır.**
Gerekçe: her uygulama zaten ayrı binary olduğu için çalışma zamanı anahtarı
gereksiz; modül bağımlılığı `Module.Dependencies()` ile zaten denetleniyor.

---

## 4. Depo ve modül düzeni

```
<workspace>/
├── core-platform/                  ayrı git deposu, kendi go.mod'u
│   ├── go.mod                        module core-platform
│   ├── contract/                     TEK dışa açık arayüz
│   │   ├── core/                     Module, Identity, Storage, Notifications, hatalar
│   │   ├── economy/                  Wallet, Entitlements, Payments
│   │   └── ai/                       AIProvider
│   ├── platform/                     L0 kernel
│   ├── mod/                          L1+ modülleri
│   │   ├── identity/ files/ notifications/
│   │   ├── economy/                  L2
│   │   └── ai/                       L3
│   ├── helper/search/                çekirdek dışı yardımcı
│   ├── migrations/<modül>/
│   ├── scaffold/                     corectl şablonları
│   └── cmd/corectl/
│
└── <app>/                          ayrı git deposu, ayrı binary, ayrı veritabanı
    ├── go.mod                        require core-platform vX.Y.Z
    ├── internal/                     uygulamaya özel
    ├── migrations/                   uygulamaya özel
    └── cmd/server/
```

### 4.1 Sınır kuralları (CI'da zorunlu)

| Kural | Nasıl zorlanır |
|---|---|
| Uygulama yalnızca `contract/**` import eder | depguard: `platform/**`, `mod/**` importu → build kırmızı |
| `platform/**` hiçbir `mod/**` import etmez | depguard |
| L1 modülleri (`identity`, `files`, `notifications`) `mod/economy` veya `mod/ai` import edemez | depguard — **L1-only derlemesinin garantisi** |
| Uygulama başka uygulamayı import etmez | ayrı depo + ayrı modül |
| DB erişimi yalnızca repository katmanında | code review |
| `echo.Context` service katmanına sızmaz | code review |

### 4.2 `contract` semver politikası

| Sürüm | Kural |
|---|---|
| **patch** `v0.4.1` | Davranış düzeltmesi; arayüz imzası değişmez |
| **minor** `v0.5.0` | Yalnızca **ekleme**. Kaldırma, imza değişikliği, davranış kırılması yasak. Mevcut uygulamalar derlenmeye devam eder. |
| **major** `v1.0.0` | Kırıcı değişiklik. Geçiş notu (`MIGRATION.md`) zorunlu. |

Her sürümde CI, `demo-l1` **ve** `demo-l3` iskeletlerini üretip derler.
Bu kapı olmadan semver bir söz değil, temennidir.

### 4.3 Yerel geliştirme

| Aşama | Yöntem |
|---|---|
| Geliştirme | Workspace kökünde `go.work`; iki depo `use` altında (uygulama deposunda `.gitignore`) |
| Yayın öncesi | `go.work` silinir; `require core-platform vX.Y.Z` |
| Güncelleme | `corectl upgrade` → `go get` + config/migration farkı |
| Acil yama | Geçici `replace`; commit'lenmez |

---

## 5. `corectl` — tekrarı ortadan kaldıran araç

| Komut | Ne yapar |
|---|---|
| `new <app> --level=LN [--only=...]` | Çalışan iskelet: kod, config, migration, Docker, Makefile, 2 örnek test |
| `add module <name>` | Uygulamaya domain modülü ekler (model/repo/service/handler + migration + test) |
| `add level <LN>` | Alt seviyeyi ekler: `main.go`, migration, config, `.env.example` güncellenir |
| `upgrade` | Çekirdek sürümünü uygular; kırıcı değişiklikte **durur ve rapor verir** |
| `doctor` | Migration durumu, ledger invariant'ı, eksik config, L1-only derleme kontrolü |

`upgrade` olmadan her uygulama kendi `main.go`'sunu elle günceller → 8 uygulama 8
farklı sürümde takılır. Yani tekrar sorununu çözerken bakım sorunu yaratılır.

### 5.1 Golden path — 30 saniye kapısı

`corectl new xscope --level=L1` → `docker compose up` → `/readyz` 200.

Docker imajları önbellekteyken bu ölçülür. Aşarsa **L0 kabul kriteri başarısız**
sayılır — hedef değil, kapı.

---

## 6. Mezun etme (graduation)

Uygulama bir çekirdek modülünden ayrışmak istediğinde:

| Durum | Yapılacak | Kural |
|---|---|---|
| Eksik/genel özellik | Çekirdeğe ekle → sürüm artır → `upgrade` | Tercih edilen |
| Davranış uygulamaya özel | Uygulama `contract` arayüzünü kendi implemente eder | Temiz ayrılık |
| Acil, çekirdeğe uymuyor | `<app>/internal/_fork/<modül>` + `FORK.md` (gerekçe, tarih, upstream planı) | **Geçici** |

Bir fork **6 ay** yaşarsa ya çekirdeğe geri taşınır ya da uygulamanın kalıcı parçası
ilan edilir. Sessizce yaşlanamaz.

---

## 7. Çekirdeğe giriş eşiği

Bir özellik çekirdeğe girer ancak:

1. en az **2 farklı uygulamada aynı biçimde** gerekiyorsa, **ve**
2. sağlayıcıdan bağımsız bir arayüzle ifade edilebiliyorsa, **ve**
3. uygulamaya özel domain bilgisi içermiyorsa.

Aksi halde uygulamada kalır.

| Modül | Karar | Gerekçe |
|---|---|---|
| `identity`, `files`, `notifications` | L1 | Her uygulamada aynı biçimde gerekli |
| `economy`, `ai` | L2 / L3 | Ayrı kardeş seviyeler |
| `search` | **Yardımcı paket** | Alanlar ve skorlama uygulamaya özel |
| `community`, `analytics`, `lifeweave`, `rewarded_ads` | **Uygulamaya ait** | Domain'e özgü |

---

## 8. Öncelik sırası ve yol haritası

`corectl` ikinci iş değil, **L0 ile aynı iştir.** L0 bitmiş sayılması için
`corectl new --level=L0` çalışıyor ve golden path 30 saniyede tamamlanıyor olmalı.

| # | Adım | Kapı |
|---|---|---|
| 1 | Depo kurtarma + `go.mod` | `go build ./...` yeşil |
| 2 | `platform/` kernel + `corectl` | `L0.md` kabul kriterleri |
| 3 | Golden path | `demo-l0` 30 saniyede `/readyz` 200 |
| 4 | L1: identity + files + notifications | `L1.md` kabul kriterleri |
| 5 | L2: ekonomi | `L2.md` kabul kriterleri |
| 6 | L3: AI | `L3.md` kabul kriterleri |

Her adım bağımsız commit ve yeşil CI. Yeşil olmadan sonrakine geçilmez.

**Belgeler koda paralel yazılır.** `L1.md` L1 koduyla, `L2.md` L2 koduyla,
`L3.md` L3 koduyla birlikte çıkar. Hiç çalışan uygulama yokken üst seviye PRD
yazmak, kaçınmaya çalıştığımız hatanın ta kendisidir.

---

## 9. CI kapıları

```
gofmt -l .              boş
go vet ./...            temiz
staticcheck ./...       temiz
govulncheck ./...       temiz
go test -race ./...     yeşil
go build ./...          başarılı
depguard                sınır kuralları (4.1) ihlal edilmemiş
gen-demo l1             demo-l1 üretilip derlenmiş
gen-demo l3             demo-l3 üretilip derlenmiş
```

---

## 10. Kapsam dışı

- Uygulamaya özel domain kodları (astroloji, eğitim, finans modelleri)
- Mobil/web istemciler
- Kubernetes, çok bölgeli dağıtım
- SSO / merkezi kimlik servisi (kapı açık: `contract/core.Identity` dış sağlayıcıya
  delege edilebilir tasarlanır, V1'de uygulanmaz)
- Community, analytics, lifeweave, rewarded ads
- i18n içerik çevirisi (`en` + `tr` iskeleti yeterli)

---

## 11. Notlar

### 11.1 Onaylanmış kararlar

| # | Karar |
|---|---|
| K1 | Çekirdek sürümlenmiş Go modülü + `corectl upgrade` |
| K2 | SSO kapısı açık, bugün ayrı hesaplar |
| K3 | Her uygulama kendi ödeme/AI anahtarları |
| K4 | Ödeme L2'de, AI L3'te — kardeş seviyeler |
| K5 | Community/Analytics/Rewarded Ads çekirdekte değil |
| K6 | Tak/çıkar (tier/capability) yok; seviye üretim zamanı kararı |
| K7 | L0 saf çekirdek; files + notifications L1'de |
| K8 | `search` çekirdek dışı yardımcı paket |
| K9 | `corectl` L0 ile aynı iş; golden path 30 saniye |

### 11.2 Önceki plandan sapmalar

| Konu | Önceki plan | Bu PRD |
|---|---|---|
| Kapsam | 15 modül, 33–55 iş günü | Seviyeli, koda paralel belge |
| Route öneki | `/api/v1/apps/<app>` | `/api/v1/...` (tek uygulama = tek binary) |
| Modül yeri | Tek depo içinde `apps/*` | Ayrı depo |
| Ödeme | 3 sağlayıcı V1'de | Motor L2'de; Stripe/PayTR adaptör olarak sonra |
| Tier sistemi | Yoktu, eklendi, sonra kaldırıldı | **Yok** (K6) |
