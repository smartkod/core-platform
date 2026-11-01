# ARCHITECTURE

Status: ACTIVE
Architecture style: MODULAR MONOLITH

## 1. System boundaries

### Presentation
Next.js / React UI.

Responsibilities:
- rendering
- interaction
- presentation state
- calling approved application interfaces

Must not contain:
- payment authority
- authorization decisions
- database access
- secret handling
- duplicated business rules

### Application
Use cases / application services / server actions or approved API handlers.

Responsibilities:
- coordinate use cases
- authorize operations
- validate inputs
- call domain and infrastructure boundaries

### Domain
Business rules and invariants.

Responsibilities:
- business decisions
- state transitions
- domain validation

Must not depend on React or browser APIs.

### Infrastructure
Database, Stripe, email, storage, external APIs.

Responsibilities:
- provider-specific implementation
- persistence
- external communication

Provider details must not leak into unrelated domain code.

## 2. Feature boundaries

Prefer feature-oriented organization:

src/
  app/
  components/
  features/
    auth/
    users/
    billing/
    subscriptions/
    dashboard/
  lib/
  server/
  types/

A feature owns its business logic, schemas, queries/actions, and feature-specific UI.

Cross-feature access must use an explicit public interface. Do not import another feature's private internals.

## 3. Dependency direction

UI
↓
Application
↓
Domain
↓
Infrastructure

Infrastructure must not define business policy.

Domain must remain independent from framework/provider details.

## 4. Default architecture decisions

- TypeScript strict mode.
- PostgreSQL relational model.
- Migration-based schema changes.
- Server-side secrets only.
- Zod or equivalent boundary validation.
- Modular monolith first.
- No microservices unless justified by a documented ADR.
- No event bus unless a concrete requirement exists.
- No Redis unless a concrete workload requires it.
- No queue unless asynchronous processing requires it.
- No abstraction merely for theoretical future providers.

## 5. Source of truth

Each important state must have one declared source of truth.

Examples:
- payment/subscription lifecycle: Stripe for Stripe-owned payment state
- application authorization: application database
- user-facing product entitlement: application database, derived from verified billing state
- UI state: client state only when appropriate

## 6. Change rule

Any architectural change must:
1. identify the current design;
2. explain why it is insufficient;
3. document alternatives;
4. record the decision in DECISIONS.md;
5. update affected documentation.

## 7. Non-goals

Do not optimize for:
- maximum abstraction
- maximum number of agents
- maximum number of packages
- premature microservices
- speculative scalability
- replacing working code for stylistic preference
