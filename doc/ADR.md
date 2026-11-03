# ADR — Mimari Karar Kayıtları

Her karar; bağlam, karar ve sonuçlarıyla birlikte yazılır. Yeni bir karar
eklerken numarayı artırın, eski kararı silmeyin; değiştirilecekse yeni bir ADR
yazıp "şunun yerine geçer" notu düşün.

---

## ADR-001: Çekirdek sektör bağımsızdır; ekonomi ve AI kaldırıldı

**Bağlam.** Depoda L2 ekonomi (cüzdan, sipariş, ödeme, abonelik) ve L3 AI
modülleri vardı. Her ikisi de "her uygulama bunu ister" varsayımına dayanıyordu.

**Karar.** İkisi de kaldırıldı. Çekirdek yalnızca kimlik, yetki, işletme ve
katalog taşır. Ödeme/abonelik bir sektör motorudur; AI bir adaptördür.

**Sonuç.**
- Ödeme yapmayan bir uygulama cüzdan şemasını taşımaz.
- Model sağlayıcısı seçimi uygulamanın kararı olur.
- Bir motor gerektiğinde `doc/ENGINE_CONTRACT.md` üzerinden bağlanır.

---

## ADR-002: Kullanıcıda tek `role` alanı yerine kapsamlı atama

**Bağlam.** `users.role` alanı bir kişinin tek bir rolünü tutabiliyordu. Bir
kişi A işletmesinde sahip, B işletmesinde personel olabilir; tek alan bunu
ifade edemez.

**Karar.** `users.role` yalnızca **platform rolü** olarak kalır (`user`,
`admin`, `moderator`). İşletme kapsamlı yetki `role_assignments` tablosunda
`(user, role, scope, scope_id)` olarak tutulur.

**Sonuç.**
- Aynı kişi birden çok işletmede farklı yetkilere sahip olabilir.
- Yetki değişikliği kullanıcı kaydını değiştirmez.
- Platform rolü ile işletme rolü karışmaz.

---

## ADR-003: Varsayılan ret ve rotadan türetilen tenant

**Bağlam.** "Rolü yoksa kullanıcıdır" veya "gövdedeki `organization_id`'ye güven"
yaklaşımları veri sızıntısına açıktır.

**Karar.**
- Yetki kontrolü eşleşen aktif atama bulamazsa reddeder.
- Tenant kimliği yalnızca rota parametresinden (`:org_id`, `:branch_id`) veya
  oturum sahibinden alınır.
- Servis, yüklediği kaydın tenant'a ait olduğunu ayrıca doğrular.

**Sonuç.**
- Bir istemci kendi tenant'ını "seçemez".
- Başka tenant'ın kaydı 403 değil 404 döner; varlık doğrulanmaz.
- `Require` çağrısını unutmak bir güvenlik açığıdır; her tenant rotası için test
  yazılmalıdır.

---

## ADR-004: Rol hiyerarşi değil, izin kümesidir

**Bağlam.** `owner > manager > staff` gibi bir hiyerarşi ilk bakışta pratiktir
ama "manager şu izne sahip olsun, owner olmasın" gibi istisnaları ifade edemez.

**Karar.** Rol, izinlerin kümesidir. İki rol aynı kapsamda birleşir (union).
Hiyerarşi yoktur.

**Sonuç.**
- İstisnalar yeni rol tanımlayarak yapılır.
- "Şu rolü olan herkes" sorgusu izin birleşimiyle yapılır.
- Rol sayısı artabilir; bu, hiyerarşinin gizli istisnalarından daha okunurdur.

---

## ADR-005: Platform ataması aşağı doğru geçerlidir, tersi değil

**Bağlam.** Platform yöneticisi her işletmede çalışabilmelidir. Bir işletme
sahibi ise platform yetkisi kazanmamalıdır.

**Karar.** Etkin izin kümesi, istenen kapsamın atamaları ile `platform`
kapsamındaki atamaların birleşimidir. Organizasyon ataması `platform`
kapsamına yükselmez. Bunun üstünde, `users.role = admin` olan hesap kök
yetkilidir ve atama tablosunda kaydı olmasa da platform işlemlerini yapabilir;
`moderator` yalnızca inceleme/okuma izinlerine sahiptir. Bu istisna, ilk
organizasyonu yayınlayacak birinin var olması için gereklidir.

**Sonuç.**
- Tek bir platform ataması tüm tenant'larda çalışır.
- Bir işletmedeki `owner`, başka işletmede hiçbir şey açmaz.
- Platform rolü atamadan önce gelir; bu yüzden `users.role` yalnızca güvenilir
  operatörler tarafından değiştirilmelidir.
- Platform ataması hâlâ kullanılabilir ve denetim izinde görünür.

---

## ADR-006: Merkezi taksonomi zorunludur

**Bağlam.** Her uygulama kendi serbest metin kategorilerini üretirse keşif,
filtreleme ve raporlama tutarsız olur.

**Karar.** Kategori yalnızca `taxonomy_nodes` tablosundan gelir. Organizasyon ve
sunum kategorileri bu düğümlere yabancı anahtarla bağlanır.

**Sonuç.**
- Yeni sektör kategorisi eklemek bir platform kararıdır.
- Bir motor kendi kategori ağacını tanımlamaz.
- Kategori yeniden adlandırıldığında tüm kullanımlar tutarlı kalır.

---

## ADR-007: Motor verisi `ExtensionRef` ile bağlanır, çekirdeğe sızmaz

**Bağlam.** Randevu süresi, sipariş SKU'su, ders kontenjanı gibi alanlar her
sektörde farklıdır. Bunları çekirdek tabloya eklemek şemayı sektöre kilitler.

**Karar.** Sunum ve varyant kayıtları opsiyonel bir `ExtensionRef`
(`extension_id`, `resource_type`, `resource_id`, `label`) taşır. Çekirdek bu
referansı saklar, okumaz ve doğrulamaz.

**Sonuç.**
- Motor kendi şemasında kendi alanlarını tutar.
- Çekirdek şeması sektörden bağımsız kalır.
- Referansın bütünlüğü motorun sorumluluğundadır.

---

## ADR-008: Organizasyon oluşturma tek transaction değildir

**Bağlam.** Organizasyon, sahip üyeliği ve `owner` ataması üç ayrı tabloya
yazılır. Tek transaction'a almak mümkündür ama `authz` servisi çağrısı modül
sınırını aşar.

**Karar.** V1'de üç adım sırayla yapılır; hata olursa istek hata döner ve
yarım kalan organizasyon operatör müdahalesi gerektirir.

**Sonuç.**
- Basit ve modül sınırına saygılı.
- Nadir bir hata durumunda sahipsiz organizasyon kalabilir.
- V2'de outbox/saga veya tek transaction'lı bir iç servis ile iyileştirilebilir.

---

## ADR-009: Organizasyon kapanınca motor verisi silinmez

**Bağlam.** Bir işletme kapanabilir. Motorun geçmiş kayıtları (sipariş, fatura,
randevu) yasal ve operasyonel nedenlerle saklanmak zorunda olabilir.

**Karar.** Çekirdek yalnızca kendi durumunu değiştirir ve atamaları iptal eder.
Motor verisine dokunmaz.

**Sonuç.**
- Geçmiş korunur.
- Motor kendi saklama/anonimleştirme politikasını uygular.
- "Kapanan işletme hâlâ işlem yapıyor" riski, atamaların iptaliyle engellenir.

---

## ADR-010: Modüller aynı şemayı paylaşır, sınırı kod korur

**Bağlam.** Modül başına ayrı PostgreSQL şeması daha temiz görünür ama çapraz
sorguları zorlaştırır ve migration sırasını karmaşıklaştırır.

**Karar.** Çekirdek modülleri `public` şemasını paylaşır; her modül kendi
migration kaynağını getirir. Tablo sahipliği belgede ve kod incelemesinde
korunur. Motorlar kendi şemalarını getirir.

**Sonuç.**
- Basit migration ve join.
- Modül sınırı derleyiciyle değil, disiplin ve depcheck ile korunur.
- Motorlar bu paylaşımdan etkilenmez.

---

## ADR-011: Üç seviye; L3 yok

**Bağlam.** Önceki model L0/L1/L2/L3 idi; L3 AI demekti.

**Karar.** Seviyeler `L0` (kernel), `L1` (identity, authz, files,
notifications), `L2` (organization, catalog) olarak tanımlandı. L3 kaldırıldı.

**Sonuç.**
- `corectl new --level` üç değer alır.
- AI bir seviye değil, uygulamanın ekleyeceği bir adaptördür.
- L2 artık "işlem motoru" değil, "hizmet katmanı"dır.
