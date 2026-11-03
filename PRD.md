# PRD — Hizmet Katmanı Çekirdeği V1

Sürüm: 1.0 · Durum: uygulanıyor · Kapsam: bu depo (`core-platform`)

---

## 1. Problem

Aynı altyapıyı tekrar tekrar yazıyoruz. Her yeni uygulama (randevu, kurs, atölye,
danışmanlık, yerel hizmet, etkinlik) aynı sorularla başlıyor:

- Kullanıcı nasıl kayıt olur, oturumu nasıl yönetilir?
- Bir işletme platformda nasıl temsil edilir, şubeleri nasıl modellenir?
- "Bu kişi bu işletmede bunu yapabilir mi?" sorusunun tek bir cevabı var mı?
- İşletme neyi sunuyor? Kategoriler nereden geliyor?

Bu soruların cevabı sektöre göre değişmiyor. Değişen şey, işletmenin **nasıl
işlem yaptığı**: randevu mu alıyor, sipariş mi topluyor, teklif mi veriyor,
ders mi planlıyor.

Bu PRD, bu ortak katmanı **tek bir çekirdek** olarak tanımlar. Çekirdek,
işletmenin kim olduğunu ve neyi sunduğunu bilir; işlemi çalıştırmaz.

### 1.1 Önceki durumdan farkı

Depoda daha önce L3 (AI) ve L2 (ekonomi: cüzdan, sipariş, ödeme, abonelik)
modülleri vardı. Bunlar kaldırıldı. Gerekçe:

- Cüzdan/ödeme bir **sektör motorudur**, çekirdek değil. Onu çekirdeğe koymak,
  ödeme yapmayan bir uygulamayı bile o şemaya ve o kavramlara bağlar.
- AI bir **adaptördür**. Model sağlayıcısı seçimi çekirdeğin değil uygulamanın
  kararıdır.
- İkisi de "her uygulama bunu ister" varsayımına dayanıyordu; oysa ortak olan
  şey işlem değil, **kimlik + yetki + işletme + katalog**tur.

## 2. Hedefler

1. Sektörden bağımsız, yeniden kullanılabilir bir Go backend çekirdeği.
2. Kimlik, yetkilendirme, işletme ve katalog modüllerini eksiksiz çalıştırmak.
3. Gelecekteki işlem motorlarının **sürümlenmiş sözleşmeler** üzerinden
   bağlanabileceği sınırları şimdiden kurmak.
4. Uygulama başına ayrı depo, ayrı binary, ayrı veritabanı; çekirdek
   sürümlenmiş bir Go modülü olarak tüketilir.
5. Sınırları kod incelemesine değil **derleyiciye ve CI'a** emanet etmek.

## 3. Hedef olmayanlar (bilinçli olarak kapsam dışı)

Aşağıdakiler bu depoda **yazılmayacaktır**:

- Randevu, rezervasyon, takvim, kapasite, sıra/bekleme listesi
- Sipariş, sepet, stok, SKU, kargo, lojistik
- Teklif, sözleşme, onay akışı
- Ders, sınav, müfredat, yoklama
- Proje, görev, zaman çizelgesi
- Ödeme, cüzdan, abonelik, fatura, iade
- AI çağrısı, prompt yönetimi, model yönlendirme
- Arama motoru (yardımcı paket olarak kalabilir, çekirdek modülü değildir)
- Bildirim şablonlarının sektörel içeriği
- Raporlama/BI, çoklu dil çeviri yönetimi

Bu liste bir eksiklik değil, **ürün kararıdır**. Bir motor bunları çekirdeğin
sözleşmeleri üzerinden kendi deposunda uygular (`doc/ENGINE_CONTRACT.md`).

## 4. Aktörler

| Aktör | Tanım |
|---|---|
| **Ziyaretçi** | Oturumu olmayan; keşif uçlarını kullanır |
| **Kullanıcı** | Hesabı olan kişi; platform rolü `user` |
| **İşletme sahibi** | Bir organizasyonda `owner` rolü; devir yapabilir |
| **İşletme yöneticisi** | `manager`; şube, üye, katalog yönetir |
| **İşletme personeli** | `staff`; sınırlı katalog yetkisi |
| **Üye** | `member`; işletmeye bağlı, yönetim yetkisi yok |
| **Moderatör** | Platform rolü `moderator`; içerik ve işletme incelemesi |
| **Platform yöneticisi** | Platform rolü `admin`; tüm platform |
| **Motor (extension)** | Çekirdeğe sözleşme üzerinden bağlanan sektör modülü |

## 5. Kavramsal model

```
User ──< Membership >── Organization ──< Branch
  │                         │
  │                         ├──< ServiceArea >── Region
  │                         └──< OrganizationCategory >── TaxonomyNode
  │                                                          │
  └──< RoleAssignment (role, scope, scope_id)                │
                                                             │
                                       Offering ─────────────┘
                                          │
                                          └──< OfferingVariant
```

- **User**: platformdaki kişi. Uygulamalar arası ortak kimlik yoktur; her
  uygulamanın kendi kullanıcı tablosu vardır.
- **Organization**: işletme/kurum. Platformun **tenant** sınırıdır.
- **Branch**: fiziksel şube, hizmet alanı veya online kanal.
- **Membership**: kişi ile organizasyon arasındaki bağ; davet edilmiş ama henüz
  kayıt olmamış kişiyi de temsil edebilir.
- **RoleAssignment**: `(user, role, scope, scope_id)`. Organizasyon kapsamlı
  yetkinin tek kaynağı.
- **TaxonomyNode**: merkezi kategori ağacı. Serbest metin kategori yerine
  geçemez.
- **Offering**: "bu işletme neyi sunuyor" kaydı. Nasıl satıldığı burada değildir.

## 6. İşlevsel gereksinimler

### 6.1 Kimlik (identity)

- E-posta + parola ile kayıt; parola minimum uzunluğu yapılandırılabilir.
- E-posta doğrulama ve parola sıfırlama tek kullanımlık, hash'lenmiş token ile.
- Erişim token'ı (JWT) + yenileme token'ı; yenileme token'ı yalnızca hash olarak
  saklanır, aile (`family_id`) bazında tekrar kullanım tespiti yapılır.
- Web istemcisi için HttpOnly çerez + CSRF double-submit; Bearer her zaman
  çerezden önce gelir.
- `PATCH /me` ile ad, dil, avatar ve telefon güncellenebilir.
- Hesap durumları: `active`, `suspended`, `closed`, `deleted`.
- Platform rolleri yalnızca `user`, `admin`, `moderator`.
- **Kabul**: parola değişince eski erişim token'ları `token_version` sayesinde
  geçersiz olur; "her yerden çıkış" tüm oturumları kapatır.

### 6.2 Yetkilendirme (authz)

- İzin anahtarları modül ad alanına göre adlandırılır (`catalog.manage`).
- Rol, izin kümesidir; hiyerarşi değildir.
- Atama kapsamı: `platform`, `organization`, `branch`, `self`.
- **Varsayılan cevap hayırdır.** Eşleşen aktif atama yoksa istek reddedilir.
- Platform kapsamlı atama her kapsamda geçerlidir (platform yöneticisi her
  işletmede yetkilidir). Organizasyon kapsamlı atama platform kapsamına yükselmez.
- `users.role = admin` olan hesap, atama tablosunda kaydı olmasa bile platform
  işlemlerini yapabilir: ilk organizasyonu yayınlayacak birinin var olması
  gerekir. `moderator` yalnızca inceleme ve okuma izinlerine sahiptir. Diğer tüm
  kararlar atama tablosundan gelir.
- Kapsam kimliği istemciden gövdeyle alınmaz; rota parametresinden (`:org_id`,
  `:branch_id`) veya oturum sahibinden türetilir.
- Atama iptal edilince satır silinmez; `revoked_at` yazılır.
- **Kabul**: aynı kullanıcı A işletmesinde `owner`, B işletmesinde `staff`
  olabilir; A'daki yetkisi B'de hiçbir şey açmaz.

### 6.3 İşletme (organization)

- Organizasyon oluşturan kişi otomatik olarak `owner` olur (üyelik + atama).
- Durum makinesi:
  `draft → pending_verification → active → suspended → closed`
  - `active` yalnızca platform yetkisiyle (`organization.publish`) açılır.
  - `closed` terminaldir.
- Her durum değişikliği `organization_status_events` tablosuna yazılır.
- Şube türleri: `physical`, `service_area`, `online`; şube adresi public
  işaretli değilse keşifte gösterilmez.
- Üyelik durumları: `invited`, `active`, `suspended`, `removed`.
- Davet e-posta adresine yapılır; yalnızca o adrese sahip hesap kabul edebilir.
- Sahiplik devri açık bir işlemdir; son sahibin üyeliği kaldırılamaz.
- Bölgeler (`country`/`city`/`district`) genel bir coğrafi referanstır; mesafe,
  teslimat bölgesi veya posta kodu yönlendirmesi içermez.
- **Kabul**: bir organizasyonun `:org_id` yolunda işlem yapan kullanıcı, o
  organizasyonda geçerli bir atamaya sahip değilse 403 alır.

### 6.4 Katalog (catalog)

- Merkezi taksonomi tek otoritedir; kategori serbest metinle temsil edilemez.
- Organizasyon birden çok kategoriye bağlanabilir (`organization_categories`).
- Sunum kaydı yalnızca ortak alanları taşır: ad, özet, açıklama, medya, hizmet
  modu, görünürlük, etiketler, bilgilendirici `price_from`.
- Fiyat hesaplama, vergi, indirim, stok, süre, kapasite **yoktur**.
- Durum makinesi: `draft → published → archived`; `archived → draft` yeniden
  açma.
- Yayınlanmış sunum silinemez; önce arşivlenir.
- `extension_ref` bir motorun kendi kaydına işaret eder; çekirdek bu kaydı
  **okumaz**.
- Kategori filtrelemesi alt kategorileri kapsar.
- **Kabul**: `visibility=private` bir sunum herkese açık uçta 404 döner.

### 6.5 Dosya ve bildirim

- Dosya yükleme sahiplik kontrollü; imzalı indirme bağlantısı süreli.
- Bildirimler önce veritabanına yazılır (outbox), sonra gönderilir; başarısız
  gönderim yeniden denenir. E-posta V1'de tek kanaldır.

## 7. Yetki / kapsam matrisi

| İzin | platform | organization | branch | self |
|---|---|---|---|---|
| `platform.admin` | ✔ | – | – | – |
| `moderation.review` | ✔ | – | – | – |
| `organization.read` | ✔ | ✔ | – | – |
| `organization.manage` | ✔ | ✔ | – | – |
| `organization.publish` | ✔ | – | – | – |
| `org.branch.manage` | ✔ | ✔ | – | – |
| `org.member.manage` | ✔ | ✔ | – | – |
| `catalog.read` | ✔ | ✔ | – | – |
| `catalog.manage` | ✔ | ✔ | – | – |
| `catalog.publish` | ✔ | ✔ | – | – |
| `taxonomy.manage` | ✔ | – | – | – |
| `review.respond` | ✔ | ✔ | – | – |

"✔", o kapsamda atanabileceği anlamına gelir. Bir üst kapsamdaki platform
ataması alt kapsamlarda da geçerlidir; tersi geçerli değildir.

## 8. İşlevsel olmayan gereksinimler

### 8.1 Güvenlik

- Parolalar bcrypt ile hash'lenir; token'lar yalnızca hash olarak saklanır.
- 5xx yanıtları iç hata mesajı sızdırmaz; her hata tek zarf biçiminde döner.
- Kimlik ve yetki kararları sunucuda verilir; istemci "izinli" bayrağı gönderemez.
- Tenant kimliği gövdeden değil, doğrulanmış kayıttan/rotadan alınır.
- Prodüksiyonda JWT sırları zorunlu, en az 32 karakter ve birbirinden farklı.
- CORS'ta `*` prodüksiyonda reddedilir.

### 8.2 Gözlemlenebilirlik

- Her istek `X-Request-ID` taşır; hata zarfı aynı kimliği döner.
- Modül başlangıç/bitiş olayları yapılandırılmış log ile yazılır.
- `GET /healthz` süreci, `GET /readyz` veritabanını, `GET /version` derlemeyi
  raporlar.

### 8.3 Performans ve ölçek

- Sayfalama zorunlu ve üst sınırlıdır (100).
- Yetki kontrolü indeksli sorguyla yapılır
  (`role_assignments(user_id)`, `(scope, scope_id)`).
- Şema değişiklikleri geri alınabilir migration dosyalarıyla yapılır.

### 8.4 Sürümleme

- `contract/core` yüzeyi semver ile yönetilir. Alan eklemek minor, alan
  kaldırmak/değiştirmek major'dır.
- Modül adı, bağımlılıkları ve migration sırası sözleşmenin parçasıdır.

## 9. Veri sahipliği

Her tablo tek bir modüle aittir. Başka bir modül o tabloyu doğrudan sorgulamaz;
`contract/core` arayüzünü kullanır. Ayrıntı: `doc/ARCHITECTURE.md` §4.

## 10. Motor sınırı

Bir motor:

1. Kendi Go modülünü ve kendi PostgreSQL şemasını getirir.
2. İzinlerini `RegisterPermission` ile kaydeder.
3. Tenant kontrollerini `Authorizer` ile yapar.
4. Sunum kaydına `ExtensionRef` ile bağlanır.
5. Rotasını `/api/v1/ext/<motor>` altında açar.

Çekirdek motorun tablolarını, iş kurallarını veya durum makinesini bilmez.
Ayrıntı: `doc/ENGINE_CONTRACT.md`.

## 11. Kabul kriterleri (V1)

- [x] `go build ./...`, `go vet ./...`, `go test ./...` yeşil.
- [x] `go run ./cmd/depcheck` sınır ihlali bulmuyor.
- [x] Kimlik: kayıt, giriş, yenileme, çıkış, parola sıfırlama, e-posta doğrulama.
- [x] Yetki: rol/izin/kapsam; varsayılan ret; platform ataması alt kapsamda geçerli.
- [x] İşletme: oluşturma, profil, şube, üyelik, davet, sahiplik devri, durum geçişi.
- [x] Katalog: taksonomi, organizasyon kategorileri, sunum, varyant, yayın durumu.
- [x] Her tenant rotası `:org_id` üzerinden yetkilendiriliyor.
- [x] Ekonomi ve AI modüllerine ait kod, config, migration ve doküman kalmadı.
- [x] Tek `PRD.md`, `doc/ARCHITECTURE.md`, `doc/ENGINE_CONTRACT.md`, `doc/ADR.md`.

## 12. Yol haritası

| Sürüm | İçerik |
|---|---|
| **V1 (bu)** | L0 kernel + L1 identity/authz/files/notifications + L2 organization/catalog |
| V1.1 | Bölge tohumlama (`data-kaynak/**`), arama iyileştirmeleri, işletme listeleme filtreleri |
| V2 | İlk motor: randevu veya sipariş (ayrı depo, sürümlenmiş sözleşme) |
| V2+ | Her motor kendi deposunda; çekirdek yalnızca sözleşme yüzeyini büyütür |

## 13. Riskler

| Risk | Etki | Önlem |
|---|---|---|
| Çekirdeğe sektör mantığı sızması | Yeniden kullanılabilirlik biter | PRD §3 + depcheck + motor sözleşmesi |
| Tenant kimliğinin gövdeden alınması | Veri sızıntısı | Rota türetimi + kayıt sahipliği kontrolü |
| Yetki modelinin "admin her şeyi yapar"a indirgenmesi | Denetlenemez erişim | Platform rolleri de normal atamadır |
| Kategori serbest metne dönmesi | Keşif ve rapor tutarsızlığı | Merkezi taksonomi zorunlu |
| Şema sahipliğinin kaybolması | Modüller birbirinin tablosunu yazar | `doc/ARCHITECTURE.md` §4 + migration kaynağı başına sahiplik |
