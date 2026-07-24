# ADR 0001: Atlas declarative schema for database migrations

**Status:** Accepted  
**Date:** 2026-07-24  
**Supersedes:** goose + hand-written SQL migrations (doc 08 prior revision)

## Context

Escalite uses PostgreSQL with `pgx` + `sqlc` (no ORM). The initial Phase 0 plan used `goose` with hand-written SQL migration files. That approach makes schema drift likely: migrations can diverge from the intended model, columns can be missing or extra, and there is no automated gate enforcing consistency.

We want a single canonical database schema, generated versioned migrations, and CI that fails when migrations do not match the declared schema — while keeping explicit SQL queries via sqlc and Postgres-native features (advisory locks, LISTEN/NOTIFY, river).

## Decision

Adopt **Atlas** (`ariga/atlas`) for schema management:

| Concern | Location / tool |
| ------- | ---------------- |
| Canonical schema | `services/api/schema/` (Atlas HCL — default format) |
| Versioned migrations | `services/api/migrations/` — **generated only** via `atlas migrate diff` |
| Query codegen | `sqlc` reads canonical schema + `services/api/queries/` |
| Runtime apply | API startup with Postgres advisory lock (preserve single-node MVP semantics) |
| CI | `atlas migrate lint` + schema drift check |

**Zero hand-written DDL migrations.** Agents and humans edit the canonical schema, then run `atlas migrate diff` and commit schema + generated migration together.

**DML-only exception:** one-off data backfills may use hand-written DML in a migration file when paired with an ADR note in the PR — never DDL.

## Consequences

### Positive

- Schema drift caught in CI before merge
- Single source of truth for sqlc and migrations
- Keeps `pgx` + `sqlc` (no ORM); explicit queries remain reviewable
- Works identically for self-hosted CE and per-tenant Cloud instances (doc 04 isolation model)

### Negative / trade-offs

- Team must learn Atlas workflow (`atlas.hcl`, `migrate diff`, `migrate lint`)
- Existing goose bootstrap (#38) must be pivoted before EL-2 schema subtasks continue
- Generated migrations are still SQL files — reviewers must read diffs, not edit DDL inline

### Follow-up

- Replace goose runtime with Atlas migrate apply (or Atlas-compatible runner)
- Add `task schema:diff` and `task migrate` wrappers in `Taskfile.yml`
- Wire CI drift gate in `p0-ci-skeleton` follow-up or Atlas setup task

## References

- Doc 08 — Implementation Decisions & Conventions (`#database-schema-workflow`)
- `.cursor/rules/14-no-handwritten-migrations.mdc`
