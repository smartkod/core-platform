# AI AGENT OPERATING CONTRACT

## Mission

Build and maintain the product without degrading its architecture, security, reliability, or maintainability.

The agent must optimize for:
- correctness
- simplicity
- explicit boundaries
- minimal change surface
- testability
- security
- reversibility

## Golden rules

1. Inspect before modifying.
2. Reuse before creating.
3. Need before capability.
4. Complexity must earn its existence.
5. Never perform unrelated refactoring during a feature task.
6. Never silently change an architectural decision.
7. Never expose secrets.
8. Never trust client-controlled authorization or billing state.
9. Never modify the database without a migration.
10. Never add a dependency without justification.
11. Never delete behavior merely because it appears unused without evidence.
12. Never invent APIs, environment variables, database fields, or provider behavior.
13. Prefer the smallest complete solution over the most sophisticated solution.
14. Treat external input and AI-generated data as untrusted.
15. If a requirement conflicts with a documented architectural rule, stop and report the conflict.

## Required workflow

### Before coding
- Read relevant docs.
- Search the repository for existing implementations.
- Identify affected modules.
- Identify data model/API/security implications.
- Produce a concise change plan.
- Assign a risk level: LOW, MEDIUM, HIGH, CRITICAL.

### During coding
- Change only the required scope.
- Preserve public contracts unless the change explicitly requires them.
- Keep business logic out of UI components.
- Keep secrets server-side.
- Validate external input at boundaries.
- Use existing abstractions when they are adequate.

### After coding
Run, as applicable:
- typecheck
- lint
- unit/integration tests
- build
- end-to-end tests for affected critical flows

Then report:
- files changed
- behavior changed
- tests run
- architectural impact
- security impact
- unresolved risks

## Risk gates

LOW:
Routine UI, copy, isolated component. May proceed automatically.

MEDIUM:
New feature or cross-module change. Plan required.

HIGH:
Authentication, authorization, database migration, billing, webhook, external provider, sensitive data. Explicit plan and validation required.

CRITICAL:
Security architecture, payment architecture, destructive migration, irreversible data operation, major architecture change. Human approval required before implementation.

## Stop conditions

Stop and ask for a decision when:
- two architecture documents conflict;
- the requested change requires a breaking public contract;
- data loss is possible;
- payment/security behavior is ambiguous;
- a new service is proposed without a concrete requirement;
- the agent cannot establish the source of truth for a critical state.
