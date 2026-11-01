# DEVELOPMENT WORKFLOW

## Local quality gate

Before considering a change complete:

1. install/update dependencies only when necessary;
2. typecheck;
3. lint;
4. run unit/integration tests;
5. run build;
6. run targeted E2E tests for critical user flows.

## Git

Prefer small, coherent commits.

Commit should describe the actual change, not the implementation process.

## Environment

Maintain:
- .env.example
- documented required variables
- safe local defaults where possible

Never commit real credentials.

## Observability

For production-critical operations, define:
- useful structured logs
- error reporting
- request/operation identifiers where appropriate
- billing/webhook traceability

Do not add an observability platform solely because it is fashionable; add it when operationally useful.
