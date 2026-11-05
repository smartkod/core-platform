# Ürün Sınırları (Product Boundaries)

Bu belge, platformun **neyi çekirdekte tuttuğunu, neyi sektöre bıraktığını** ve
bunun neden değişmez bir kural olduğunu tanımlar.

Teknik bağlanma sözleşmesi ayrı bir belgededir: `doc/ENGINE_CONTRACT.md`.
Buradaki metin, o sözleşmenin **neden** böyle olduğunu açıklar.

---

## 1. Tek cümlelik kural

Çekirdek, **işletmenin kim olduğunu** ve **müşterinin hizmete nasıl
ulaşacağını** standartlaştırır. Sektöre özgü alanlar eklenti/paket olarak
tasarlanır; **çekirdeğe sektör mantığı gömülmez.**

Bir özellik şu soruya "evet" diyorsa çekirdeğe aittir:

> Sektörü ne olursa olsun her işletme bu kavrama sahip mi?

"Sadece kuaförlerde olur", "sadece marketlerde olur" diyen her şey motora
aittir.

---

## 2. Kullanıcı yolculuğu

```
Keşfet/ara → Karşılaştır → Talep/rezervasyon → Mesaj/takip → Yorum/tekrar
```

Bu beş adım sektörden bağımsızdır. Yalnızca dördüncü adımın **içi** değişir:
bir randevu, bir teklif, bir sipariş veya bir üyelik olabilir. Bu yüzden
çekirdek yolculuğu bilir, adımın içeriğini bilmez.

---

## 3. Çekirdek: bir kez tanımla

| Varlık | Neden çekirdekte |
|---|---|
| Hesap / rol | Her işletmede aynı soru: kim, hangi yetkiyle? |
| İşletme / şube | Hizmetin sahibi ve fiziksel/dijital konumu |
| Uzman / ekip | Hizmeti veren kişi, sektör ne olursa olsun var |
| Hizmet / varyant | "Ne sunuyorsun" sorusunun ortak cevabı |
| Bölge / lokasyon | Hizmetin nerede verildiği |
| İşlem / durum | Ortak yaşam döngüsü **kimliği** (içeriği değil) |
| Mesaj / bildirim | İletişim kanalı sektörden bağımsız |
| Yorum / itiraz | Güven mekanizması her pazaryerinde gerekli |

Bugünkü kod karşılıkları: `mod/identity`, `mod/authz`, `mod/organization`,
`mod/catalog`, `mod/notifications`, `mod/files`.

---

## 4. Sektör konfigürasyonu: gerektiğinde ekle

| Alan | Hangi motor |
|---|---|
| Randevu süresi / slot | Randevu ve kapasite |
| Teklif kapsam / kalem | Talep, teklif ve proje |
| Ürün SKU / stok | Katalog, stok ve sipariş |
| Ders grup / devam | Randevu veya üyelik |
| Etkinlik görev / bütçe | Talep, teklif ve proje |
| Sağlık yetki / izin | Üyelik ve içerik |
| Proje kilometre taşı | Talep, teklif ve proje |
| Üyelik dönem / kota | Üyelik, abonelik ve içerik |

Bu alanlar çekirdek tablolarına **kolon olarak eklenmez**. Her motor kendi
şemasını getirir (`doc/ENGINE_CONTRACT.md` §4).

---

## 5. Yönetişim: merkezî kontrol

| Konu | Kural | Kod |
|---|---|---|
| Taksonomi | Tek ana kayıt; motor kendi kategori ağacını kurmaz | `mod/catalog` |
| Yetki | Rol + scope; izin her istekte sunucuda | `mod/authz` |
| Sözleşme | Sürümlü; motor çekirdeğin iç tipini import etmez | `contract/core` |
| Log | Denetlenebilir; parola/token/CSRF yazılmaz | `platform/logger` |
| Veri izolasyonu | Kategori/kiracı bazlı; testle kanıtlı | `test/security` |
| Moderasyon | Politika bazlı, koda gömülü değil | `mod/authz` izinleri |
| Ölçüm | Ortak event şeması | henüz yok — motorlar bunu tüketecek |
| Uyum | Sektör bazlı; çekirdek yalnızca kanca sağlar | motor sorumluluğu |

---

## 6. Dört işlem modeli

Sektörlerin çoğu dört motordan biriyle tarif edilir. Bu bir sınıflandırma
değil, **sınır çizme** aracıdır.

### Motor 01 — Randevu ve kapasite

`Hizmet seç → Slot bul → Onayla → Hatırlat → Tamamla`

Uygunluk takvimi, çalışma saatleri, süre, tampon, kapasite; iptal, geç kalma,
no-show, tekrar randevu; personel/oda/ekipman planlaması.
**Güzellik · fitness · eğitim · bakım**

### Motor 02 — Talep, teklif ve proje

`Brief → Teklif → Kapsam → Aşamalar → Teslim`

İş tanımı, dosya, bütçe aralığı, lokasyon, zaman beklentisi; teklif sürümleri,
onay, değişiklik talebi; proje durumu, teslimat, kabul, anlaşmazlık.
**Web tasarım · tadilat · etkinlik · danışmanlık**

### Motor 03 — Katalog, stok ve sipariş

`Ürün → Stok → Sepet → Ödeme → Teslim`

SKU, varyant, fiyat, stok, kampanya, vergi, iade; şube stoğu, rezervasyonlu
stok, kargo; satıcı/platform sorumluluk ayrımı.
**Market · perakende · ekipman · dijital ürün**

### Motor 04 — Üyelik, abonelik ve içerik

`Plan → Yetki → Kullanım → Yenile`

Dönem, kota, kredi/seans, yenileme, dondurma; içerik erişimi, dijital rapor,
canlı seans. **İşletme aboneliği ile son kullanıcı aboneliği ayrı tanımlanır.**
**Fitness · eğitim · astroloji · B2B bakım**

---

## 7. Değişmez mimari kural

> Ortak bir `transaction` kimliği ve yaşam döngüsü olayları kullanılabilir.
> Ancak randevu, teklif, sipariş ve üyelik tabloları/kuralları **tek bir devasa
> işlem şemasına** dönüştürülmemelidir.

Her motor kendi alan modeline ve durum makinesine sahiptir; çekirdeğe **olay ve
raporlama** üzerinden bağlanır.

Bu kuralın bedeli: çekirdek "her işlemi tek tabloda görmek" gibi bir kolaylığı
bilinçli olarak reddeder. Karşılığında bir motorun şema değişikliği diğerlerini
kıramaz, bir sektörün kuralı başka sektöre sızmaz ve bir motorun hatası
çekirdeği durdurmaz.

### Neden tek şema cazip ama yanlış

Tek `transactions` tablosu ilk ay işe yarar: dört motor da "bir işlem"dir.
Ancak sütunlar hızla sektöre özel hale gelir — `slot_id`, `sku`, `milestone`,
`quota` — ve her yeni sektör tabloyu biraz daha genişletir. Altı ay sonra tablo
hem boş sütunlarla dolar hem de her motorun durum makinesi aynı `status`
alanında çakışır. Kaçınılmaz sonuç: kimsenin tam olarak anlamadığı, hiçbir
motorun sahiplenmediği bir çekirdek. Bu belge o yolu kapatır.

---

## 8. Bu belgeye nasıl başvurulur

- Yeni bir özellik isteği geldiğinde §1'deki soru sorulur.
- Cevap "hayır" ise özellik çekirdeğe değil, bir motora veya motora hazırlık
  olan bir sözleşme uzantısına gider.
- Cevap "evet" ise `doc/FEATURES.md` güncellenir ve `doc/ENGINE_CONTRACT.md`
  sözleşmesine dokunulmadan kalır.
- Bir motor çekirdekten bir şey istiyorsa, bu istek **sürümlenmiş sözleşmeye**
  eklenir; çekirdek o motorun adını öğrenmez.
