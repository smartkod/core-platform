# Nasıl Çalışır — Komut Rehberi

## Kısa cevap

> Bu proje bir **Go backend**. `npm` kullanmaz. `package.json` yoktur.

| Komut | Sonuç |
|---|---|
| `npm run dev` | ❌ **çalışmaz** — bu bir Node projesi değil |
| `go run ./cmd/server` | ✅ çalışır (yerel, Go ile) |
| `docker compose up -d --build` | ✅ çalışır (önerilen) |

Servis ayağa kalktıktan sonra bir **HTTP API** olur. Tarayıcıda açılacak bir
sayfa yoktur.

### Tarayıcıda `404` görmek normaldir

`http://localhost:18080/` adresine gidince daha önce `404 NOT_FOUND` dönüyordu.
Bu bir hata değildi: backend'in kökünde sayfa yok. Artık kökte **keşif belgesi**
var — hangi endpoint'lerin olduğunu listeler:

```powershell
(Invoke-WebRequest http://localhost:18080/ -UseBasicParsing).Content
```

```json
{
  "name": "core-platform",
  "message": "This is a JSON API, not a web page. ...",
  "health": ["GET /healthz", "GET /readyz", "GET /version"],
  "api": ["POST /api/v1/auth/login", "GET /api/v1/me", ...],
  "auth": { "flow": "POST /api/v1/auth/register or /login ..." }
}
```

Bu belge yalnızca `APP_ENV != production` iken açıktır: geliştiriciye yol
gösterir, üretimde saldırgana bilgi vermez.

### Neden `npm run dev` aklına geldi

Bu depoda `web/` klasörü var ama içi **boş** (`web/react` boş bir klasör).
Eski sürümden kalmış bir kalıntı, hiçbir şey çalıştırmıyor. Repoda:

```
package.json      → yok
node_modules/     → yok
web/react/        → boş
```

Yani "frontend'i başlat" diye bir şey yok, çünkü **frontend yazılmadı**. Bu
depo yalnızca backend çekirdeğini içerir (PRD'de "Mobil/web istemciler: kapsam
dışı" olarak yazılı).

---

## Doğru çalıştırma — Docker (önerilen)

### En kolay yol: hazır betikler

```powershell
cd C:\Projects\OPENCODE-projeleri\2-core-platform

.\scripts\run.ps1      # başlat (boş portu kendi bulur)
.\scripts\stop.ps1     # durdur (veri korunur)
```

`run.ps1` şunları yapar: boş host portu bulur, konteynerleri başlatır,
`/readyz` 200 dönene kadar bekler ve kullanılabilir URL'leri yazar.
Zaten çalışıyorsa hiçbir şeyi bozmaz.

```powershell
.\scripts\stop.ps1 -Drop    # durdur + veritabanını sil
.\scripts\run.ps1 -Fresh    # başlat + veritabanını sıfırla
```

### Elle

```powershell
$env:APP_PORT="18080"        # 8080 doluysa başka numara seç
$env:DB_PORT_HOST="15532"    # 5432 doluysa başka numara seç

docker compose up -d --build
```

Hazır mı kontrol et:

```powershell
Invoke-WebRequest http://localhost:18080/readyz -UseBasicParsing
```

`200 {"status":"ready"}` dönerse çalışıyor.

Denemek için:

```powershell
# sağlık
Invoke-WebRequest http://localhost:18080/healthz -UseBasicParsing

# kayıt
Invoke-RestMethod -Uri "http://localhost:18080/api/v1/auth/register" -Method Post `
  -ContentType "application/json" `
  -Body '{"email":"ben@example.com","password":"correct horse battery","name":"Ben"}'
```

Veya tüm akışı tek komutla:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke.ps1 -BaseUrl http://localhost:18080
```

### İşletme ve katalog kontrolü

Duman betiği kimlik akışının ardından işletme oluşturma, sahiplik yetkisi,
tenant izolasyonu (403/404), şube ve davet akışını da doğrular:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke.ps1 -BaseUrl http://localhost:18080
```

Durdur:

```powershell
docker compose down        # veriyi korur
docker compose down -v     # veritabanını da siler
```

---

## "Bağlantı kurulamıyor" / servis kayboluyor

`Invoke-WebRequest : Uzak sunucuya bağlanılamıyor` hatası, servisin o anda
**çalışmadığı** anlamına gelir. En sık sebep: `docker compose down` çalıştırılmış
veya Docker Desktop yeniden başlamış. `down` konteynerleri siler; otomatik geri
gelmezler.

Kontrol ve düzeltme:

```powershell
docker compose ps -a        # boşsa hiçbir şey çalışmıyor
.\scripts\run.ps1           # başlat
```

Betik kullanmıyorsan port değişkenlerini her oturumda yeniden vermen gerekir;
`run.ps1` bunu senin yerine yapar.

---

## Docker'sız çalıştırma (yerel Go)

PostgreSQL gerekir. En kolayı sadece veritabanını Docker'da çalıştırmak:

```powershell
# 1. Sadece Postgres
docker compose up -d postgres

# 2. Secret'ları ayarla (üretimde zorunlu, geliştirmede de önerilir)
$env:JWT_ACCESS_SECRET="0123456789abcdef0123456789abcdef"
$env:JWT_REFRESH_SECRET="fedcba9876543210fedcba9876543210"
$env:DB_HOST="localhost"
$env:DB_PORT="15532"
$env:DB_NAME="core_platform"
$env:SERVER_PORT="18080"

# 3. Çalıştır
go run ./cmd/server
```

Ortam değişkeni vermek istemezsen `config/config.yaml` kullanılır (env onu
ezer). Ayar önceliği: **env > config.yaml > varsayılan**.

> **`.env` dosyası artık kullanılmıyor.** Eski bozuk sürümden kalan dosya
> `.env.ESKI` olarak arşivlendi. `go run` `.env` dosyasını okumaz; değişkenler
> ya ortamdan ya `config/config.yaml`'dan gelir.

---

## "App konteyneri ölüyor" sorunu

Belirti:

```
docker compose ps -a
2-core-platform-app-1   Created / Exited
```

Log'da:

```
failed to connect to `host=postgres ...`: hostname resolving error
(lookup postgres on 192.168.65.7:53: dial udp ...: network is unreachable)
```

Bu bir **kod hatası değil**, Docker Desktop'ın iç DNS'i. Ağ oluşturulurken
DNS geçici olarak cevap vermiyor ve uygulama açılışta ölüyor.

**Çözüm (uygulandı):** `docker-compose.yml`'a `restart: unless-stopped`
eklendi. Artık geçici DNS hatasında konteyner kendini yeniden başlatır ve
bağlantı kurulunca ayağa kalkar.

Hâlâ sorun varsa sırayla:

```powershell
# 1. Ağı ve konteynerleri tamamen yeniden oluştur
docker compose down -v
docker compose up -d --build

# 2. Docker Desktop'ı yeniden başlat
#    (Settings > Troubleshoot > Restart)

# 3. Son çare: Docker Desktop'ı fabrika ayarlarına döndürmeden DNS'i kontrol et
docker run --rm alpine nslookup postgres
```

---

## Ne çalışıyor, ne çalışmıyor

| | Durum |
|---|---|
| HTTP API (Go) | ✅ |
| PostgreSQL şeması + migration | ✅ |
| Kayıt / giriş / token yenileme / çıkış | ✅ |
| Yetkilendirme (rol + izin + kapsam) | ✅ |
| Dosya yükleme | ✅ |
| Bildirim (outbox + e-posta) | ✅ |
| İşletme, şube, üyelik | ✅ |
| Taksonomi ve katalog | ✅ |
| Sektör motorları (randevu, sipariş, ödeme) | ⬜ ayrı depo — `doc/ENGINE_CONTRACT.md` |
| **Web arayüzü** | ⬜ **yok** — bu backend |

Web arayüzü istersen ayrı bir depoda yazılır; bu çekirdeğe `fetch` ile bağlanır.
