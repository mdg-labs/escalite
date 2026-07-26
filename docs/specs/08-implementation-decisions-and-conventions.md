# 08 — Implementation Decisions & Conventions (agent-binding)

**Outline URL:** https://outline.mdg-labs.dev/doc/08-implementation-decisions-conventions-agent-binding-VCZBeGw3HB  
**Outline document id:** `19d87fc5-7bbd-45ba-be42-f40eeb253783`

This doc pins every previously open "X or Y" choice for autonomous implementers: chi, gqlgen, pgx+sqlc, pg-schema-diff + goose (declarative SQL schema), river, pnpm, Turborepo, Vite, urql, Expo, GitHub Actions, UUIDv7, slog, and more. Database schema is canonical in `services/api/schema/sql/`; migrations are generated via `task schema:diff` (pg-schema-diff) and applied at runtime with goose — zero hand-written DDL. See `docs/adr/0002-sql-schema-pg-schema-diff-goose.md`. It defines API error codes, naming conventions, env prefix `ESCALITE_`, conventional commits with DCO, and ADR requirements for deviations. The Definition of Done requires code, tests (integration Postgres for timing/scheduling/auth), working `docker compose up`, docs/ADR updates, and clean lint/vuln scans. Phase 0 bootstrap order is explicit: scaffold → compose → schema → auth → GraphQL/codegen/web shell → river → E2E smoke.

> **Source of truth:** Outline MCP (collection: Escalite, doc id: `19d87fc5-7bbd-45ba-be42-f40eeb253783`). Fetch via Outline MCP before relying on this doc's content — this file is a pointer, not a mirror.

## Database schema workflow {#database-schema-workflow}

Mandatory for all schema changes. Full detail: [ADR 0002](../adr/0002-sql-schema-pg-schema-diff-goose.md) and `.cursor/rules/14-no-handwritten-migrations.mdc`.

| Concern | Location / tool |
| ------- | ---------------- |
| Canonical schema | `services/api/schema/sql/schema.sql` (tables) + `realtime_notify.sql` (NOTIFY functions/triggers) |
| Versioned migrations | `services/api/migrations/` — **generated only** via `task schema:diff` |
| Query codegen | `sqlc` reads canonical schema + `services/api/queries/` |
| Runtime apply | goose at API startup with Postgres advisory lock |
| CI | Drift gate via `tools/schema-diff --check-drift` (see task #147) |

**Zero hand-written DDL migrations.** Edit canonical SQL, generate, verify, commit source + migration together.

```bash
# 1. Edit services/api/schema/sql/schema.sql and/or realtime_notify.sql

# 2. Generate migration (from repo root; needs Postgres with migrations applied)
task schema:diff -- add_my_column

# 3. Verify
task migrate   # goose apply; needs DATABASE_URL or ESCALITE_DATABASE_URL

# 4. Commit: schema/sql/*.sql + migrations/<new>.sql
```

**Variants:**

| Mode | Command |
| ---- | ------- |
| No local Postgres | `SCHEMA_DIFF_EPHEMERAL_PG=1 task schema:diff -- <name>` |
| Baseline squash (rare) | `SCHEMA_DIFF_FROM_EMPTY=1 task schema:diff -- bootstrap` |
| Faster local runs | `SCHEMA_DIFF_SKIP_VALIDATION=1 task schema:diff -- <name>` |

**Rename policy:** pg-schema-diff treats column/table renames as drop + add. Review generated SQL; substitute `ALTER TABLE … RENAME` when semantically renaming (see ADR 0002).

**DML-only exception:** one-off data backfills may use hand-written DML in a migration file when paired with an ADR note in the PR — never DDL.

**Superseded:** [ADR 0001](../adr/0001-atlas-declarative-schema.md) (Atlas HCL workflow).
