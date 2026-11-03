# Motor Sözleşmesi (Engine Contract)

Bu belge bir **sektör motorunun** (randevu, sipariş, teklif, ders, proje, ödeme…)
çekirdeğe nasıl bağlanacağını tanımlar. Motor ayrı bir depodur, ayrı sürümlenir
ve ayrı deploy edilir.

Amaç: çekirdek motorun ne yaptığını bilmeden, motorun çekirdeğin kimlik, yetki,
işletme ve katalog verisini güvenle kullanabilmesi.

---

## 1. Sözleşme yüzeyi

Motor yalnızca şunları import eder:

| Paket | Ne verir |
|---|---|
| `contract/core` | `Module`, `Deps`, `Authorizer`, `Organizations`, `Catalog`, `Identity`, `Notifications`, tipler, sentinel hatalar |
| `platform` | `platform.Run` (yalnızca uygulamanın `main` paketinde) |

Motor `mod/identity`, `mod/authz`, `mod/organization` veya `mod/catalog`
paketlerini **import etmez**. İhtiyaç duyduğu her şey `core.Deps` alanlarındadır.

```go
func (m *Engine) Start(ctx context.Context, d core.Deps) error {
    if d.Authorizer == nil || d.Organizations == nil || d.Catalog == nil {
        return fmt.Errorf("engine: core services missing")
    }
    ...
}
```

## 2. Sürümleme

- Çekirdek `contract/core` yüzeyi **semver** ile yönetilir.
- Alan eklemek minor; alan kaldırmak veya imza değiştirmek major'dır.
- Motor, `go.mod` içinde çekirdeğin belirli bir major sürümüne bağlanır.
- Bir motor sürümü, desteklediği çekirdek aralığını `README` veya
  `engine.yaml` benzeri bir manifestte ilan eder.
- Çekirdek, motorun iç tiplerini bilmez; motor da çekirdeğin internal
  paketlerini bilmez.

## 3. Modül iskeleti

```go
package booking

import (
    "context"

    "github.com/labstack/echo/v4"
    "gorm.io/gorm"

    "core-platform/contract/core"
)

type Module struct{ /* ... */ }

func New() *Module { return &Module{} }

func (m *Module) Name() string           { return "booking" }
func (m *Module) Version() string        { return "0.1.0" }
func (m *Module) Dependencies() []string { return []string{"identity", "authz", "organization", "catalog"} }

func (m *Module) MigrationSource() core.MigrationSource {
    return core.MigrationSource{Schema: "booking", FS: migrationsFS, Dir: "."}
}

func (m *Module) Migrate(context.Context, *gorm.DB) error { return nil }

func (m *Module) Start(ctx context.Context, d core.Deps) error { /* ... */ return nil }
func (m *Module) Stop(context.Context) error { return nil }

func (m *Module) RegisterRoutes(g *echo.Group, d core.Deps) {
    r := g.Group("/ext/booking", d.Identity.Middleware())
    r.POST("/appointments", m.create, d.Authorizer.Require("booking.appointment.create", core.ScopeOrganization))
}
```

Uygulamanın `main.go` dosyası moturu listeye ekler:

```go
Modules: []core.Module{
    identity.New(),
    authz.New(),
    files.New(),
    notifications.New(),
    organization.New(),
    catalog.New(),
    booking.New(), // motor
},
```

## 4. Ayrı şema kuralı

Motor kendi PostgreSQL şemasını getirir:

```go
core.MigrationSource{Schema: "booking", FS: fs, Dir: "."}
```

- Çekirdek şemasına (`public`) migration yazmaz.
- Kendi `booking.schema_migrations` geçmişine sahiptir; çekirdek sürümünden
  bağımsız ilerler.
- Motor tabloları çekirdek kimliklerine **yabancı anahtar** koyamaz (şemalar
  arası FK yoktur). Bunun yerine `organization_id uuid NOT NULL` ve
  `user_id uuid NOT NULL` sütunlarını tutar; bütünlüğü uygulama katmanında
  doğrular.
- Silme davranışı motorun sorumluluğundadır: bir organizasyon kapatıldığında
  çekirdek motorun satırlarını silmez.

## 5. Yetki

### 5.1 İzin kaydı

Motor, izinlerini başlangıçta kaydeder:

```go
for _, p := range []core.Permission{
    {Key: "booking.appointment.read",    Scope: core.ScopeOrganization, Description: "..."},
    {Key: "booking.appointment.create",  Scope: core.ScopeOrganization, Description: "..."},
    {Key: "booking.calendar.manage",     Scope: core.ScopeOrganization, Description: "..."},
} {
    if err := d.Authorizer.RegisterPermission(ctx, p); err != nil { return err }
}
```

Anahtar kuralı: `<motor>.<kaynak>.<eylem>`. Çekirdek anahtarlarıyla çakışmaz.

### 5.2 Rol genişletme

Motor kendi rolünü tanımlayabilir veya mevcut bir role izin eklemek için yeni
bir rol tanımlayıp atama isteyebilir:

```go
d.Authorizer.UpsertRole(ctx, core.Role{
    Key:         "booking_manager",
    Name:        "Booking manager",
    Permissions: []string{"booking.appointment.read", "booking.appointment.create"},
})
```

Sistem rollerini (`owner`, `manager`, …) değiştirmek çekirdeğin kararıdır;
motor yalnızca kendi ad alanındaki rolleri yönetir.

### 5.3 Kontrol

```go
err := d.Authorizer.Authorize(ctx, core.Check{
    UserID:  principal.UserID,
    Action:  "booking.appointment.create",
    Scope:   core.ScopeOrganization,
    ScopeID: orgID, // sunucuda çözülen tenant
})
```

Kurallar:

- `orgID` istek gövdesinden alınmaz.
- Önce ilgili kayıt (ör. şube, sunum) yüklenir, sonra kapsam ondan türetilir.
- `Allowed` yalnızca UI ipucu içindir; korunan işlem `Authorize` kullanır.

## 6. İşletme ve katalog kullanımı

### 6.1 İşletme

```go
org, err := d.Organizations.ByID(ctx, orgID)
if err != nil { return err }
if org.Status != core.OrgStatusActive {
    return core.ErrInvalidState
}
```

- Motor, organizasyonun durumunu kendi tablosuna kopyalamaz; her istekte
  çekirdekten okur veya kendi önbelleğini kısa tutar.
- Şubeler `d.Organizations.Branches(ctx, orgID)` ile alınır.
- Üyelik doğrulaması `MembershipFor` ile yapılabilir, ancak yetki kararı yine
  `Authorizer`'a aittir.

### 6.2 Sunum bağı

Motor, kendi kaydını çekirdekteki sunuma **referansla** bağlar:

```go
offering, err := d.Catalog.CreateOffering(ctx, core.OfferingInput{
    OrganizationID: orgID,
    TaxonomyID:     taxonomyID,
    Name:           "60 dakikalık seansta",
    Slug:           "60-dk-seans",
    ServiceMode:    core.ServiceModeOnSite,
    ExtensionRef: &core.ExtensionRef{
        ExtensionID:  "booking",
        ResourceType: "service",
        ResourceID:   serviceID, // motorun kendi kimliği
    },
})
```

- Çekirdek `ExtensionRef`'i saklar, **okumaz** ve doğrulamaz.
- Motor, `offering.ID` ile kendi kaydı arasındaki bağı kendi şemasında tutar.
- Sunum silinirse/arşivlenirse motorun ne yapacağı motorun kararıdır; çekirdek
  cascade uygulamaz.

### 6.3 Kategori

Motor yeni kategori ağacı tanımlamaz. Kategori gerekiyorsa çekirdeğin
taksonomisine düğüm eklenir (`CreateTaxonomyNode`, platform yetkisi) veya
mevcut düğüm kullanılır. Serbest metin kategori motor tarafında da üretilmez.

## 7. Bildirim

Motor doğrudan e-posta göndermez:

```go
d.Notifications.Enqueue(ctx, core.NotificationInput{...})
```

Outbox, yeniden deneme ve tercih kontrolü çekirdeğin sorumluluğundadır.

## 8. Tenant yaşam döngüsü

| Olay | Çekirdek davranışı | Motordan beklenen |
|---|---|---|
| Organizasyon `suspended` | Yeni tenant işlemi yetkilendirilmez | Motor yeni kayıt açmaz |
| Organizasyon `closed` | Atamalar iptal edilebilir | Motor kendi kayıtlarını arşivler/saklar |
| Üyelik `removed` | İlgili atama iptal edilir | Motor kullanıcıyı kendi tarafında pasifleştirir |
| Sahiplik devri | `owner` ataması el değiştirir | Motor özel bir şey yapmaz |

Çekirdek, motorun verisini silmez; bu bilinçli bir karardır (ADR-009).

## 9. Hata ve tutarlılık

- Motor, çekirdeğin sentinel hatalarını kullanır (`core.ErrNotFound`,
  `core.ErrForbidden`, `core.ErrValidation`, `core.ErrInvalidState`).
- Kendi iç hatalarını `core.ErrProviderFailure` ile sarabilir; 5xx gövdesi
  çekirdek tarafından zaten gizlenir.
- Motorun kendi durum makinesi varsa koşullu `UPDATE` kullanmalıdır; çekirdek
  bu konuda bir yardımcı sağlamaz.

## 10. Test ve kabul

Bir motor birleştiğinde CI en az şunları doğrular:

1. `go build ./...`
2. `go run ./cmd/depcheck` — motor `mod/*` import etmiyor.
3. Motorun migration'ları boş bir veritabanına uygulanıyor.
4. Motor rotaları kimlik ve yetki middleware'i olmadan açılmıyor.
5. Tenant A'nın kimliğiyle tenant B'nin kaydı istenince 404/403 dönüyor.

## 11. Sürüm geçişi

- Çekirdek major sürüm atladığında motor `go.mod`'unu güncellemeden derlenmez.
  Bu istenen davranıştır: sessiz kırılma yerine derleme hatası.
- Çekirdek, kullanımdan kaldırdığı alanı en az bir minor sürüm boyunca tutar ve
  `Deprecated:` yorumuyla işaretler.
- Motor, çekirdeğin internal paketlerine veya tablolarına bağımlı olmadığı için
  çekirdek şema değişikliklerinden etkilenmez.
