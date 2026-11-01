# SECURITY BASELINE

Security principle: Secure by Default.

## Secrets

Never expose:
- database credentials
- Stripe secret key
- webhook signing secret
- auth secrets
- private API keys

Use environment variables or approved secret storage.

Never commit .env files containing secrets.

## Authentication

Authentication proves identity.

Authorization decides whether an authenticated identity may perform an action.

Never treat authentication as authorization.

Every protected server operation must verify authorization independently of UI visibility.

## Input validation

Validate all external input at the boundary:
- forms
- API requests
- query parameters
- webhooks
- imported files
- third-party responses

Do not assume TypeScript types make runtime input safe.

## Stripe

- Stripe secret operations run server-side.
- Verify webhook signatures.
- Treat webhook delivery as retryable.
- Store and check processed event IDs for idempotency.
- Never trust client-provided payment status.
- Never trust client-provided price or amount as authoritative.
- Do not grant paid entitlements solely from a browser redirect.
- Use verified server-side billing state.

## Database

- Use parameterized queries/ORM.
- Apply least privilege where practical.
- Use transactions for multi-step invariants.
- Do not log secrets or sensitive payment data.
- Destructive migrations require explicit review.

## Logging

Logs must not contain:
- passwords
- tokens
- secret keys
- full payment credentials
- unnecessary personal data

## Dependency security

Before adding a dependency:
- establish the need;
- check whether existing tooling solves it;
- assess maintenance/security implications;
- document the reason for significant dependencies.

## Security changes

Any change affecting auth, permissions, billing, secrets, webhook handling, or sensitive data is HIGH or CRITICAL risk.
