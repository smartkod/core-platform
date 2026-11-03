# Core Platform

Sektör bağımsız **hizmet katmanı çekirdeği**. Bir işletmenin kim olduğunu, neyi
sunduğunu ve kimin neyi yapabileceğini tanımlar; randevu, sipariş, teklif, ders,
ödeme veya lojistik gibi **hiçbir sektör işlemini çalıştırmaz**.

Bu depo bir **Go backend kütüphanesidir**. `npm run dev` yoktur; tarayıcıda
açılacak bir sayfa yoktur. Servis bir HTTP API sunar.

## Katmanlar

| Katman | İçerik | Amaç |
|---|---|---|
| **L0** | `platform/`, `contract/core` | Kernel: config, PostgreSQL, migration, HTTP zarfı, health, modül yöneticisi |
| **L1** | `identity` · `authz` · `files` · `notifications` | Kimlik, yetki, dosya, bildirim |
| **L2** | `organization` · `catalog` | İşletme profili + şube + üyelik; merkezi taksonomi + sunum kaydı |

L2 bir işlem motoru değildir. Motorlar (randevu, sipariş, teklif, ödeme…)
çekirdeğin **sürümlenmiş sözleşmeleri** üzerinden bağlanır:
`doc/ENGINE_CONTRACT.md`.

## Hızlı başlangıç

```powershell
.\scripts\run.ps1
```

Elle:

```bash
docker compose up -d --build
curl http://localhost:8080/healthz
```

Ayrıntı: **`references/NASIL-CALISIR.md`**.

## Durum

| Alan | Durum |
|---|---|
| L0 kernel | ✅ çalışıyor |
| L1 identity · authz · files · notifications | ✅ çalışıyor |
| L2 organization · catalog | ✅ çalışıyor |
| Sektör motorları (randevu, sipariş, ödeme…) | ❌ bu depoda yok — bilinçli olarak |

`gofmt`, `go build ./...`, `go vet ./...`, `go test ./...` ve
`go run ./cmd/depcheck` yeşil.

## Mimari

```
core-platform/
├── contract/core/     uygulamanın görebildiği tek arayüz
├── platform/          L0 kernel (modül import etmez)
├── mod/               çekirdek modülleri (birbirini import etmez)
│   ├── identity/      hesap, oturum, token
│   ├── authz/         rol + izin + kapsam
│   ├── files/         yükleme ve erişim
│   ├── notifications/ outbox + e-posta
│   ├── organization/  işletme, şube, üyelik, bölge
│   └── catalog/       taksonomi, sunum, varyant
├── cmd/corectl/       iskelet üretici
└── doc/               ARCHITECTURE, ENGINE_CONTRACT, ADR
```

Sınırlar derleyiciyle ve CI ile korunur:

- `platform/` hiçbir `mod/` paketini import edemez.
- Modüller birbirini import edemez; `contract/core` üzerinden konuşur.
- `contract/` ne `platform` ne `mod` import edebilir.

`go run ./cmd/depcheck` bu üç kuralı denetler.

### Yetki modeli

Kullanıcıdaki `role` alanı **yalnızca platform rolüdür** (`user`, `admin`,
`moderator`). İşletme kapsamlı yetki `mod/authz` üzerinden yürür:

- **İzin**: `catalog.manage`, `org.member.manage`, `organization.publish`…
- **Rol**: izin kümesi (`owner`, `manager`, `staff`, `member`, platform rolleri).
- **Atama**: bir rolü, bir kullanıcıya, bir kapsamda verir
  (`platform` · `organization` · `branch` · `self`).

Varsayılan cevap **hayır**'dır. Kapsam kimliği istek gövdesinden değil, sunucuda
yüklenen kayıttan/rota parametresinden alınır.

### Veri sahipliği

| Tablo | Sahip |
|---|---|
| `users`, `refresh_tokens`, `one_time_tokens` | identity |
| `roles`, `permissions`, `role_permissions`, `role_assignments` | authz |
| `files`, `file_objects` | files |
| `notifications`, `notification_preferences` | notifications |
| `organizations`, `branches`, `memberships`, `regions`, `service_areas` | organization |
| `taxonomy_nodes`, `organization_categories`, `offerings`, `offering_variants` | catalog |

Tüm modüller aynı PostgreSQL şemasını (`public`) paylaşır ama tabloları
birbirine karışmaz. Bir motor kendi şemasını ve kendi `schema_migrations`
geçmişini getirir.

## Uygulama üretme

```bash
go run ./cmd/corectl new fitscope --level=L2 --dir=../fitscope
cd ../fitscope
docker compose up
```

```bash
corectl new <app> --level=L0|L1|L2 [--only=identity,files]
corectl add module <name>
corectl add level <LN>
corectl upgrade          # çekirdek sürümünü uygular, major'da durur
corectl doctor           # config, modules.yaml ve migration kontrolü
```

## Uç noktalar (özet)

| Yol | Anlam |
|---|---|
| `GET /healthz` | süreç ayakta (DB'ye bakmaz) |
| `GET /readyz` | DB erişilebilir; değilse 503 |
| `GET /version` | ad, sürüm, commit, derleme zamanı |
| `POST /api/v1/auth/register` · `login` · `refresh` · `logout` | kimlik |
| `GET /api/v1/me` | oturum sahibi |
| `GET /api/v1/organizations` · `/organizations/by-slug/:slug` | herkese açık keşif |
| `POST /api/v1/organizations` | işletme oluştur (sahip = oturum) |
| `GET/POST /api/v1/organizations/:org_id/...` | şube, üye, hizmet alanı |
| `GET /api/v1/taxonomy` · `/offerings` | herkese açık katalog |
| `GET/POST /api/v1/organizations/:org_id/offerings` | işletme kataloğu |
| `GET /api/v1/authz/me/permissions` | etkin izinler |

## Geliştirme

```bash
make check          # gofmt + vet + test (CI ile aynı)
make test
make run
make depcheck
```

Duman testi (çalışan bir örneğe karşı):

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke.ps1 -BaseUrl http://localhost:8080
```

Testler `TEST_DATABASE_DSN` yoksa veritabanı gerektiren senaryoları atlar;
CI'da Postgres servisi tanımlıdır. Windows'ta cgo olmadığı için `-race`
çalışmaz; Linux CI'da çalıştırılır.

## Belgeler

- `PRD.md` — ürün gereksinimleri: kapsam, ilkeler, kabul kriterleri
- `doc/ARCHITECTURE.md` — sınırlar, veri sahipliği, yetki/tenant matrisi
- `doc/ENGINE_CONTRACT.md` — bir sektör motorunun çekirdeğe nasıl bağlanacağı
- `doc/ADR.md` — alınan kararlar ve gerekçeleri
- `references/L0.md` — kernel gereksinimleri
- `references/NASIL-CALISIR.md` — çalıştırma ve sorun giderme

## Lisans

`LICENSE` dosyasına bakın.
