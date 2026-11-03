# HİZMET KATMANI — BAĞIMSIZ ÇEKİRDEK (CORE) PRD + AI UYGULAMA PROMPT'U

**Belge türü:** Product Requirements Document (PRD) + Kodlama ajanı uygulama talimatı  
**Ürün aşaması:** V1 — Bağımsız, sektörlerden bağımsız çekirdek  
**Stratejik ufuk:** 2027–2029  
**Temel karar:** Çekirdek geliştirilecek; randevu, e-ticaret, teklif, eğitim, proje, rezervasyon veya başka bir sektör işlem motoru geliştirilmeyecek.  
**Teknoloji kararı:** Bu belge belirli bir programlama dili, framework, veritabanı, bulut sağlayıcısı veya fiziksel dağıtım topolojisi dayatmaz. Mevcut projenin teknolojisi ve gerçek ihtiyaçları esas alınır.

---

# 0. KODLAMA AJANI İÇİN ANA TALİMAT

Bu belgeyi yalnızca fikir veya tavsiye metni olarak değil, **ürün gereksinimi ve kapsam sözleşmesi** olarak uygula.

Görevin, farklı sektörlerde kullanılabilecek bağımsız bir **Hizmet Katmanı Çekirdeği (Service Layer Core)** tasarlamak ve mevcut proje koşullarına göre geliştirmektir.

## Değiştirilemez ana kural

> **Yalnızca çekirdeği üret. Hiçbir sektörün işlem motorunu üretme. Ancak çekirdek, gelecekte geliştirilecek farklı işlem motorlarının ortak sözleşmeler üzerinden bağlanmasına hazır olsun.**

Bu kural aşağıdaki anlamlara gelir:

- Randevu/rezervasyon motoru oluşturma.
- Ürün, stok, sepet, sipariş veya teslimat motoru oluşturma.
- Teklif, fiyat kalemi, pazarlık veya proje yürütme motoru oluşturma.
- Ders, sınıf, devam, sınav veya öğrenme yönetimi motoru oluşturma.
- Etkinlik, görev, bütçe veya organizasyon motoru oluşturma.
- Sağlık kayıtları, teşhis, tedavi veya klinik süreç motoru oluşturma.
- Abonelik tahsilatı, satıcı ödemesi, komisyon veya para dağıtım motoru oluşturma.
- Bu motorları örneklemek amacıyla çalışan demo iş akışları da oluşturma.

Bunun yerine:

- Ortak çekirdek varlıklarını ve sorumluluklarını tanımla.
- Çekirdek API'lerini, olay sözleşmelerini ve genişleme noktalarını oluştur.
- Bir motorun çekirdeğe nasıl bağlanacağını tarif eden **soyut entegrasyon sözleşmesini** hazırla.
- Motorun kendi verisinin, kurallarının, durum makinesinin ve iş akışının kendisine ait olmasını güvence altına al.
- Motor bulunmadığında çekirdek yine bağımsız olarak çalışabilsin.
- Henüz motor yokken sahte işlem akışları, örnek sektör tabloları veya gereksiz placeholder modülleri üretme.

## İlk çalışma biçimi

1. Önce mevcut depo/proje yapısını incele. Kullanıcı özellikle istemedikçe mevcut dosyaları silme, yeniden adlandırma veya baştan yazma.
2. Mevcut teknoloji yığınını, mimariyi, veri modelini, kimlik/yetki sistemini, testleri ve güvenlik kontrollerini tespit et.
3. Bu PRD ile mevcut yapı arasında fark analizi çıkar.
4. Çakışma varsa sessizce karar verme: mevcut çalışan davranışı koruyan, en az değişiklik gerektiren çözümü tercih et ve mimari karar kaydına yaz.
5. Uygulamaya başlamadan önce çekirdek sınırlarını ve motor sınırlarını açıkça belgeleyen kısa bir mimari plan hazırla.
6. Ardından yalnızca bu belgede tanımlanan çekirdek kapsamını uygula.
7. Her aşamada test çalıştır; başarısız testleri gizleme veya yalnızca test geçsin diye gereksinimi düşürme.
8. Sonuçta yapılanları, yapılmayanları, bilinen sınırlamaları ve test kanıtlarını raporla.

**Kapsam belirsizliğinde varsayılan karar:** Yeni özellik eklemek yerine kapsam dışında bırak. Sektör mantığını çekirdeğe taşımak yerine uzantı/işlem motoruna bırak.

---

# 1. ÜRÜN TANIMI

## 1.1 Ürün vizyonu

Hizmet Katmanı; farklı sektörlerde faaliyet gösteren işletmelerin kendilerini tanımlamasını, hizmetlerini yayımlamasını, müşteriler tarafından keşfedilmesini ve müşteri–işletme ilişkisinin ortak, güvenli ve denetlenebilir bir altyapı üzerinden kurulmasını sağlayan **sektör bağımsız platform çekirdeğidir**.

Çekirdeğin amacı bütün sektörleri tek bir işlem modeline zorlamak değildir. Amaç, sektörler arasında gerçekten ortak olan temel kabiliyetleri bir kez sağlamak ve sektörlere özgü işlem biçimlerinin bağımsız motorlar olarak eklenebilmesini mümkün kılmaktır.

## 1.2 Ürün vaadi

- Bir hesap, farklı yetki kapsamlarında müşteri, uzman veya işletme çalışanı olarak hareket edebilir.
- Bir işletme, bir veya daha fazla şube/lokasyon ve ekip üyesi tanımlayabilir.
- İşletme, çekirdeğin standart katalog yapısı üzerinden hizmet veya sunum kaydı yayımlayabilir.
- Müşteri, kategori, konum ve ortak katalog bilgileri üzerinden işletme/hizmet keşfedebilir.
- Müşteri ve işletme, ortak mesajlaşma ve bildirim altyapısı üzerinden iletişim kurabilir.
- Yorumlar, şikâyetler, itirazlar, moderasyon ve denetim kayıtları merkezi politikalara göre yönetilebilir.
- Gelecekteki işlem motorları, çekirdeğin tanımladığı sözleşmelere uyarak bağlanabilir; ancak çekirdeğin veri ve iş kurallarını ele geçiremez.

## 1.3 Ürün tanımı olmayan şeyler

Hizmet Katmanı V1 bir pazaryeri işlem motoru, rezervasyon yazılımı, e-ticaret altyapısı, CRM paketi, ERP, LMS, ödeme kuruluşu veya her sektöre özel otomasyon ürünü değildir.

Çekirdek, bu ürünlerin bazı ortak altyapı ihtiyaçlarını karşılayabilir. Fakat bunların sektör kurallarını ve işlemlerini uygulamaz.

---

# 2. TASARIM İLKELERİ

1. **Category-agnostic:** Çekirdek, sektör adını bilmek zorunda kalmadan temel işlevlerini yerine getirebilmelidir.
2. **Engine-agnostic:** Çekirdek, randevu, sipariş, teklif veya başka bir işlem motorunun iç modelini bilmemelidir.
3. **Shared core, separate workflows:** Ortak kimlik ve işletme altyapısı paylaşılır; sektör iş akışları ayrı kalır.
4. **API and contract first:** Motorlarla entegrasyon, belgelenmiş ve sürümlenmiş sözleşmeler üzerinden yapılır.
5. **Extension, not core mutation:** Sektöre özel alanlar çekirdek tablolarına gelişigüzel kolon eklenerek değil, kontrollü uzantılarla yönetilir.
6. **Least privilege:** Her işlem, rol ve kapsam bakımından en az yetki ilkesiyle korunur.
7. **Tenant isolation:** İşletme/kurum verileri birbirinden yalıtılır.
8. **Auditability:** Güvenlik ve kritik yönetim hareketleri izlenebilir olmalıdır.
9. **Data minimization:** Çekirdek, yalnızca ortak işlev için gerekli veriyi tutar.
10. **No speculative complexity:** Henüz doğrulanmamış ihtiyaçlar için mikroservis, event bus, workflow engine, plugin marketplace veya benzeri altyapılar sırf gelecekte gerekebilir diye eklenmez.
11. **Progressive extensibility:** İlk sürüm yalın kalır; yeni motorlar sözleşme bozmadan daha sonra eklenebilir.
12. **Backward compatibility:** API ve veri sözleşmelerindeki değişiklikler sürümleme ve geçiş politikası olmadan mevcut tüketicileri bozmaz.

---

# 3. MİMARİ SINIR: ÇEKİRDEK VE İŞLEM MOTORLARI

## 3.1 Üst seviye kavramsal mimari

```text
                         KULLANICI KANALLARI
             Web / Mobil / Yönetim Paneli / Harici İstemci
                                  |
                                  v
                    HİZMET KATMANI — CORE API
                                  |
     +----------------------------+-----------------------------+
     |                            |                             |
     v                            v                             v
 KİMLİK & YETKİ              İŞLETME & KATALOG             KEŞİF & İLETİŞİM
 Hesap / Rol                 İşletme / Şube               Arama / Mesaj
 Üyelik kapsamı              Ekip / Hizmet kaydı           Bildirim / Yorum
 Erişim politikası           Taksonomi / Bölge             Moderasyon
     |                            |                             |
     +----------------------------+-----------------------------+
                                  |
                                  v
                     ORTAK YÖNETİŞİM KATMANI
              Audit / Gizlilik / Onay / API / Event
              Veri izolasyonu / Politika / Ölçüm
                                  |
                    SÜRÜMLENMİŞ ENTEGRASYON SINIRI
                                  |
              +-------------------+-------------------+
              |                   |                   |
              v                   v                   v
        İŞLEM MOTORU A       İŞLEM MOTORU B       İŞLEM MOTORU C
          (Gelecekte)         (Gelecekte)          (Gelecekte)
        Kendi verisi          Kendi verisi         Kendi verisi
        Kendi kuralları       Kendi kuralları      Kendi kuralları
        Kendi durumları       Kendi durumları      Kendi durumları
```

Bu diyagram kavramsal sınırları gösterir. Monolit, modüler monolit, servis tabanlı yapı veya başka bir fiziksel topoloji dayatmaz. Uygulama biçimi mevcut projenin ihtiyaçlarına göre seçilir; **mantıksal sınırlar her durumda korunur.**

## 3.2 Çekirdeğin sahip olduğu sorumluluklar

Çekirdek aşağıdaki ortak kabiliyetlerin sahibidir:

- Hesap ve kimlik kaydı.
- Rol, izin ve kapsam denetimi.
- İşletme/kurum profili ve sahiplik ilişkisi.
- Şube/lokasyon ve ekip üyeliği.
- Sektör bağımsız hizmet/sunum kataloğu temel alanları.
- Merkezi taksonomi ve kategori kayıtları.
- Bölge ve hizmet alanı temel tanımları.
- Ortak arama/keşif için standart indekslenebilir alanlar ve sorgu sözleşmesi.
- Müşteri–işletme iletişim bağlamı.
- Mesaj ve bildirim altyapısı.
- Yorum, değerlendirme, şikâyet ve moderasyon altyapısı.
- Onay, gizlilik, saklama ve denetim ilkeleri.
- Motor/uzantı kaydı ve entegrasyon sözleşmeleri.
- Ortak olay zarfı, API sürümleme ve ölçüm standardı.

## 3.3 İşlem motorlarının sahip olduğu sorumluluklar

Her motor kendi alanına ait aşağıdaki konularda tek otoritedir:

- Sektöre veya işlem türüne özgü veri modeli.
- İş kuralları ve doğrulamalar.
- Özel durum makinesi ve durum geçişleri.
- Fiyat hesaplama ve özel ücret kalemleri.
- Takvim, slot, kapasite, stok, sepet, sipariş, teklif, görev, ders veya proje mantığı.
- İşlem iptali, iade, değişiklik, yeniden planlama ve özel istisnalar.
- Motorun gerektirdiği sektörel evrak ve özel alanlar.
- Motorun işlem performansı ve sektörel KPI'ları.

Çekirdek bu alanların yerine karar vermez; yalnızca ortak kimlik, işletme, yetki, iletişim, denetim ve entegrasyon hizmetleri sunar.

---

# 4. ÇEKİRDEK KAPSAMI — V1'DE ÜRETİLECEKLER

Aşağıdaki gereksinimler çekirdek kapsamındadır. Mevcut projede eşdeğer ve güvenli bir kabiliyet varsa yeniden üretmek yerine mevcut kabiliyet kontrollü biçimde kullanılmalıdır.

## 4.1 Hesap ve kimlik

**Amaç:** Platformdaki insan veya sistem aktörünü tekil ve güvenli biçimde temsil etmek.

Çekirdek şunları sağlamalıdır:

- Tekil hesap kimliği.
- Hesap durumu: etkin, askıda, kapalı/silinme sürecinde gibi ortak hesap seviyesinde anlamlı değerler.
- Temel profil bilgileri ve iletişim bilgileri.
- E-posta/telefon doğrulama durumu; kullanılan doğrulama yöntemi mevcut kimlik altyapısına göre belirlenir.
- Kimlik doğrulama oturumu veya mevcut auth sağlayıcısına bağlanma noktası.
- Hesabın birden fazla işletme/kurumla ilişkilendirilebilmesi.
- Hesabın müşteri, işletme sahibi, ekip üyesi, moderatör veya yönetici gibi farklı bağlamlarda işlem yapabilmesi.
- Hesap kapatma, veri dışa aktarma ve silme taleplerinin yönetişim akışına bağlanması.

**Çekirdeğe gömülmeyecekler:** Sektöre özel kullanıcı profilleri, öğrenci numarası, hasta dosyası, sürücü/kurye operasyon profili, eğitmen sertifika süreçleri veya motorun özel kullanıcı nitelikleri. Bunlar yalnızca ihtiyaç ve hukuki dayanak varsa ilgili uzantı tarafından yönetilir.

## 4.2 Rol, izin ve kapsam (RBAC + gerektiğinde ABAC)

**Amaç:** Kullanıcının ne yapabileceğini ve hangi kaynak üzerinde yapabileceğini merkezi olarak kontrol etmek.

Gereksinimler:

- Rol tek başına yeterli kabul edilmemeli; izinler kaynak ve kapsamla birlikte değerlendirilmelidir.
- Kapsam türleri asgari olarak platform, işletme, şube ve kullanıcının kendi kaydı seviyelerini desteklemelidir.
- Bir kullanıcı bir işletmede yönetici, başka bir işletmede yalnızca ekip üyesi olabilir.
- İşletme sahibi, yöneticisi, çalışanı, müşterisi, platform yöneticisi ve moderatör gibi ortak roller tanımlanabilir; kesin rol listesi konfigüre edilebilir olmalıdır.
- Motorlar kendi eylemlerini çekirdeğin yetkilendirme servisine/sözleşmesine bildirir veya ortak policy kontrolünü kullanır.
- Çekirdek, motorun tüm özel izinlerini önceden bilmek zorunda değildir; motor izin anahtarlarını kontrollü bir namespace altında kaydedebilir.
- Yetki kontrolü yalnızca arayüzde değil, sunucu tarafında her korumalı işlemde yapılmalıdır.
- Varsayılan davranış **deny by default** olmalıdır.

**Çekirdeğe gömülmeyecekler:** Motorun özel rol hiyerarşisi, örneğin “baş eğitmen”, “depo sorumlusu”, “doktor asistanı”, “proje onaycısı” gibi sektörel roller. Motor bunları çekirdeğin izin/kapsam mekanizması üzerinde tanımlar.

## 4.3 İşletme / kurum profili

**Amaç:** Hizmet sağlayan tüzel/gerçek işletme veya organizasyonun ortak kimliğini ve platformdaki varlığını temsil etmek.

Gereksinimler:

- Tekil işletme/kurum kimliği.
- Görünen ad, kısa açıklama, profil görseli/logo, genel iletişim kanalları.
- Durum: taslak, doğrulama bekliyor, aktif, askıda, kapalı gibi ortak yayın/yönetişim durumları.
- İşletme sahibi ve yetkili üyelerle ilişki.
- Birden fazla şube/lokasyonla ilişki.
- Birden fazla kategori/taksonomi kaydıyla ilişki; kategori atamaları merkezi taksonomiye dayanmalıdır.
- Genel çalışma bilgileri yalnızca ortak profil niteliğindeyse tutulmalı; sektör saat/kapasite kuralları motor kapsamındadır.
- İşletme sahipliği ve yönetim yetkisi denetlenebilir olmalıdır.
- İşletme doğrulama seviyesi/politika durumu tutulabilir; doğrulama yöntemi ve sektörel belgeler politika/uzantı kapsamındadır.

**Çekirdeğe gömülmeyecekler:** POS, muhasebe, depo, bordro, CRM satış hunisi, şube vardiyası, sektörel ruhsat iş akışı, üretim veya tedarik zinciri.

## 4.4 Şube ve lokasyon

**Amaç:** İşletmenin fiziksel veya hizmet sunduğu yerleri ortak biçimde tanımlamak.

Gereksinimler:

- Bir işletmenin sıfır veya daha fazla şubesi/hizmet lokasyonu olabilir.
- Adres, ülke, il/şehir, ilçe/bölge ve koordinat gibi genel alanlar desteklenebilir.
- Lokasyon fiziksel şube, hizmet verilen alan veya çevrimiçi hizmet gibi genel türleri ayırt edebilmelidir.
- İşletmenin hizmet alanı bir veya daha fazla bölge ile ilişkilendirilebilir.
- Hassas lokasyonların görünürlüğü politika ile sınırlandırılabilmelidir.
- Konum verisi doğrulanmalı ve tutarlı biçimde saklanmalıdır.

**Çekirdeğe gömülmeyecekler:** Rota optimizasyonu, kurye dağıtımı, araç takip, sevkiyat, gerçek zamanlı sürücü konumu, saha ekiplerinin rota planı, mesafe bazlı dinamik fiyatlama.

## 4.5 Uzman / ekip üyesi

**Amaç:** Bir işletme adına hizmet sunan veya işletme alanında görevli kişiyi temsil etmek.

Gereksinimler:

- Hesap ile işletme/şube arasındaki üyelik ilişkisi.
- Üyelik rolü ve kapsamı.
- Üyelik durumu ve davet/kabul/çıkarma kayıtları.
- Profilde gösterilmesine izin verilen genel ad, görsel ve kısa tanıtım bilgileri.
- Kişinin birden fazla işletme veya şubede görev alabilmesi.
- Ekip üyesi kaydı, kullanıcı hesabı ile aynı şey olarak modellenmemeli; davet edilen kişi henüz hesap oluşturmamış olabilir.

**Çekirdeğe gömülmeyecekler:** Uzman takvimi, vardiya, ders yükü, performans primi, personel bordrosu, sertifika yenileme workflow'u, klinik nöbet veya proje görev ataması.

## 4.6 Hizmet / sunum kataloğu

**Amaç:** İşletmenin müşteriye sunduğu şeyi, sektör bağımsız ortak alanlarla tanımlamak.

Çekirdek katalog kaydı mümkün olduğunca yalın olmalıdır:

- Tekil katalog/sunum kimliği.
- Sahip işletme ve isteğe bağlı şube ilişkisi.
- Ad, kısa/açıklayıcı metin ve medya referansları.
- Merkezi kategori/taksonomi ilişkisi.
- Yayın durumu ve görünürlük.
- Hizmetin genel sunum biçimi: yerinde, çevrimiçi, müşterinin konumunda veya karma gibi geniş ve ortak sınıflar.
- Genel fiyat gösterimi gerekiyorsa yalnızca bilgilendirici ve opsiyonel bir fiyat sunumu; hesaplama, vergi, indirim, ücret kalemi veya tahsilat mantığı değildir.
- Dil, etiket ve arama için ortak metadata.
- İlgili uzantı veya motorun kaynak kimliğine referans verebilme.

Hizmet ile hizmet varyantı ayrımı gerekebilir; ancak varyantlar çekirdekte yalnızca ad, referans ve genel görünürlük gibi ortak alanlara sahip olmalıdır. Süre, kapasite, SKU, paket içeriği, katılımcı sayısı, ders seviyesi, lisans veya teslimat seçeneği gibi alanlar özel motorun sorumluluğudur.

**Çekirdeğe gömülmeyecekler:** Sepet, SKU, stok, fiyat motoru, paket/opsiyon hesaplama, takvim slotu, ders planı, teklif kalemi, proje kapsamı veya abonelik kotası.

## 4.7 Taksonomi ve kategori yönetimi

**Amaç:** Sektörleri ve alt kategorileri tekil, tutarlı ve yönetilebilir bir ana kayıt altında toplamak.

Gereksinimler:

- Tekil kategori kaydı ve kararlı kimlik.
- Hiyerarşik kategori yapısı.
- Çoklu dil/yerelleştirme ihtiyacına uygun ad ve açıklama alanları.
- Kategori slug/kod standardı.
- Etkin/pasif ve yayınlanabilirlik durumu.
- Kategori birleştirme, taşıma ve devre dışı bırakma sırasında eski referansları yönetme politikası.
- Etiketler ile kategoriler birbirinden farklı kavramlar olmalıdır.
- Katalog kayıtları ve işletmeler kategori kimliğine referans vermelidir; serbest metin kategori adı ana kayıt yerine geçmemelidir.
- Sektöre özel alan şeması gerekiyorsa kategori kaydı bu şemanın sahibi olmaz; yalnızca uygun uzantı/şema referansını taşıyabilir.

**Çekirdeğe gömülmeyecekler:** Her kategori için sabit ve kod içine yazılmış özel form alanları. Örneğin market kategorisi için stok adedi, astroloji için doğum haritası bilgisi, fitness için antrenman programı veya sağlık için klinik kayıt alanları çekirdekte bulunmaz.

## 4.8 Bölge ve hizmet kapsamı

**Amaç:** Bir işletmenin nerede hizmet sunduğunu genel düzeyde tanımlamak.

Gereksinimler:

- Ülke/şehir/bölge/ilçe gibi standart coğrafi referanslarla ilişki.
- İşletme veya katalog kaydının genel hizmet alanı.
- Çevrimiçi hizmet seçeneği.
- Arama/filtrelemede kullanılabilecek standart coğrafi alanlar.

**Çekirdeğe gömülmeyecekler:** Canlı kurye takibi, saha iş emri rotası, teslimat ücret motoru, seyahat süresi tahmini veya dinamik kapsama/kapasite motoru.

## 4.9 Keşif ve arama altyapısı

**Amaç:** Müşterinin işletme ve sunumları ortak alanlar üzerinden bulmasını sağlamak.

Gereksinimler:

- İşletme, kategori, konum ve katalog temel alanları için ortak arama sözleşmesi.
- Yayınlanmamış, askıda veya görünürlüğü kısıtlanmış kayıtlar arama sonuçlarında sızmamalıdır.
- Filtreleme ve sıralama parametreleri sürümlü olmalıdır.
- Kategori bazlı filtre alanları varsa uzantı tarafından sağlanmalı; çekirdek bu alanların sektör anlamını bilmemelidir.
- Sıralama/ranking politikası açıklanabilir ve kötüye kullanıma karşı korunabilir olmalıdır.
- Arama motoru ürünü veya altyapısı mevcut proje ihtiyaçlarına göre seçilir; teknoloji dayatılmaz.

**Çekirdeğe gömülmeyecekler:** Her motorun kendi işlem kayıtlarını arama indeksine kontrolsüz şekilde kopyalamak; motorun rezervasyon müsaitliği, stokta bulunma veya teklif uygunluğu gibi canlı işlem kararlarını çekirdek araması üzerinden vermek. Böyle sorgular ilgili motora yönlendirilir.

## 4.10 İletişim bağlamı, mesaj ve dosya referansı

**Amaç:** Müşteri ile işletme/uzman arasında ortak iletişim altyapısı sağlamak.

Gereksinimler:

- Konuşma/iletişim kaydı ve katılımcılar.
- Konuşmanın işletme, şube, katalog öğesi veya harici motor kaynağıyla ilişkilendirilebilmesi.
- Mesaj göndereni, zamanı, türü ve teslim/okunma metadata'sı.
- Yetkiye bağlı mesaj görünürlüğü.
- Dosyalar için güvenli referans; dosya erişimi ayrıca yetkilendirilmelidir.
- İstenmeyen iletişim, engelleme, raporlama ve moderasyon desteği.
- Motorlar kendi işlem bağlamları için çekirdek konuşma altyapısını kullanabilir veya sözleşmeyle ayrı bir iletişim modülü bağlayabilir.

**Çekirdeğe gömülmeyecekler:** Motorun özel otomatik mesaj senaryoları, teklif pazarlığı kuralları, sipariş durumuna göre zorunlu mesaj zinciri, klinik hasta iletişimi gibi özel iş akışları. Çekirdek yalnızca genel mesajlaşma kabiliyetini sağlar.

## 4.11 Bildirim altyapısı

**Amaç:** Ortak bildirim dağıtımını ve kullanıcı tercihlerini yönetmek.

Gereksinimler:

- Uygulama içi bildirim kaydı.
- E-posta, SMS, push gibi kanallar için sağlayıcıdan bağımsız gönderim arayüzü.
- Kullanıcı bildirim tercihleri ve gerekli izin/ret kayıtları.
- Şablon ve dil desteği.
- Tekrarlı olaylarda idempotent gönderim veya tekrar önleme.
- Başarısız gönderim ve teslim deneme kayıtları.
- Motor olaylarının çekirdek bildirim sözleşmesine bağlanabilmesi.

**Çekirdeğe gömülmeyecekler:** Randevu hatırlatma kuralları, teslimat mesaj senaryosu, ders devamsızlık uyarısı veya proje kilometre taşı alarmının iş mantığı. Motor olayı ve hedef alıcıyı tanımlar; çekirdek ortak kanal ve tercih politikasını uygular.

## 4.12 Yorum, değerlendirme, şikâyet ve itiraz

**Amaç:** Platform genelinde güven ve geri bildirim altyapısı sağlamak.

Gereksinimler:

- İşletme veya katalog öğesi için değerlendirme/yorum.
- Yorumun yazarı, hedefi, zamanı ve moderasyon durumu.
- Yorum düzenleme/silme ve itiraz geçmişi.
- Şikâyet/raporlama kaydı ve gerekçe sınıfları.
- Moderasyon kararının aktörü, zamanı ve nedeni.
- Sahte/tekrarlı değerlendirme ve manipülasyon riskini azaltacak kontroller.
- Bir değerlendirme “doğrulanmış deneyim” olarak işaretlenecekse bunun kanıtı ilgili motorun imzalı/sürümlü olay veya referansıyla doğrulanmalıdır. Çekirdek kendi başına gerçekleşmiş işlem varsaymamalıdır.
- İşletmenin yanıt verebilmesi için yetki kontrolü.

**Çekirdeğe gömülmeyecekler:** Sektöre özel başarı puanları, eğitim notu, sağlık sonucu, teslimat performans metriği veya motorun özel kalite skoru. Bunlar gerekiyorsa ilgili motorun kendi ölçüm alanıdır.

## 4.13 Moderasyon ve platform yönetişimi

**Amaç:** İçerik ve davranış politikalarını merkezi, denetlenebilir biçimde uygulamak.

Gereksinimler:

- İçerik/hesap/işletme raporlama.
- Politika kodu, ihlal türü ve karar kaydı.
- İnceleme kuyruğu ve moderasyon rol/kapsamı.
- Uyarı, gizleme, geçici askıya alma ve itiraz gibi ortak platform eylemleri.
- Karar geçmişi ve denetim izi.
- Sektör riskine göre ek politika tanımı yapılabilmesi.

**Çekirdeğe gömülmeyecekler:** Mesleki lisansın doğruluğuna ilişkin uzman kararını otomatik verme, sağlık değerlendirmesi, finansal tavsiye kalitesi veya hukuki geçerlilik yorumu. Çekirdek politika altyapısı sağlar; uzmanlık ve sektörel uygunluk doğrulamasını ilgili yetkili süreç yürütür.

## 4.14 Denetim kayıtları, gizlilik ve veri yönetişimi

**Amaç:** Kritik işlemlerin kim tarafından, ne zaman, hangi kapsamda yapıldığını izlemek.

Gereksinimler:

- Kritik kimlik, yetki, işletme, moderasyon ve veri erişimi hareketleri audit log üretmelidir.
- Audit kaydı normal kullanıcı tarafından değiştirilememelidir.
- Kayıtlar aktör, eylem, hedef, zaman, sonuç ve korelasyon kimliği içermelidir.
- Hassas veri loglara açık metin olarak yazılmamalıdır.
- Veri saklama, silme, dışa aktarma ve anonimleştirme politikaları tanımlanmalıdır.
- Amaç sınırlaması ve veri minimizasyonu uygulanmalıdır.
- Kategori/uzantı verilerinin erişim kuralları çekirdek güvenlik kontrollerini atlayamamalıdır.
- Veri saklama süresi ve bölgesel hukuki zorunluluklar yapılandırılabilir politika olarak ele alınmalıdır.

**Çekirdeğe gömülmeyecekler:** Sağlık, finans, çocuk, biyometrik veya başka hassas verilerin iş anlamı ve özel kayıt modeli. Böyle veriler yalnızca açık gereksinim, hukuki dayanak, erişim ve saklama politikası varsa özel ve izole uzantıda ele alınır. V1'de bu tür veri alanları oluşturma.

## 4.15 Ortak ölçüm ve olay standardı

**Amaç:** Platformun ortak kullanımını ölçmek; motorların özel analitiğini çekirdeğe taşımamak.

Gereksinimler:

- Ortak olay zarfı: event_id, event_name, occurred_at, actor_ref, tenant_ref (uygunsa), resource_ref, source, schema_version, correlation_id ve gerekli metadata.
- Ortak kullanım olayları: hesap oluşturma, işletme oluşturma, katalog yayımlama, arama, profil görüntüleme, konuşma başlatma, mesaj gönderme, yorum gönderme gibi genel olaylar.
- Olaylar mümkün olduğunca kişisel veri içermemeli; ölçüm için gerekli minimum bağlam kullanılmalıdır.
- Event isimleri ve şemaları belgelenmeli ve sürümlenmelidir.
- Motorun özel KPI'ları çekirdek event listesine zorla eklenmemeli; motor kendi event namespace'ini kullanabilmelidir.

**Çekirdeğe gömülmeyecekler:** Rezervasyon dönüşüm hunisi, stok devir hızı, ders başarı skoru, teklif kazanma oranı veya proje kârlılık hesabı gibi motor analitiği.

---

# 5. ÇEKİRDEKTEKİ ORTAK VARLIKLAR VE ASGARİ VERİ MODELİ

Bu bölüm kavramsal varlıkları tanımlar; doğrudan tablo adı, ORM veya fiziksel veritabanı şeması dayatmaz. İsimler mevcut projenin adlandırma standardına uyarlanabilir.

| Varlık | Çekirdekteki anlamı | Çekirdeğin sahibi olduğu veri | Çekirdeğin sahibi olmadığı veri |
|---|---|---|---|
| Account | Platform hesabı | Kimlik referansı, temel profil, hesap durumu | Sektörel müşteri/uzman profili |
| RoleAssignment | Rol ve kapsam ataması | Rol, izin, scope, geçerlilik | Motorun özel rol hiyerarşisi |
| Organization/Business | İşletme/kurum | Ortak profil, sahiplik, yayın durumu | ERP, muhasebe, stok, operasyon |
| Branch/Location | Şube veya hizmet lokasyonu | Genel adres, lokasyon türü, görünürlük | Rota, vardiya, kapasite |
| Membership | Hesap–işletme/ekip ilişkisi | Davet, kabul, rol ve üyelik durumu | Bordro, görev planı |
| Offering | Genel hizmet/sunum kaydı | Ad, açıklama, kategori, yayın, ortak metadata | SKU, slot, teklif kapsamı, ders içeriği |
| TaxonomyNode | Merkezi kategori kaydı | Tekil kategori kimliği, hiyerarşi, ad | Kategoriye özel işlem kuralları |
| ServiceArea | Genel hizmet bölgesi | Bölge referansı, kapsam türü | Teslimat/rota hesaplama |
| Conversation | İletişim bağlamı | Katılımcılar, ilişki referansları, görünürlük | Motorun özel müzakere süreci |
| Message | Mesaj | Gönderen, içerik referansı, zaman, durum metadata'sı | Motorun özel otomatik iş akışı |
| Notification | Bildirim | Alıcı, kanal, şablon, gönderim sonucu | Sektörel hatırlatma kuralı |
| Review | Değerlendirme | Puan/yorum, hedef, moderasyon | Sektörel kalite skoru |
| Report/Appeal | Şikâyet/itiraz | Başvuru, gerekçe, karar referansı | Mesleki/teknik uzman kararı |
| AuditEvent | Denetim olayı | Aktör, eylem, hedef, zaman, sonuç | Hassas iş verisinin açık kopyası |
| Extension | Uzantı kaydı | Kimlik, sürüm, izin, veri namespace'i | Uzantının iş verisi |
| EngineBinding | Motor kaynak bağlantısı | Motor kimliği ve dış kaynak referansı | Motorun asıl işlem kaydı |
| EventEnvelope | Ortak olay zarfı | Olay kimliği, kaynak, sürüm, korelasyon | Motorun tüm event içeriği |

## 5.1 Kimlik ve referans ilkeleri

- Her çekirdek kaydının kararlı, tekil ve dışarıya açılması güvenli bir kimliği olmalıdır.
- Motor kaynak kimliği ile çekirdek kimliği birbirine karıştırılmamalıdır.
- Motorun kayıtları çekirdeğe kopyalanmamalı; gerektiğinde referans verilmelidir.
- Bir kaydın sahibi/tenant kapsamı açıkça tanımlanmalıdır.
- İlişki kurulurken yalnızca istemciden gelen ID'ye güvenilmemeli; sunucu tarafında sahiplik ve yetki doğrulanmalıdır.

## 5.2 Genel işlem referansı — işlem motoru değildir

Çekirdekte yalnızca entegrasyon ve iletişim bağlamını bağlamak için minimal bir **InteractionReference / EngineBinding** bulunabilir.

Bu kayıt en fazla şunları taşımalıdır:

- Kaynak motorun kararlı kimliği ve sürümü.
- Motorun kendi kaynak kaydına ait referans.
- Çekirdekteki ilgili hesap, işletme, şube veya katalog referansları.
- İsteğe bağlı ortak görünüm etiketi veya normalize edilmiş özet durum.
- Oluşturulma/güncellenme zamanı ve korelasyon kimliği.

Bu yapı şunları yapmamalıdır:

- İşlem yaratma veya yürütme.
- İşlem için ortak zorunlu durum makinesi tanımlama.
- Rezervasyon/sipariş/teklif/proje kayıtlarını kendi veri tablosunda tutma.
- Motorun durum geçişine karar verme.
- Fiyat, kapasite, uygunluk, stok veya tahsilat hesaplama.

**Durum ilkesi:** Çekirdek, motorun özel durumunu sahiplenmez. Kullanıcı arayüzünde birleşik bir özet gösterilmesi gerekiyorsa motorun sunduğu, sürümlenmiş ve yalnızca bilgilendirme amaçlı bir özet kullanılır. Bu özet motorun gerçek durum makinesinin yerine geçmez.

---

# 6. UZANTI (EXTENSION) VE MOTOR UYUMLULUK MODELİ

## 6.1 Temel kural

Sektörel farklılıklar çekirdek koduna koşul blokları eklenerek çözülmemelidir.

**Yanlış yaklaşım:**

```text
if category == "market": stock_field_required = true
if category == "fitness": trainer_schedule_required = true
if category == "education": classroom_id_required = true
if category == "astrology": birth_chart_required = true
```

**Doğru yaklaşım:**

```text
Core
  ├── ortak hesap / işletme / katalog / yetki / iletişim kabiliyetleri
  ├── Extension Registry
  └── Engine Integration Contract
        ├── Motor A kendi şemasını ve iş kurallarını yönetir
        ├── Motor B kendi şemasını ve iş kurallarını yönetir
        └── Motor C kendi şemasını ve iş kurallarını yönetir
```

## 6.2 Uzantı kaydı (Extension Registry)

Çekirdek, gerektiğinde uzantıların varlığını tanıyabilecek asgari bir kayıt mekanizması sunmalıdır.

Uzantı tanımı şu metadata'yı içerebilir:

- Benzersiz uzantı kimliği.
- Ad ve açıklama.
- Sürüm.
- Sağlayıcı/sahip.
- Uzantı türü: katalog alanı sağlayıcısı, arayüz katkısı, motor adaptörü, politika sağlayıcısı vb. (V1'de yalnızca gerçekten ihtiyaç duyulan türler uygulanır).
- İhtiyaç duyduğu izin anahtarları.
- Kullandığı kaynak türleri ve olay şemaları.
- İlgili kategori kapsamı (opsiyonel; kategorinin iş mantığını çekirdeğe taşımaz).
- Veri namespace'i ve sürümlü şema referansı.
- Etkin/pasif durumu.
- Uyumluluk aralığı: desteklediği Core API/contract sürümü.

Uzantı kaydı, otomatik kod çalıştırma veya güvenilmeyen eklenti yükleme mekanizması anlamına gelmez. V1'de uzantılar yalnızca güvenilir, açıkça kaydedilmiş ve izinleri tanımlanmış entegrasyonlardır.

## 6.3 Motor bağlama sözleşmesi (Engine Integration Contract)

V1'de somut bir motor yazılmayacaktır. Yalnızca aşağıdaki ortak entegrasyon sözleşmesi belgelenmeli ve gerekli teknik arayüzleri oluşturulmalıdır.

Bir motor, bağlanabilmek için en azından:

1. Kendini benzersiz motor kimliği ve sürümüyle tanımlar.
2. Desteklediği yetenekleri deklaratif olarak bildirir; bu bildirim çekirdeğin motorun iş mantığını bildiği anlamına gelmez.
3. Çekirdek kaynaklarına yalnızca tanımlı API ve yetki kapsamları üzerinden erişir.
4. Çekirdek varlıklarını kendi veritabanına kopyalamak yerine kararlı referansları kullanır.
5. Kendi işlem kaynaklarını kendi bounded context'inde saklar.
6. Ortak olay zarfına uyar ve olay şemasını sürümler.
7. Aynı isteğin tekrar gönderilmesi halinde tutarlı davranmak için uygun idempotency/korelasyon mekanizması sağlar.
8. Gerekli olduğunda çekirdeğe yalnızca minimal özet veya kaynak bağlantısı verir.
9. Yetki, tenant izolasyonu, audit ve veri politikalarını atlayamaz.
10. Sürüm uyumluluğunu ve kaldırma/geçiş koşullarını bildirir.

### Soyut adaptör yüzeyi

Aşağıdaki maddeler kavramsal arayüzlerdir; hepsini gerçek bir dağıtık servis veya mesaj kuyruğu olarak uygulamak zorunlu değildir:

- `registerEngine(descriptor)` — motor kimliğini ve sürümünü kaydetme.
- `getEngineCapabilities(engineId)` — beyan edilmiş yetenekleri okuma.
- `resolveCoreReferences(context)` — motor için yetkili çekirdek referanslarını çözme.
- `authorizeEngineAction(actor, action, resource, scope)` — eylem yetkisini kontrol etme.
- `publishEngineEvent(envelope)` — sürümlü olay alma/yayınlama sözleşmesi.
- `linkEngineResource(binding)` — çekirdek kaydı ile motor kaynak referansını bağlama.
- `getEngineResourceSummary(reference)` — varsa motorun ortak ekranda gösterilebilecek özetini alma.
- `revokeEngineBinding(bindingId)` — entegrasyon erişimini kontrollü kapatma.

Bu isimler örnek sözleşme isimleridir; mevcut projenin dil ve isim standardına göre değiştirilebilir. **Hiçbirinin içinde rezervasyon oluştur, sipariş ver, teklif hesapla veya benzeri iş motoru metodu bulunmamalıdır.**

## 6.4 Motor uyumluluk testi

Gerçek bir motor uygulanmadan da sözleşme doğrulanabilir:

- Sözleşme şema testleri.
- Yetkisiz erişim testleri.
- Tenant izolasyonu testleri.
- Sürüm uyumluluk testleri.
- İdempotency ve tekrar gönderim testleri.
- Bozuk/eksik olay zarfı testleri.
- Bilinmeyen alan ve namespace izolasyonu testleri.
- Motor bağlantısını iptal etme testleri.

Testler, gerçek rezervasyon/sipariş/teklif iş akışı simüle etmemelidir. Gerekirse yalnızca sözleşme geçerliliğini sınayan sahte (mock) taşıyıcı/fixture kullanılabilir; bu bir motor sayılmaz ve iş kuralı içermemelidir.

---

# 7. ÇEKİRDEĞE ALINMAYACAKLAR — KESİN KAPSAM DIŞI

Aşağıdaki liste örnek niteliğindedir ancak sınırları bağlayıcıdır.

| Alan | V1 çekirdekte yapılmayacak | Gelecekte kimin sorumluluğu |
|---|---|---|
| Randevu | Takvim, slot, müsaitlik, rezervasyon, iptal kuralı | Randevu/rezervasyon motoru |
| E-ticaret/market | SKU, stok, sepet, sipariş, teslimat, iade | Ticaret/sipariş motoru |
| Teklif | Teklif kalemi, fiyat hesabı, pazarlık, kabul akışı | Teklif motoru |
| Web tasarım/yazılım projesi | Brief, kilometre taşı, görev, dosya teslimi, revizyon akışı | Proje motoru |
| Eğitim | Ders, sınıf, müfredat, devam, sınav, puan | Eğitim/LMS motoru |
| Fitness | Antrenman programı, ölçüm, seans planı, üyelik kullanımı | Fitness motoru |
| Etkinlik/organizasyon | Görev listesi, bütçe, tedarikçi planı, masa/konuk yerleşimi | Organizasyon motoru |
| Sağlık | Hasta dosyası, teşhis, tedavi, klinik karar, reçete | Yetkili sağlık sistemi/özel motor |
| Astroloji/kişisel danışmanlık | Doğum haritası hesaplama, kişiye özel rapor üretimi | İlgili dijital içerik/seans motoru |
| Ödeme | Tahsilat, cüzdan, komisyon, satıcı hakedişi, para transferi | Bağımsız ödeme entegrasyonu/ödeme motoru |
| Abonelik | Plan faturalaması, dönem, kota tüketimi, tahsilat | Abonelik/entitlement motoru |
| Lojistik | Rota, araç, kurye atama, canlı takip | Lojistik motoru |
| CRM/ERP | Satış hunisi, muhasebe, stok muhasebesi, bordro | Harici ürün veya bağımsız modül |
| Pazarlama otomasyonu | Kampanya motoru, kupon, dinamik promosyon | Pazarlama motoru |
| Yapay zekâ sektörel ajanı | Teşhis, fiyat kararı, otomatik teklif, sektör karar motoru | Ayrı AI/uzantı ürünü |

### 7.1 V1'de ayrıca yapılmayacak teknik çalışmalar

- Mikroservis mimarisine geçiş, yalnızca “ileride ölçeklenir” gerekçesiyle yapılmayacaktır.
- Gereksiz mesaj kuyruğu/event streaming platformu kurulmayacaktır.
- Genel amaçlı workflow engine geliştirilmeyecektir.
- Plugin marketplace veya güvenilmeyen dinamik kod yükleme sistemi yapılmayacaktır.
- Her sektör için ayrı dashboard/tema/form oluşturulmayacaktır.
- Henüz motor olmadığı halde motor tabloları, demo motorlar veya kullanılmayan modüller eklenmeyecektir.
- Her olası sektör alanını kapsamak için devasa JSON alanları veya sınırsız custom-field altyapısı oluşturulmayacaktır.
- Öngörülmemiş gelecekteki gereksinimler adına gereksiz soyutlama katmanları eklenmeyecektir.

---

# 8. VERİ MİMARİSİ VE SINIRLAR

## 8.1 Veri sahipliği kuralı

Her veri alanı için tek bir yetkili sahip (source of truth) belirlenmelidir.

- Hesap ve rol verisinin sahibi Core'dur.
- İşletme ve şube ortak profilinin sahibi Core'dur.
- Merkezi taksonominin sahibi Core'dur.
- Ortak katalog kaydının sahibi Core'dur.
- Motorun özel işlem verisinin sahibi ilgili motordur.
- Motor verisinin çekirdeğe kopyalanması yerine referans veya kontrollü özet kullanılmalıdır.

## 8.2 Uzantı verisi

Sektöre özel alan ihtiyacı ortaya çıkarsa:

1. Alanın gerçekten sektöre özgü olup olmadığı doğrulanır.
2. Alanın sahibi olacak uzantı/motor belirlenir.
3. Veri namespace'i ve şema sürümü tanımlanır.
4. Okuma/yazma yetkisi, saklama süresi ve hassasiyet seviyesi belirlenir.
5. Çekirdek yalnızca keşif veya profil görünürlüğü için gereken, güvenli ve açıkça seçilmiş özet alanı indeksler.
6. Çekirdeğin ana veri modeli sektör alanlarıyla kirletilmez.

Kontrolsüz `metadata` veya JSON alanı, her şeyi içine doldurmak için kullanılmamalıdır. Kullanılacaksa boyut, şema, erişim, indeks, doğrulama ve versiyon kuralları bulunmalıdır.

## 8.3 Tenant ve veri izolasyonu

- Her işletme sahipliğindeki kaynak açık tenant/sahiplik bağı taşımalıdır.
- Tenant sınırı API, servis katmanı ve veri erişiminde doğrulanmalıdır.
- Kullanıcı arayüzü filtreleri güvenlik kontrolü yerine geçmez.
- İşletmeler arası veri erişimi varsayılan olarak kapalıdır.
- Platform yöneticisi erişimi ayrıca yetkilendirilir ve denetlenir.
- Motor, çekirdeğin başka tenant'a ait verisini doğrudan sorgulayamaz.
- Ortak arama indeksinde de tenant/görünürlük kuralları uygulanmalıdır.

---

# 9. GÜVENLİK, GİZLİLİK VE UYUM GEREKSİNİMLERİ

V1 güvenlik tasarımı sonradan eklenecek bir özellik değil, çekirdek kapsamının parçasıdır.

Asgari gereksinimler:

- Güvenli kimlik doğrulama ve oturum yönetimi; mevcut güvenilir auth altyapısı varsa tekrar yazma.
- Sunucu tarafında rol + kaynak + kapsam bazlı yetkilendirme.
- Varsayılan ret ve en az yetki.
- İşletme/tenant izolasyonu.
- Hassas alanların loglarda maskelenmesi.
- Girdi doğrulama, çıktı kodlama ve yaygın web güvenlik risklerine karşı koruma.
- CSRF, XSS, injection, SSRF, IDOR/BOLA ve yetki yükseltme risklerinin değerlendirilmesi.
- Rate limiting ve kötüye kullanım önlemleri; uygulanma biçimi mevcut altyapıya göre seçilir.
- Dosya yükleme varsa tür, boyut, erişim ve zararlı içerik kontrolleri.
- Gizli anahtarların kaynak kodda tutulmaması.
- Yönetici ve moderatör eylemlerinin audit log'a yazılması.
- API ve entegrasyon kimlik bilgilerinin güvenli saklanması ve iptal edilebilmesi.
- Veri dışa aktarma/silme süreçlerinin doğrulanabilir ve yetkili olması.
- Yasal gereksinimlerin faaliyet gösterilen ülkeye göre ayrıca hukuk uzmanı tarafından doğrulanması.

### Hassas veri sınırı

V1 çekirdek, sağlık/finans/çocuk/biyometrik veri gibi özel nitelikli veya yüksek riskli verilerin ayrıntılı alan modelini oluşturmaz. Bir sektör motoru gelecekte böyle veri gerektirirse veri sınıflandırması, hukuki dayanak, açık yetki, saklama/anonimleştirme ve erişim kontrolleri ayrıca tasarlanmalıdır.

---

# 10. API, OLAY VE SÖZLEŞME STANDARTLARI

## 10.1 API ilkeleri

- API sözleşmeleri belgelenmeli ve sürümlenmelidir.
- İstemcinin gönderdiği sahiplik/tenant bilgisinin doğruluğu varsayılmamalıdır.
- Hata yanıtları tutarlı ve makine tarafından okunabilir olmalıdır.
- Liste uçlarında sayfalama, filtre ve sınırlar tanımlanmalıdır.
- İsteklerde korelasyon kimliği bulunmalıdır.
- Kritik yazma işlemlerinde uygun idempotency yaklaşımı kullanılmalıdır.
- API değişiklikleri geriye dönük uyumluluk politikasıyla yönetilmelidir.
- Motorların doğrudan iç veritabanına erişmesi yasaklanmalıdır.

## 10.2 Ortak olay zarfı

Kavramsal asgari alanlar:

```json
{
  "event_id": "unique-id",
  "event_name": "core.resource.changed",
  "schema_version": "1.0",
  "occurred_at": "ISO-8601 timestamp",
  "source": {
    "type": "core-or-registered-extension",
    "id": "stable-source-id",
    "version": "source-version"
  },
  "actor_ref": "optional-account-reference",
  "tenant_ref": "optional-tenant-reference",
  "resource_ref": {
    "type": "resource-type",
    "id": "stable-resource-id"
  },
  "correlation_id": "request-or-workflow-correlation-id",
  "data": {}
}
```

Bu yalnızca ortak zarf örneğidir. `data` alanı sınırsız ve şemasız veri çöplüğüne dönüştürülmemelidir. Her event kendi sürümlü şemasına sahip olmalıdır.

## 10.3 Olay sahipliği

- Core event'lerini Core tanımlar ve yayınlar.
- Motor event'lerini ilgili motor tanımlar ve yayınlar.
- Core, motor event'inin iş anlamını yeniden yorumlayıp kendi işlem kuralına dönüştürmez.
- Motorun çekirdeği ilgilendiren olayı doğrulanmış kaynak kimliği ve sözleşme üzerinden iletilir.
- Olay tesliminde tekrar, gecikme ve sıra garantileri açıkça belgelenmelidir; garanti edilemeyen davranış garanti edilmiş gibi sunulmamalıdır.

---

# 11. KULLANICI VE YÖNETİM DENEYİMİ

V1 yönetim arayüzü yalnızca çekirdek kabiliyetlerini yönetir.

## 11.1 İşletme paneli

Asgari bölümler:

- İşletme profili.
- Şube/lokasyon yönetimi.
- Ekip ve üyelik yönetimi.
- Ortak hizmet/sunum kataloğu.
- Kategori ve hizmet alanı seçimi.
- Gelen genel mesajlar/konuşmalar (iletişim yetkisi kapsamında).
- Yorumlar ve şikâyetlere yanıt.
- Hesap, gizlilik ve bildirim tercihleri.

Bu panelde randevu takvimi, sipariş listesi, stok, teklif panosu, ders yönetimi, proje görevleri veya ödeme hakediş ekranı bulunmayacaktır.

## 11.2 Platform yönetim paneli

Asgari bölümler:

- Hesap yönetimi.
- İşletme doğrulama ve yayın durumu.
- Merkezi taksonomi yönetimi.
- Rol/izin ve kapsam yönetimi.
- Şikâyet/itiraz/moderasyon kuyruğu.
- Uzantı/motor kayıtları ve sürüm uyumluluğu.
- Denetim kayıtlarına yetkili erişim.
- Ortak platform ölçümleri.
- Güvenlik ve sistem durumu için gerekli temel görünüm.

## 11.3 Arayüz sınırları

- Çekirdek arayüzleri sektör bağımsız olmalıdır.
- Sektör özel alanlarının çekirdek formuna gömülmesi yasaktır.
- Uzantılar kendi arayüzlerini sunabilir; fakat kimlik, yetki, gezinme ve tasarım entegrasyonu ortak sözleşmeyle yapılır.
- Erişilebilirlik, mobil kullanım ve okunabilirlik temel gereksinimdir.

---

# 12. V1 UYGULAMA AŞAMALARI

Aşamalar sıralıdır; her aşamanın kabul kriterleri sağlanmadan sonraki aşamaya geçilmemelidir.

## Aşama 0 — Mevcut proje keşfi ve güvenli başlangıç

- Depo ve dosya yapısını incele.
- Kullanılan dil/framework/veritabanı/ORM/auth sistemini belirle.
- Mevcut tablolar, API'ler, roller, testler ve güvenlik kontrollerini envanterle.
- Kırılma riski olan noktaları listele.
- PRD ile mevcut uygulama arasındaki farkları çıkar.
- Gereksiz teknoloji veya altyapı değişikliği yapma.

**Çıktı:** `BASELINE.md`, `GAP_ANALYSIS.md` veya mevcut dokümantasyon standardına uygun eşdeğeri.

## Aşama 1 — Alan sınırları ve çekirdek sözleşmeleri

- Core bounded context sınırlarını tanımla.
- Core ile motorların veri sahipliğini göster.
- Kavramsal varlıkları mevcut teknolojiye uygun veri modeline dönüştür.
- Yetki ve tenant modelini tanımla.
- Engine/extension descriptor ve entegrasyon sözleşmesini tanımla.
- API ve event sürümleme kurallarını yaz.

**Çıktı:** `CORE_ARCHITECTURE.md`, `DATA_OWNERSHIP.md`, `ENGINE_CONTRACT.md`, karar kayıtları.

## Aşama 2 — Temel çekirdek kabiliyetleri

Öncelik sırası:

1. Hesap/kimlik entegrasyonu.
2. Rol, izin ve kapsam.
3. İşletme/kurum ve sahiplik.
4. Şube/lokasyon ve ekip üyeliği.
5. Merkezi taksonomi.
6. Ortak katalog/sunum kaydı.
7. Bölge/hizmet alanı.

**Kabul:** Temel kaynaklar güvenli biçimde oluşturulabilir, güncellenebilir, yayımlanabilir ve doğru tenant sınırında okunabilir.

## Aşama 3 — Ortak etkileşim ve güven kabiliyetleri

- Arama/keşif sözleşmesi.
- Konuşma/mesaj altyapısı.
- Bildirim tercihleri ve kanal arayüzü.
- Yorum/değerlendirme.
- Şikâyet/itiraz ve moderasyon.
- Audit ve ortak event standardı.

**Kabul:** Kullanıcı ve işletme ortak platform işlevleriyle iletişim kurabilir; kritik eylemler izlenebilir; gizlilik sınırları korunur.

## Aşama 4 — Uzantı ve motor bağlantı hazırlığı

- Extension Registry.
- Engine descriptor ve sürüm uyumluluğu.
- EngineBinding/InteractionReference.
- Yetki kontrol sözleşmesi.
- Event envelope ve şema doğrulama.
- Sözleşme testleri.

**Kabul:** Gerçek bir motor geliştirmeden, bağımsız bir motorun hangi sözleşmeye uyarak bağlanacağı açık ve test edilebilir olmalıdır.

## Aşama 5 — Sertleştirme ve V1 kabulü

- Güvenlik testleri.
- Yetki yükseltme ve tenant izolasyonu testleri.
- Veri sızıntısı kontrolleri.
- API sözleşme testleri.
- Yük ve hata senaryoları için temel testler.
- Yedekleme/geri yükleme ve operasyon notları.
- Dokümantasyon ve çalıştırma talimatları.

**Kabul:** Çekirdek, hiçbir sektör işlem motoruna bağlı olmadan çalışır ve motor entegrasyonuna hazırdır.

---

# 13. V1 KABUL KRİTERLERİ (DEFINITION OF DONE)

V1 ancak aşağıdaki koşulların tamamı sağlandığında tamamlanmış sayılır:

## Ürün

- [ ] Hesap, işletme, şube, ekip, katalog ve taksonomi çekirdek varlıkları mevcuttur.
- [ ] Bir hesap birden fazla işletmeyle kontrollü biçimde ilişkilendirilebilir.
- [ ] İşletme ve şube görünürlük durumları merkezi olarak yönetilir.
- [ ] Katalog kaydı sektör özel alanlara zorunlu olmadan yayımlanabilir.
- [ ] Kategori ana kayıtları tekil ve tutarlıdır.
- [ ] Genel keşif/arama için belgelenmiş sözleşme vardır.
- [ ] Ortak mesajlaşma, bildirim ve değerlendirme kabiliyetleri tanımlı kapsamda çalışır.
- [ ] Moderasyon, şikâyet ve itiraz kayıtları denetlenebilir.

## Mimari

- [ ] Core ile motorların veri sahipliği belgelenmiştir.
- [ ] Core hiçbir işlem motorunun özel durum makinesine sahip değildir.
- [ ] Motor bağlantısı sürümlenmiş sözleşmeye dayanır.
- [ ] Motorlar Core veritabanına doğrudan erişmez.
- [ ] Sektöre özel alanlar çekirdek ana modeline gömülmemiştir.
- [ ] Gerçek motor veya demo işlem motoru üretilmemiştir.
- [ ] Entegrasyon sözleşmesi testlerle doğrulanmıştır.

## Güvenlik

- [ ] Sunucu tarafı rol/kapsam yetkilendirmesi vardır.
- [ ] Tenant izolasyonu test edilmiştir.
- [ ] IDOR/BOLA ve yetki yükseltme senaryoları test edilmiştir.
- [ ] Hassas veri loglara açık şekilde yazılmaz.
- [ ] Yönetim/moderasyon eylemleri audit kaydı üretir.
- [ ] API giriş doğrulaması ve kötüye kullanım kontrolleri vardır.

## Operasyon

- [ ] Kurulum, yapılandırma ve çalıştırma belgeleri günceldir.
- [ ] Testler tekrarlanabilir biçimde çalışır.
- [ ] Hata ve eksiklerin listesi vardır.
- [ ] Yedekleme/geri yükleme yaklaşımı belgelenmiştir.
- [ ] API/contract sürüm politikası belgelenmiştir.

---

# 14. TEST STRATEJİSİ

Testler yalnızca mutlu yol senaryolarından oluşmamalıdır.

Asgari test grupları:

1. **Unit tests:** Alan kuralları, değer doğrulama ve ortak servisler.
2. **Integration tests:** Kimlik, veri erişimi, bildirim ve audit entegrasyonları.
3. **Authorization tests:** Rol, izin, kapsam ve kaynak sahipliği.
4. **Tenant isolation tests:** İşletmeler arası veri sızıntısı.
5. **API contract tests:** Sözleşme alanları ve sürüm uyumluluğu.
6. **Extension contract tests:** Uzantı kimliği, izinleri, namespace ve sürüm kontrolü.
7. **Event tests:** Şema, kaynak doğrulaması, tekrar gönderim ve korelasyon.
8. **Abuse/security tests:** Rate limit, IDOR/BOLA, girdi manipülasyonu ve yetki aşımı.
9. **Regression tests:** Mevcut çalışan işlevlerin bozulmaması.

Gerçek sektör motoru olmadığı için randevu, sipariş, ders, teklif veya ödeme iş akışına ilişkin testler yazılmayacaktır.

---

# 15. ÖLÇÜM VE BAŞARI GÖSTERGELERİ

Çekirdek başarısı, kaç sektör modülü yazıldığıyla ölçülmez.

V1 için önerilen teknik/ürün göstergeleri:

- Çekirdek kaynaklarının ne kadarının ortak sözleşmeler üzerinden kullanılabildiği.
- Yetkisiz erişim ve tenant izolasyonu testlerinin başarı oranı.
- API sözleşme uyumluluğu.
- Çekirdekte sektör özel koşul/alan sayısının düşük tutulması.
- Yeni bir motorun çekirdek ana veri modelini değiştirmeden bağlanabilmesi.
- Bir hesabın farklı işletme ve roller arasında güvenli çalışabilmesi.
- İşletme ve katalog kayıtlarının eksiksiz, doğrulanmış ve arama için uygun olması.
- Audit kapsamındaki kritik eylemlerin izlenebilirliği.

**Önemli:** V1'de gerçek motor geliştirilmediği için “motor entegrasyonu tamamlandı” iddiasında bulunma. Yalnızca “entegrasyon sözleşmesi hazırlandı ve sözleşme testleri geçti” denebilir.

---

# 16. RİSKLER VE ÖNLEMLER

| Risk | Sonuç | Önlem |
|---|---|---|
| Çekirdeğe sektör mantığı sızması | Her yeni sektör çekirdeği karmaşıklaştırır | Bounded context ve kesin kapsam dışı listesi |
| Aşırı genel veri modeli | Anlamsız, doğrulanamayan veri yapısı | Yalın ortak alanlar + sürümlü uzantı şemaları |
| Her şeyi tek işlem modeline bağlamak | Randevu, sipariş ve teklif çakışır | İşlem durumları ve kuralları motor sahipliğinde |
| Motorların Core DB'ye doğrudan erişmesi | Sıkı bağımlılık ve güvenlik açığı | Yalnızca API/contract üzerinden erişim |
| Kapsamı erken büyütmek | V1 gecikir, bakım maliyeti artar | Aşamalı kabul kriterleri ve out-of-scope disiplini |
| Aşırı mikroservisleştirme | Operasyon ve geliştirme yükü artar | Önce mantıksal modüler sınırlar, gerektiğinde fiziksel ayrıştırma |
| Belirsiz rol modeli | Veri sızıntısı ve yetki hataları | Rol + kaynak + scope tabanlı merkezi politika |
| Hassas verinin çekirdeğe alınması | Mevzuat ve veri ihlali riski | Veri minimizasyonu ve özel uzantı sınırı |
| Sektör verisinin arama indeksine taşması | Gizlilik ve tutarsızlık | Yalnızca izinli ortak özet alanları indeksleme |
| Motor sözleşmesinin sürümsüz kalması | Entegrasyon kırılması | Şema sürümleme ve contract testleri |

---

# 17. GELECEK MOTORLAR İÇİN UYUMLULUK HEDEFİ

Çekirdek gelecekte aşağıdaki motorlarla entegre olabilecek biçimde tasarlanmalıdır. Bu liste V1 geliştirme kapsamı değildir.

| Gelecekteki motor | Motorun kendi alanı | Core'dan kullanacağı ortak kabiliyet |
|---|---|---|
| Randevu / rezervasyon | Takvim, slot, müsaitlik, rezervasyon, iptal | Hesap, işletme, uzman, şube, katalog, bildirim, mesaj |
| Ticaret / market | SKU, varyant, stok, sepet, sipariş, teslimat | Hesap, işletme, katalog referansı, konum, mesaj, yorum |
| Teklif / profesyonel hizmet | Brief, teklif kalemi, fiyat, kabul | Hesap, işletme, katalog, konuşma, dosya referansı, bildirim |
| Proje / üretim | Görev, kilometre taşı, teslim, revizyon | Hesap, ekip, işletme, iletişim, dosya referansı |
| Eğitim | Ders, sınıf, içerik, devam, sınav | Hesap, kurum, eğitmen, katalog, yetki, bildirim |
| Fitness | Program, seans, ölçüm, üyelik kullanımı | Hesap, işletme, uzman, katalog, iletişim |
| Etkinlik / organizasyon | Etkinlik, görev, bütçe, katılımcı akışı | Hesap, işletme, katalog, ekip, mesaj, bildirim |
| Dijital içerik / astroloji | İçerik üretimi, seans, rapor, erişim hakkı | Hesap, işletme, katalog, ödeme/erişim entegrasyon referansı |
| Sağlık hizmetleri | Yetkili ve mevzuata tabi klinik süreçler | Yalnızca hukuken ve teknik olarak güvenli ortak kimlik/kurum/yetki entegrasyonu |
| Abonelik / hak yönetimi | Plan, dönem, kota, entitlement, faturalama | Hesap, işletme, ürün/hizmet referansı, bildirim |
| Lojistik | Atama, rota, teslim, takip | İşletme, lokasyon, hesap, bildirim |

Uyumluluk, tüm motorların aynı veritabanını veya aynı işlem tablosunu paylaşması anlamına gelmez. Uyumluluk; **kimlik, yetki, referans, API, olay ve yönetişim sözleşmelerinin ortak olması** anlamına gelir.

---

# 18. KODLAMA AJANININ TESLİM ETMESİ GEREKENLER

Mevcut proje yapısına uygun biçimde aşağıdaki çıktıları hazırla. Dosya adları mevcut repo standardına uyarlanabilir.

## Dokümantasyon

- Core mimari ve bounded context açıklaması.
- Core / motor sorumluluk matrisi.
- Veri sahipliği haritası.
- Kavramsal veri modeli/ER diyagramı.
- API sözleşmeleri.
- Engine/extension entegrasyon sözleşmesi.
- Yetki ve tenant izolasyonu matrisi.
- Güvenlik ve veri koruma notları.
- Kurulum, yapılandırma ve test talimatı.
- Mimari karar kayıtları (ADR).

## Yazılım

- Yalnızca çekirdek kapsamındaki gerekli modüller.
- Merkezi ve test edilebilir yetki kontrolü.
- Sektörden bağımsız veri modelleri.
- Sürümlenmiş API/contract yapısı.
- Extension/engine kayıt ve referans bağlama altyapısı (yalnızca gerekli minimum kapsam).
- Ortak olay zarfı ve doğrulaması.
- Audit, gizlilik ve tenant izolasyonu kontrolleri.
- Otomatik testler.

## Kesinlikle teslim edilmeyecekler

- Çalışan herhangi bir işlem motoru.
- Motorlara ait tablo, iş akışı, durum makinesi veya sektör formu.
- Kullanılmayan örnek özellikler.
- Sırf gelecekte gerekebilir diye eklenmiş altyapı.

---

# 19. SON UYGULAMA KOMUTU

Şimdi mevcut projeyi incele ve bu PRD'yi temel alarak **yalnızca bağımsız Hizmet Katmanı Çekirdeği V1'i** geliştir.

Öncelik sırası:

1. Mevcut sistemi ve çalışan özellikleri koru.
2. Çekirdek ile işlem motorları arasındaki sınırı uygula.
3. Hesap, yetki, işletme, şube, ekip, taksonomi ve ortak katalog altyapısını kur.
4. Ortak keşif, iletişim, bildirim, değerlendirme ve yönetişim kabiliyetlerini tanımlı kapsamda tamamla.
5. Sürümlenmiş motor/uzantı sözleşmesini hazırla.
6. Güvenlik, tenant izolasyonu ve sözleşme testlerini tamamla.
7. Dokümantasyonu ve test sonuçlarını teslim et.

Her yeni tablo, alan, servis, bağımlılık veya altyapı bileşeni için şu soruyu sor:

> “Bu gerçekten bütün sektörlerin ortak ihtiyacı mı, yoksa belirli bir işlem motorunun ihtiyacı mı?”

Yanıt belirli bir motorun ihtiyacıysa **çekirdeğe ekleme**.

**Nihai hedef:** İşletmenin kim olduğunu, hangi ortak hizmetleri sunduğunu, nerede faaliyet gösterdiğini, kimlerin hangi yetkiyle hareket ettiğini ve müşterinin işletmeyle nasıl iletişim kurduğunu standartlaştıran; fakat rezervasyon, sipariş, teklif, ders, proje, ödeme veya başka bir sektörel işlemi kendisi yürütmeyen; gelecekteki tüm bu motorlara sürümlenmiş sözleşmelerle bağlanabilen yalın, güvenli ve bağımsız bir çekirdek.
