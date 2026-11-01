# Doğrulama Kayıtları

> Kural: Yalnızca gerçekten çalıştırılan testleri yaz. Kanıt dosyaları değişirse `check` kaydı eskimiş sayar.
> Kanıt hash'i: `node scripts/ai-memory.mjs evidence <dosya...>`

| ID | Tarih | Kontrol | Komut | Sonuç | Kanıt dosyaları | Kanıt hash |
|---|---|---|---|---|---|---|
| V-001 | 2026-10-23 | Hafıza sistemi otomatik testleri (envanter, artımlı tarama, değişiklik/taşıma, bozuk indeks, gizli dosya, durum, eski kanıt, mtime gizleme) | `node scripts/test-memory.mjs` | PASS — 25/25 | `scripts/ai-memory.mjs`<br>`scripts/test-memory.mjs` | `e3ea443867b5ee03` |
| V-002 | 2026-10-23 | Monorepo lint, typecheck, birim testleri ve üretim build'i | `npm run lint && npm run typecheck && npm run test && npm run build` | PASS — 3 test, 0 hata | `package.json`<br>`tsconfig.base.json`<br>`eslint.config.mjs`<br>`scripts/dev.mjs`<br>`scripts/setup.mjs`<br>`apps/api/package.json`<br>`apps/api/src/main.ts`<br>`apps/api/src/app.module.ts`<br>`apps/api/src/health/health.controller.ts`<br>`apps/web/package.json`<br>`apps/web/next.config.ts`<br>`apps/web/src/app/layout.tsx`<br>`apps/web/src/app/page.tsx`<br>`packages/shared/src/index.ts`<br>`packages/ui/src/index.ts`<br>`infra/docker-compose.yml`<br>`.env.example`<br>`README.md` | `569ef1242291242f` |
| V-003 | 2026-10-23 | Yerel altyapı + dev smoke: PostgreSQL, Redis, RustFS/S3 healthy; API `/health` ve web ana sayfası 200 | `npm run infra:up && npm run dev` | PASS — PG/Redis/S3 healthy, API 200, web 200 | `infra/docker-compose.yml`<br>`apps/api/src/health/health.controller.ts`<br>`scripts/dev.mjs`<br>`apps/web/src/app/page.tsx` | `f136700a87f5122c` |
