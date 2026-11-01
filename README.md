# Core Platform

Modüler, çok uygulamalı bir backend çekirdeği. Sekiz uygulama (fitscope,
eduscope, astroscope, financescope, webscope, CodeScope, Mimariscope,
maketscope) aynı altyapıyı buradan alır.

## Hızlı başlangıç

**Bu bir Go backend'dir.** `npm run dev` yoktur — `package.json` da yoktur.
Tarayıcıda açılacak bir sayfa yok; servis bir HTTP API sunar.

```powershell
.\scripts\run.ps1
```

Betik boş portu bulur, konteynerleri başlatır, hazır olana kadar bekler ve
URL'leri yazar. Durdurmak için `.\scripts\stop.ps1`.

Elle yapmak isterseniz:

```bash
docker compose up -d --build
```

```powershell
Invoke-WebRequest http://localhost:8080/healthz -UseBasicParsing
```

→ Ayrıntılı anlatım: **`references/NASIL-CALISIR.md`**
→ Adım adım deneme: **`references/DENEME.md`**

## Durum

| Alan | Durum |
|---|---|
| **L0 — Çekirdek kernel** | ✅ çalışıyor |
| **L1 — identity · files · notifications** | ✅ çalışıyor |
| L1 — search | yardımcı paket, çekirdekte değil |
| L2 — ekonomi | tasarım hazır, kod yok |
| L3 — AI | tasarım hazır, kod yok |

`gofmt`, `go build ./...`, `go vet ./...` ve `go test ./...` yeşil.
Duman testi: **42/42 kontrol**.

## Nasıl çalışır

Çekirdek bir **kütüphanedir**, kopyalanan bir şablon değil. Her uygulama kendi
git deposunda, kendi binary'si ve kendi veritabanıyla çalışır; çekirdeği
sürümlenmiş bir Go modülü olarak tüketir.

```
core-platform/        ayrı depo, ayrı go.mod
├── contract/core/    uygulamanın görebildiği tek arayüz
├── platform/         L0 kernel
├── cmd/corectl/      iskelet üretici
└── references/       PRD.md, L0.md, ...

<app>/                ayrı depo, ayrı go.mod
├── cmd/server/       ~25 satır
├── internal/modules/ uygulamaya özel
└── migrations/
```

Sınırlar derleyiciyle korunur: uygulama yalnızca `contract/core` import eder;
`platform` ve `mod` iç paketleri uygulamaya kapalıdır.

## Yeni uygulama

```bash
cd core-platform
go run ./cmd/corectl new fitscope --level=L0 --dir=../fitscope
cd ../fitscope
docker compose up
```

`corectl new` bağımlılıkları çözer, `vendor/` üretir ve Docker imajını ağ
erişimi gerektirmeyecek şekilde hazırlar. Üretilen proje elle düzenleme
olmadan derlenir ve çalışır.

Diğer komutlar:

```bash
corectl new <app> --level=L0|L1|L2|L3 [--only=identity,files]
corectl add module <name>
corectl add level <LN>
corectl upgrade          # çekirdek sürümünü uygular, major'da durur
corectl doctor           # config, modules.yaml ve migration kontrolü
```

## Çekirdeği çalıştırma

```bash
cp .env.example .env      # gerekirse
docker compose up         # app + postgres
```

| Endpoint | Anlam |
|---|---|
| `GET /healthz` | süreç ayakta (DB'ye bakmaz) |
| `GET /readyz` | DB erişilebilir; değilse 503 |
| `GET /version` | ad, sürüm, commit, derleme zamanı |

## Geliştirme

```bash
make check          # gofmt + vet + test (CI ile aynı)
make test
make run
```

Duman testi (çalışan bir örneğe karşı, 28 kontrol):

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke.ps1 -BaseUrl http://localhost:18080
```

Adım adım deneme kılavuzu: **`references/DENEME.md`**.

Testler `-race` olmadan çalışır çünkü Windows'ta C gcc yok; Linux CI'da
`go test -race ./...` çalıştırılmalıdır.

## Belgeler

- `references/PRD.md` — şemsiye: problem, seviyeler, sınırlar, semver, yol haritası
- `references/L0.md` — çekirdek kernel gereksinimleri ve kabul kriterleri
- `references/L1.md` — kimlik + dosya + bildirim (`identity` uygulandı)
- `references/L2.md` · `references/L3.md` — ekonomi ve AI tasarımı
- `references/NASIL-CALISIR.md` — çalıştırma komutları ve sık sorunlar
- `references/DENEME.md` — kendi ortamında deneme kılavuzu
- `doc/GELISTIRME-PLANI.md` — depo kurtarma öncesi durum tespiti (tarihsel)

Seviye belgeleri (`L1.md`, `L2.md`, `L3.md`) koduyla birlikte yazılır.

## Lisans

`LICENSE` dosyasına bakın.
