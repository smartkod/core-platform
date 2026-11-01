# Changelog

All notable changes are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added - L2 economy: wallet ledger + rewarded ads

- `contract/economy`: the public L2 surface (`Wallet`, `Ads`, sentinel errors,
  reason codes). Applications reach it through `core.Deps.Economy`, so they
  never import the module package; `core.EconomyProvider` is wired by the
  bootstrap like the L1 providers.
- `mod/economy`: append-only token ledger. Balances move with a single
  conditional `UPDATE ... WHERE balance + delta >= 0`; a trigger blocks
  `UPDATE`/`DELETE` on `wallet_entries`, so the audit trail cannot be rewritten.
  `SUM(delta) == balance` is asserted by the tests.
- Welcome grant (10 tokens by default) is applied on the first wallet view and
  keyed `signup_grant:<userID>`, so it pays exactly once.
- Rewarded ads: `Pick` -> server-stamped `StartView` -> `ClaimView`. The
  required duration and both stamps are server facts; claiming early returns
  `REWARD_TOO_EARLY` and the client cannot influence the amount.
- One earning session per `(user, ad, day)` enforced by a unique index, so
  watching one creative repeatedly is structurally impossible.
- Watch and click rewards share one earn pool; caps default to 3/day and
  10/month over UTC boundaries and are read from `reward_settings`.
- Admin surface: ads CRUD (video/image/text), affiliate URLs restricted to
  https, per-ad stats, and the reward dials with `updated_by` audit.
  There is deliberately no user-facing credit/debit endpoint; admins move
  tokens through `/economy/admin/wallets/:user_id/{credit,debit}` with a fixed
  reason code.
- `mod/economy/migrations/0001_economy_init.sql` also carries the catalog,
  orders, payments, payment events, entitlements and refunds tables for the
  next increment (no service code yet).
- `cmd/server` now boots at L2 so the module runs end to end.

### Added — AI provider client (prep for L3)

- `platform/ai`: OpenAI-compatible chat client behind a one-method `Provider`
  interface. Base URL, model, timeout and retries are configuration; the key is
  read from `AI_API_KEY` only and has no YAML field, so it cannot be committed.
- Retries apply to 5xx, 429 and network failures. A 4xx fails immediately:
  repeating a bad request wastes the caller's latency budget and hides the bug.
- `HTTPError` carries the status and the provider's own message, so "invalid
  model name" surfaces instead of a generic 400.
- `cmd/aicheck` plus `scripts/ai-check.ps1` verify endpoint, key and model with
  one real request, printing usage and latency. The key is never printed.
- Config fail-fast: `AI_ENABLED=true` without a key or base URL stops the
  process at boot.

### Added — L1 notifications

- `mod/notifications`: outbox-based delivery with exponential backoff, template
  rendering, per-channel preferences and an in-app inbox.
- `text/template` for plain text and `html/template` for HTML bodies, so
  user-supplied values cannot inject markup into a mail.
- `missingkey=error` so a template typo fails instead of rendering `<no value>`
  into a customer's inbox.
- `FOR UPDATE SKIP LOCKED` in the worker query, so multiple instances never
  double-send.
- `platform/mailer`: SMTP, log-only and no-op transports behind `core.Mailer`.

### Added — L1 files

- `mod/files`: upload, list, download, delete with ownership scoping.
- Content sniffing via `mimetype`; the detected type wins over the declared one,
  so HTML renamed to `.png` is rejected.
- SVG refused unconditionally, even when added to the allow-list: an SVG can
  carry script, so serving one is stored XSS.
- Path traversal defence in the local storage adapter, including the Windows
  `/absolute` form that `filepath.IsAbs` misses.
- Object keys are derived from a UUID, never from the client's filename;
  checksums stream through a `hashReader` so large uploads are never buffered.
- `platform/storage`: local disk driver with write-then-rename, payload volume.
- `core.Deps` gained `Identity` and `Files`, resolved after module start by
  interface, so modules never import each other.

### Changed

- Replaced echo's `Timeout` middleware with a context-only version.
  `http.TimeoutHandler` buffers the whole response in memory, which breaks
  streaming downloads, and panics on multipart bodies.
- Core modules share the `public` schema; migration files carry ordinals unique
  across modules (`0001_identity_init.sql`, `0002_files_init.sql`) and the
  bootstrap merges them into one history, rejecting duplicate filenames.

### Added — L1 identity

- `mod/identity`: registration, login, refresh rotation with reuse detection,
  logout, logout-everywhere, e-mail verification, password reset, soft delete.
- Argon2id password hashing (OWASP minimum parameters, encoded hash string so
  parameters can be raised without invalidating existing passwords).
- Bearer and HttpOnly-cookie transports behind one token service. Bearer always
  wins when both are present; CSRF is required only for cookie-auth writes.
- Access and refresh secrets are separate and must differ; a production process
  refuses to start without them.
- `token_version` invalidation so a password change kills live access tokens.
- Admin user listing and deletion behind `RequireRole("admin")`.
- `core.ConfigView` so modules read configuration through a narrow interface
  instead of the concrete config struct.
- Modules own their migrations via `MigrationProvider`; the bootstrap discovers
  them by interface, so a forgotten registration is impossible.
- `platform/uuid`: UUIDv7 generator.
- `corectl new --level=L1` now produces a buildable identity-enabled project.

### Added — L0 core kernel

- `platform/` kernel: config, logger, database, migrations, module manager,
  HTTP layer, health endpoints, security helpers, injectable clock.
- `contract/core` as the only package an application may import.
- `cmd/corectl` with `new`, `add`, `doctor`, `upgrade`.
- `references/PRD.md` plus `L0.md`, `L1.md`, `L2.md`, `L3.md`.

### Changed

- Module ordering is a topological sort with cycle detection; an unknown
  dependency is fail-closed.
- Config precedence is env > yaml > default with production fail-fast.
- Error responses use one envelope and stable SCREAMING_SNAKE codes.
- Middleware chain is global so `/healthz` also carries `X-Request-ID`.
- Docker image no longer copies `.env.example` and runs as a non-root user.

### Removed

- The 111 legacy files that made `go build ./...` fail (broken imports,
  destroyed struct tags, four competing `main` functions). They remain in git
  history at commit `707fae7`.
- Committed binaries `main.exe` and `git-filter-repo`.

## [0.1.0]

- Repository initially tagged as "final" but did not compile. Superseded by
  the L0 rewrite above.
