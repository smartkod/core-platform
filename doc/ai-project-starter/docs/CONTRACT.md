# DEVELOPMENT CONTRACT

## Before implementation

The agent MUST:
- inspect existing code;
- search for reusable functionality;
- read relevant architecture/security/domain documents;
- identify affected boundaries;
- identify data migrations;
- identify external integrations;
- classify risk.

## Implementation

The agent MUST:
- preserve established conventions;
- keep changes localized;
- validate inputs;
- enforce authorization server-side;
- use transactions where required by business invariants;
- make external operations idempotent where retries are possible;
- avoid duplicate business logic.

The agent MUST NOT:
- rewrite unrelated modules;
- change dependencies casually;
- disable tests to make a build pass;
- weaken security checks;
- hard-code credentials;
- trust client-submitted price, role, entitlement, or payment status.

## Completion criteria

A task is complete only when:
- implementation is present;
- affected tests pass;
- typecheck/lint/build are clean or documented;
- migrations are included where required;
- security implications are checked;
- architecture remains compliant;
- documentation is updated when a contract changed.

## Final report format

Summary:
Files:
Tests:
Migration:
Security:
Architecture:
Risks:
Follow-up:
