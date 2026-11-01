# core-platform — Yerel Deneme Kılavuzu

Bu belge, çekirdeği kendi makinenizde sıfırdan ayağa kaldırıp L1 akışını
uçtan uca doğrulamak için gereken adımları içerir.

Süre: ~5 dakika (Docker imajları önbellekteyse).

---

## 0. Ön koşullar

| Araç | Kontrol | Beklenen |
|---|---|---|
| Go | `go version` | 1.22+ |
| Docker | `docker version` | çalışıyor |
| Docker Compose | `docker compose version` | v2+ |

Kontrol:

```powershell
go version
docker version --format '{{.Server.Version}}'
docker compose version
```

---

## 1. Çekirdek deposunda temel doğrulama

```powershell
cd C:\Projects\OPENCODE-projeleri\2-core-platform

go build ./...
go vet ./...
go test ./...
```

Hepsi hatasız dönmeli. `gofmt` kontrolü:

```powershell
gofmt -l cmd internal platform contract mod
```

Çıktı **boş** olmalı.

> **Not:** `go test -race` Windows'ta çalışmaz (gcc yok, `-race` cgo ister).
> CI'da Linux üzerinde çalıştırılmalıdır.

---

## 2. Çekirdeği Docker'da çalıştır

`APP_PORT` ve `DB_PORT_HOST` başka projelerle çakışmasın diye değiştirilebilir.
Aşağıdaki örnek 18080 / 15532 kullanıyor.

```powershell
cd C:\Projects\OPENCODE-projeleri\2-core-platform

$env:APP_PORT="18080"
$env:DB_PORT_HOST="15532"
$env:JWT_ACCESS_SECRET="0123456789abcdef0123456789abcdef"
$env:JWT_REFRESH_SECRET="fedcba9876543210fedcba9876543210"

docker compose up -d --build
```

Hazır olana kadar bekle:

```powershell
# 200 dönene kadar dener
for ($i=0; $i -lt 60; $i++) {
  try {
    $r = Invoke-WebRequest http://localhost:18080/readyz -UseBasicParsing -TimeoutSec 2
    if ($r.StatusCode -eq 200) { "READY"; break }
  } catch { Start-Sleep -Milliseconds 500 }
}
```

Beklenen: `READY`. Tipik süre 15–30 sn (ilk build'de daha uzun).

---

## 3. Sağlık uçları

| İstek | Beklenen |
|---|---|
| `GET /healthz` | `200 {"status":"ok"}` |
| `GET /readyz` | `200 {"status":"ready"}` |
| `GET /version` | 200, uygulama adı + sürüm |

```powershell
(Invoke-WebRequest http://localhost:18080/healthz -UseBasicParsing).Content
(Invoke-WebRequest http://localhost:18080/version -UseBasicParsing).Content
```

**Liveness/readiness ayrımı (önemli kontrol):**

```powershell
docker compose stop postgres

# healthz hâlâ 200 dönmeli (süreç ayakta)
(Invoke-WebRequest http://localhost:18080/healthz -UseBasicParsing).StatusCode   # 200

# readyz 503 dönmeli (bağımlılık yok)
try { Invoke-WebRequest http://localhost:18080/readyz -UseBasicParsing } catch {
  $_.Exception.Response.StatusCode.value__                                      # 503
}

docker compose start postgres
```

---

## 4. L1 kimlik akışı — uçtan uca

**En kolay yol:** hazır duman testi betiği.

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke.ps1 -BaseUrl http://localhost:18080
```

Beklenen çıktı: `all 27 checks passed`, çıkış kodu `0`. Her satır tek bir
davranışı doğrular ve `PASS`/`FAIL` yazar; bir şey bozulursa hangi davranışın
bozulduğunu doğrudan gösterir.

Aşağıdaki bölümler aynı kontrolleri elle yapmak isteyenler içindir.

### 4.1 Kayıt

```powershell
$b = "http://localhost:18080/api/v1"
$reg = Invoke-RestMethod -Uri "$b/auth/register" -Method Post -ContentType "application/json" `
  -Body '{"email":"test@example.com","password":"correct horse battery","name":"Test"}'
$reg.data
```

Beklenen: `id` bir UUIDv7 (ör. `01a1965d-a14b-79f2-...`), `email_verified: false`.

### 4.2 Giriş

```powershell
$login = Invoke-RestMethod -Uri "$b/auth/login" -Method Post -ContentType "application/json" `
  -Body '{"email":"test@example.com","password":"correct horse battery"}'
$access  = $login.data.access_token
$refresh = $login.data.refresh_token
```

### 4.3 Korumalı `/me`

```powershell
Invoke-RestMethod -Uri "$b/me" -Headers @{Authorization="Bearer $access"} | ConvertTo-Json
```

Beklenen: kullanıcı bilgisi. **Token olmadan** çağırınca `401 UNAUTHORIZED`.

### 4.4 Refresh rotation

```powershell
$r2 = Invoke-RestMethod -Uri "$b/auth/refresh" -Method Post -ContentType "application/json" `
  -Body "{`"refresh_token`":`"$refresh`"}"
$newRefresh = $r2.data.refresh_token
$newRefresh -ne $refresh     # True olmalı: token döndü
```

### 4.5 Reuse tespiti (güvenliğin kalbi)

Eski token'ı tekrar gönder:

```powershell
try {
  Invoke-RestMethod -Uri "$b/auth/refresh" -Method Post -ContentType "application/json" `
    -Body "{`"refresh_token`":`"$refresh`"}"
} catch { $_.Exception.Response.StatusCode.value__ }   # 401
```

Şimdi **yeni** token da ölmüş olmalı (tüm aile iptal edildi):

```powershell
try {
  Invoke-RestMethod -Uri "$b/auth/refresh" -Method Post -ContentType "application/json" `
    -Body "{`"refresh_token`":`"$newRefresh`"}"
} catch { $_.Exception.Response.StatusCode.value__ }   # 401 — kritik!
```

Bu ikinci `401` testin en önemli kısmı: sadece eski token değil, **ailesi**
iptal ediliyor.

### 4.6 Yetki kontrolü

```powershell
$l = Invoke-RestMethod -Uri "$b/auth/login" -Method Post -ContentType "application/json" `
  -Body '{"email":"test@example.com","password":"correct horse battery"}'

try {
  Invoke-RestMethod -Uri "$b/admin/users" -Headers @{Authorization="Bearer $($l.data.access_token)"}
} catch { $_.Exception.Response.StatusCode.value__ }   # 403 FORBIDDEN
```

### 4.7 Çıkış ve geçersizleşme

```powershell
Invoke-RestMethod -Uri "$b/auth/logout" -Method Post -ContentType "application/json" `
  -Body "{`"refresh_token`":`"$($l.data.refresh_token)`"}"
```

Çıkış sonrası aynı refresh token → `401`.

`logout-all` ise **eski access token'ı** da geçersiz kılar (`token_version`):

```powershell
$l2 = Invoke-RestMethod -Uri "$b/auth/login" -Method Post -ContentType "application/json" `
  -Body '{"email":"test@example.com","password":"correct horse battery"}'

Invoke-RestMethod -Uri "$b/me/logout-all" -Method Post `
  -Headers @{Authorization="Bearer $($l2.data.access_token)"}

try {
  Invoke-RestMethod -Uri "$b/me" -Headers @{Authorization="Bearer $($l2.data.access_token)"}
} catch { $_.Exception.Response.StatusCode.value__ }   # 401
```

### 4.8 Girdi doğrulama

```powershell
# zayıf şifre
try {
  Invoke-RestMethod -Uri "$b/auth/register" -Method Post -ContentType "application/json" `
    -Body '{"email":"weak@example.com","password":"123"}'
} catch { $_.Exception.Response.StatusCode.value__ }   # 422

# aynı e-posta: hesap varlığı sızmamalı
$dup = Invoke-RestMethod -Uri "$b/auth/register" -Method Post -ContentType "application/json" `
  -Body '{"email":"test@example.com","password":"correct horse battery"}'
$dup.data.message   # "if the address is available, ..." — Conflict değil
```

### 4.9 Hata zarfı biçimi

Her hata aynı biçimde dönmeli:

```json
{
  "success": false,
  "error": { "code": "UNAUTHORIZED", "message": "unauthorized" },
  "request_id": "a58e9366ed248c5d8e457b46"
}
```

`request_id` her yanıtta `X-Request-ID` başlığında da bulunur.

### 4.10 Brute force koruması

Art arda hatalı giriş dene; bir noktadan sonra `429` görmelisin:

```powershell
$e = "brute@example.com"
Invoke-RestMethod -Uri "$b/auth/register" -Method Post -ContentType "application/json" `
  -Body (@{email=$e;password="correct horse battery"}|ConvertTo-Json) | Out-Null

for ($i=1; $i -le 13; $i++) {
  try {
    Invoke-RestMethod -Uri "$b/auth/login" -Method Post -ContentType "application/json" `
      -Body (@{email=$e;password="yanlis"}|ConvertTo-Json) -ErrorAction Stop | Out-Null
    "$i : 200"
  } catch { "$i : $([int]$_.Exception.Response.StatusCode)" }
}
```

Beklenen: birkaç `401` sonra `429`.

**Önemli tasarım noktası:** yalnızca **başarısız** denemeler sayılır. Başarılı
giriş kotayı tüketmez — aksi halde aynı ofis/NAT arkasındaki tüm kullanıcılar
birbirinin kotasını yerdi. Bunu doğrulamak için art arda 20 kez **doğru**
şifreyle gir; hiç `429` görmemelisin.

---

## 5. Üretilen proje (golden path)

Bu, asıl kazanç: yeni uygulamaya tek komutla başlamak.

```powershell
cd C:\Projects\OPENCODE-projeleri\2-core-platform
go run ./cmd/corectl new demo-l1 --level=L1 --only=identity --dir=demo-l1
```

Çıktı:

```
created ...\demo-l1 (level L1)
modules: identity
```

Üretilen proje kendi başına derlenir:

```powershell
cd demo-l1
go build ./...
go test ./...
```

Docker'da çalıştır (farklı port):

```powershell
$env:APP_PORT="18081"
$env:DB_PORT_HOST="15533"
docker compose up -d --build
```

`http://localhost:18081/readyz` → `200`, ve aynı kimlik akışı çalışır.

> `--level=L1` (yalnız) şu an hata verir, çünkü `files` ve `notifications`
> henüz yazılmadı. `--only=identity` ile sadece kimlik üretilir. Bu bilinçli:
> yarım iskelet üretip derlenmeyen proje vermek yerine durur.

---

## 6. Temizlik

```powershell
cd C:\Projects\OPENCODE-projeleri\2-core-platform\demo-l1
docker compose down -v
cd ..
Remove-Item -Recurse -Force demo-l1

docker compose down -v
```

---

## 7. Sorun giderme

| Belirti | Sebep | Çözüm |
|---|---|---|
| `port is already allocated` | Port başka projede | `$env:APP_PORT` / `$env:DB_PORT_HOST` değiştir |
| `identity: an access token secret is required` | Compose env geçmiyor | `$env:JWT_ACCESS_SECRET` ayarla |
| `/readyz` 503 | Postgres hazır değil | 10 sn bekle, `docker compose logs postgres` |
| Build'de `go.mod requires go >= X` | Docker imajı eski | `corectl` go sürümünü kendi toolchain'inden alır; `go version` 1.22+ olsun |
| `level L1 is not available` | files/notifications yok | `--only=identity` kullan (beklenen davranış) |

Log'lar:

```powershell
docker compose logs app --tail 50
docker compose logs postgres --tail 20
```

---

## 8. Bilinmesi gerekenler

- **Testler gerçek PostgreSQL ister.** `go test ./...` DB gerektiren testleri
  `TEST_DATABASE_DSN` yoksa **atlar** (skip), sessizce geçmez.
  DB'li çalıştırmak için:

  ```powershell
  docker run -d --name test-pg -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=identity_test -p 15599:5432 postgres:15-alpine
  Start-Sleep 6
  $env:TEST_DATABASE_DSN="host=localhost port=15599 user=postgres password=postgres dbname=identity_test sslmode=disable"
  go test ./mod/identity/... -v
  docker rm -f test-pg
  ```

- **`-race` Windows'ta yok.** Linux CI'da zorunlu.
- **Sırlar repoda değil.** `.env.example` sadece isimleri içerir.
- **Üretimde fail-fast.** `JWT_ACCESS_SECRET` boşsa veya 32 karakterden
  kısaysa süreç **hiç başlamaz**.
