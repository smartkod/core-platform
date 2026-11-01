# Hafıza Sistemi Mimarisi

Bu dosya yalnızca yeni kurulan AI hafıza altyapısını anlatır. Ürün mimarisi için `doc/00` ve `doc/02` geçerlidir.

## Bileşenler

| Bileşen | Görev |
|---|---|
| `AGENTS.md` | OpenCode'un otomatik yüklediği kalıcı kurallar ve başlangıç/bitiş protokolü |
| `doc/03`, `doc/04` | İnsan tarafından okunabilir canlı durum; tek kaynak |
| `.ai/FILE_INDEX.json` | Üretilen SHA-256 dosya envanteri (makine okur) |
| `.ai/SESSION_START.md` | Üretilen oturum brifingi; değişiklik + son durum + test tazeliği |
| `.ai/VERIFICATION.md` | Test kayıtları ve kanıt dosyalarının hash'i |
| `scripts/ai-memory.mjs` | Artımlı tarama, indeks, brifing, denetim (sıfır bağımlılık) |
| `.opencode/plugins/project-memory.ts` | Proje açılışında `start` komutunu otomatik çalıştırır |
| `.opencode/commands/resume.md` | `/resume`: hafızayı yükleyip devam eder |
| `.opencode/commands/handoff.md` | `/handoff`: oturum sonu kaydını ve test kanıtını günceller |

## Veri akışı

1. Açılışta eklenti `ai-memory.mjs start --quiet` çalıştırır.
2. Script her dosyayı `stat` eder. `size + mtime` indeksle aynıysa içerik okunmadan atlanır; farklıysa SHA-256 hesaplanır.
3. Eski/yeni hash'ler karşılaştırılır; aynı hash'e sahip silinmiş+yeni dosya "taşınmış" sayılır.
4. İndeks atomik yazılır (`.tmp` + rename). Bozuk/eski indeks `update` ile yeniden üretilir.
5. `.ai/SESSION_START.md`; değişiklikleri, `doc/04` son durumunu ve test kanıtı tazeliğini birleştirir.
6. AI yalnızca raporlanan değişiklikleri ve görev bağımlılıklarını okur.

## Doğrulama modeli

- `check` indeksi **tam hash taramasıyla** karşılaştırır; böylece `mtime` korunarak yapılan içerik değişikliği yakalanır.
- `VERIFICATION.md` satırlarındaki kanıt dosyaları yeniden hash'lenir; kod değişmişse kayıt "eskimiş" sayılır ve `check` başarısız olur.
- Durum alanları (`SON İŞ`, `SIRADAKİ İŞ`, `ENGEL`) boşsa denetim başarısız olur.
- Testler `scripts/test-memory.mjs` içinde geçici bir fixture üzerinde çalışır; gerçek proje dosyalarını değiştirmez.

## Sınırlamalar

- Git deposu yoksa değişiklik takibi yalnızca dosya hash'ine dayanır. Git eklenirse `git status` ek kanıt olarak kullanılmalıdır.
- OpenCode'un güvenilir bir "oturum kapandı" semantik hook'u yoktur; oturum sonu kaydı `AGENTS.md` + `/handoff` ile yapılır. Eklenti yalnızca dosya indeksini tazeler.
- İndeks `.gitignore` dışındadır; başka makinede yeniden üretilir. `doc/` ve `.ai/VERIFICATION.md` kalıcı kayıttır.
- 5 MB üzeri dosyalar ve gizli/üretilmiş klasörler indekslenmez.
