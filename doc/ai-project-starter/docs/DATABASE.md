# DATABASE CONTRACT

## Principles

- PostgreSQL is the application relational source of truth for application data.
- Schema changes happen through versioned migrations.
- Production schema must never be changed manually as the normal workflow.
- Foreign keys and constraints should encode important invariants.
- Avoid duplicated state unless its synchronization rule is explicit.

## Modeling rules

For each entity define:
- identity
- ownership
- lifecycle
- required fields
- nullable fields
- unique constraints
- foreign keys
- indexes
- deletion behavior
- audit requirements

## Transactions

Use a transaction when multiple writes must succeed or fail together.

## IDs

Choose one project-wide ID strategy and document it. Do not mix strategies casually.

## Timestamps

Use a consistent timezone strategy. Store timestamps in UTC at the persistence boundary unless a documented requirement says otherwise.

## Migrations

Every schema change must include:
- migration
- reason
- compatibility consideration
- rollback/recovery consideration where applicable

Never silently edit an old migration that may already have been applied.

## Stripe-linked data

Keep only application-required Stripe identifiers and state locally.

Typical examples:
- stripe_customer_id
- stripe_subscription_id
- stripe_price_id
- last_processed_webhook_event_id or a dedicated webhook event table

Do not assume local billing state is authoritative until it has a defined synchronization rule.

## Performance

Add indexes based on actual query patterns and constraints, not speculation.

Do not introduce caching before measuring a real bottleneck.
