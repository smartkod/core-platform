
Özellikle Auth + PostgreSQL + Stripe + Next.js için “golden path” oluşturursak, bundan sonra yeni proje açarken AI'a yalnızca:

# “Yeni proje oluştur: [ürünün amacı]”dememiz yeterli olabilir. Agent geri kalan mimari başlangıcını bu sözleşmelere göre hazırlayabilir.

# AI Project Starter

Professional governance starter for AI-assisted applications built around:
- Next.js / React / TypeScript
- PostgreSQL
- Stripe
- Zod
- automated tests

## Core principle

The human states the desired outcome. Agents inspect the existing system, plan the smallest safe change, implement it, test it, and validate architectural compliance.

## Operating loop

REQUEST
→ ARCHIFY
→ IMPACT ANALYSIS
→ CHANGE PLAN
→ IMPLEMENT
→ TEST
→ ARCHITECTURE VALIDATION
→ DONE

## Architecture default

Start as a modular monolith. Introduce services only when a concrete requirement justifies them.

## First files to read

1. AGENTS.md
2. docs/ARCHITECTURE.md
3. docs/CONTRACT.md
4. docs/SECURITY.md
5. docs/DATABASE.md
6. docs/BILLING.md
7. docs/DECISIONS.md
