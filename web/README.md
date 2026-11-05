# Web arayüzü

Çekirdeğin yönetim arayüzü. Bilinçli olarak **derlemesiz ve bağımlılıksız**:
üç dosya (`index.html`, `styles.css`, `app.js`) ve tarayıcı. `npm install` yok,
build adımı yok, güncelleme yok.

Ayrı bir origin'de (`http://localhost:3000`) çalışır; API Docker içinde
`http://localhost:18080` üzerinde durur. İki ayrı origin olması bilinçlidir:
arayüz ile API arasındaki sınır gerçek bir HTTP sınırıdır, bu yüzden CORS,
CSRF ve çerez davranışı üretimde de aynı şekilde test edilir.

## Çalıştırma

1. API'yi başlatın (proje kökünde):

   ```powershell
   docker compose up -d
   ```

2. Arayüzü başlatın:

   ```powershell
   ./scripts/web.ps1
   ```

3. Tarayıcıda **http://localhost:3000** adresini açın.

Port veya klasör değiştirmek için:

```powershell
./scripts/web.ps1 -Port 3001
```

Statik sunucu için Python dışında bir şey kullanmak isterseniz herhangi bir
sunucu yeterlidir; arayüzün ihtiyacı yalnızca dosyaları doğru MIME türleriyle
sunmak.

## API adresi

Varsayılan `http://localhost:18080`. Farklıysa giriş ekranındaki
**Bağlantı ayarı** bölümünden değiştirin; seçim yalnızca o tarayıcıda saklanır.

## CORS zorunlu

API, tarayıcı isteklerini bir allow-list ile sınırlar. Arayüzün origin'i
`config/config.yaml` içindeki `security.allowed_origins` listesinde olmalı:

```yaml
security:
  allowed_origins:
    - http://localhost:3000
```

Listede yoksa tarayıcı isteği hiç göndermez ve konsolda CORS hatası görürsünüz.
Üretimde alan adını buraya ekleyin; `*` üretimde reddedilir.

## Oturum nasıl taşınır

- Erişim tokeni yalnızca bellekte tutulur (`localStorage`'a yazılmaz), böylece
  bir XSS hatası uzun ömürlü bir kimlik bilgisi sızdıramaz.
- Yenileme tokeni `HttpOnly` çerezdedir; JavaScript onu okuyamaz.
- Sayfa yenilendiğinde oturum `POST /api/v1/auth/refresh` ile geri gelir —
  tarayıcı çerezi kendisi gönderdiği için bu çalışır.
- Her yazma isteği, okunabilir `csrf` çerezinden kopyalanan `X-CSRF-Token`
  başlığını taşır (double-submit).

## Kapsam

Arayüz çekirdeğin sözleşmelerini yönetir: işletme, şube, ekip, katalog ve
bildirim tercihleri. Randevu, sipariş, ödeme gibi sektör işlemleri burada
**yoktur** ve olmayacaktır: onlar sürümlenmiş sözleşmeler üzerinden bağlanan
motorlara aittir (`doc/ENGINE_CONTRACT.md`).

## Yetki

Görünen düğmeler kullanıcının izinlerine göre çizilir, ama bu yalnızca bir
kolaylıktır. Her isteği sunucu ayrıca denetler; yetkisiz bir çağrı arayüzde
düğme görünse bile `403` alır.
