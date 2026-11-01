# DOMAIN MODEL

This document is intentionally project-specific.

Before implementation, define:

## Actors

- User:
- Admin:
- Other roles:

## Core entities

For each entity:
- purpose
- identity
- owner
- lifecycle
- invariants

## Business rules

Write business rules in plain language before encoding them.

Example:

A user may access a paid feature only when the application has a verified active entitlement.

## State machines

For each stateful entity document:

States:
- ...

Allowed transitions:
- ...

Forbidden transitions:
- ...

Side effects:
- ...

## Terminology

Define canonical names for domain concepts.

Agents must not introduce synonyms for established domain concepts without reason.
