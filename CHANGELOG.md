# Changelog

All notable changes are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0]

### Changed - the platform is now a sector-independent service-layer core

The previous AI (L3) and economy (L2) modules were removed. The core no longer
contains a wallet, an order machine, an AI gateway or any other sector engine.
What remains is the part every business application needs before it can run a
transaction: identity, authorization, organization, catalogue.

- Levels are now `L0` (kernel), `L1` (identity, authorization, files,
  notifications) and `L2` (organization, catalogue). `L3` is gone.
- `contract/core` defines the new module surface: `Authorizer`, `Organizations`
  and `Catalog`, with provider interfaces so modules never import each other.
- `cmd/depcheck` now enforces the boundary between the six core modules rather
  than an L1/L2/L3 hierarchy that no longer exists.

### Added - authorization (`mod/authz`)

- Roles are sets of permissions; an assignment binds one role to one user in one
  scope (`platform`, `organization`, `branch`, `self`). Nothing is global by
  accident, and the default answer is denial.
- `role_assignments` keeps revoked rows for audit, with a partial unique index
  so a double grant cannot create two indistinguishable active rows.
- Seeded system roles (`platform_admin`, `moderator`, `owner`, `manager`,
  `staff`, `member`) and the core permission vocabulary.
- `Require(action, scope)` middleware reads the tenant from the route
  (`:org_id`, `:branch_id`, or the session user for `self`); it never trusts a
  body-supplied scope id.

### Added - organization (`mod/organization`)

- Organizations, branches, memberships, regions and service areas.
- Status machine `draft → pending_verification → active → suspended → closed`,
  with an append-only status-event audit row written in the same transaction.
- Ownership transfer is an explicit operation that refuses to leave an
  organization without an administrator.
- Invitations are addressed to an e-mail and can be accepted only by an account
  with that address.

### Added - catalogue (`mod/catalog`)

- Central taxonomy (`taxonomy_nodes`) is the only authoritative category source.
- Offerings and variants carry only shared fields. `extension_ref` lets an
  engine attach its own record; the core never reads it.
- Offering status machine `draft → published → archived`; publishing and
  deleting a live offering are separate, auditable actions.
- Category browsing includes descendants, so browsing a parent category is not
  empty.

### Added - identity additions

- `phone`, `phone_verified_at` and `closed_at` on `users`; `moderator` platform
  role; optional-credential middleware for public discovery endpoints.

### Removed

- `mod/economy`, `contract/economy`, `mod/ai`, `platform/ai`, `cmd/aicheck`,
  their config blocks, migration history and tests.
- `web/`, the stale project memory folder, duplicated skills, and the old
  level documents (`references/L1.md`, `L2.md`, `L3.md`, `PRD.md`, `DENEME.md`).

## [0.1.0]

Initial kernel: configuration, PostgreSQL, embedded migrations, the HTTP
envelope, health endpoints, the module manager and the scaffolder — plus the
first modules (identity, files, notifications).
