# ADR 0002: SQL schema with pg-schema-diff and goose

**Status:** Accepted (spike proven)  
**Date:** 2026-07-26  
**Supersedes:** [ADR 0001 — Atlas declarative schema](./0001-atlas-declarative-schema.md)

## Context

Escalite uses PostgreSQL with `pgx` + `sqlc` (no ORM). ADR-0001 adopted Atlas for declarative schema management, but Atlas free tier cannot diff or inspect Postgres **functions and triggers** — a hard paywall for NOTIFY triggers used by GraphQL realtime fan-out.

We already maintain canonical SQL mirrors:

| File | Contents |
| ---- | -------- |
| `services/api/schema/sql/schema.sql` | Tables, indexes, FKs, checks (exported from HCL during Atlas era) |
| `services/api/schema/sql/realtime_notify.sql` | `CREATE FUNCTION` + `CREATE TRIGGER` for LISTEN/NOTIFY |

The Atlas workaround (separate stamp script for triggers) adds workflow friction and splits the mental model. We need one diff tool that handles tables **and** triggers, generates versioned migrations, and applies them at runtime with advisory-lock semantics preserved from doc 08.

## Decision

Adopt **pg-schema-diff** ([`github.com/stripe/pg-schema-diff`](https://github.com/stripe/pg-schema-diff)) for migration **generation** and **goose** for migration **apply** (runtime, advisory lock — wired in follow-up tasks #144–#145).

**Pinned version:** `v1.0.7` (CLI: `go install github.com/stripe/pg-schema-diff/cmd/pg-schema-diff@v1.0.7`)

| Concern | Location / tool |
| ------- | ---------------- |
| Canonical schema | `services/api/schema/sql/schema.sql` + `realtime_notify.sql` |
| Versioned migrations | `services/api/migrations/` — **generated only** via `pg-schema-diff plan` |
| Query codegen | `sqlc` reads canonical schema + `services/api/queries/` |
| Runtime apply | goose at API startup with Postgres advisory lock |
| CI | Drift gate: diff live/empty schema against canonical SQL (task #147) |

**Zero hand-written DDL migrations.** Agents and humans edit canonical SQL, then run `task schema:diff` and commit SQL + generated migration together.

### Migration generation workflow

pg-schema-diff needs a **running Postgres server** to load canonical SQL and introspect catalog metadata (tables, functions, triggers). It creates temporary databases on that server; it does not diff `.sql` files as plain text.

| Mode | Command | Postgres source |
| ---- | ------- | --------------- |
| **Incremental** (normal) | `task schema:diff -- add_column` | `DATABASE_URL` → compose/dev Postgres with migrations already applied |
| **Baseline squash** (rare) | `SCHEMA_DIFF_FROM_EMPTY=1 task schema:diff -- bootstrap` | same server; diff from empty → canonical SQL |
| **No local Postgres** | `SCHEMA_DIFF_EPHEMERAL_PG=1 task schema:diff -- <name>` | one-off Docker container; removed on exit via `trap` |
| **Faster local runs** | `SCHEMA_DIFF_SKIP_VALIDATION=1 task schema:diff -- <name>` | skips pg-schema-diff plan replay validation |

Default `DATABASE_URL` when unset: `postgres://escalite:escalite@127.0.0.1:5432/escalite?sslmode=disable` (compose dev port mapping).

**DML-only exception:** one-off data backfills may use hand-written DML in a migration file when paired with an ADR note in the PR — never DDL.

### File load order

Within a single `--to-dir`, pg-schema-diff applies `.sql` files in **lexical order**. `realtime_notify.sql` sorts before `schema.sql`, which fails because triggers reference tables that do not exist yet.

**Policy (until #144 wires `task schema:diff`):** pass two `--to-dir` flags in dependency order:

```bash
pg-schema-diff plan \
  --from-dsn "$DATABASE_URL" \
  --to-dir services/api/schema/sql-tables \   # contains schema.sql only
  --to-dir services/api/schema/sql-triggers   # contains realtime_notify.sql only
```

Alternatively, prefix filenames (`01_schema.sql`, `02_realtime_notify.sql`) in a single directory. The generator task (#144) will enforce ordering so agents never hit this manually.

### Column / table rename policy

pg-schema-diff treats renames as **drop + add**, which is destructive for columns with data.

**Example** (spike 2026-07-26): renaming `organizations.name` → `organizations.org_name` in `schema.sql` produced:

```sql
ALTER TABLE "public"."organizations" ADD COLUMN "org_name" text ... NOT NULL;
ALTER TABLE "public"."organizations" DROP COLUMN "name";
```

**Policy:**

1. After `pg-schema-diff plan`, **review the generated SQL** before committing.
2. When the plan shows `DROP COLUMN` + `ADD COLUMN` (or `DROP TABLE` + `CREATE TABLE`) for what is semantically a rename, **replace** those statements with `ALTER TABLE ... RENAME COLUMN ...` (or `ALTER TABLE ... RENAME TO ...`) in the generated migration file.
3. Document the manual substitution in the PR body.
4. Never rely on pg-schema-diff to infer renames — it does not.

Index renames may use pg-schema-diff's temporary-index swap pattern (`ALTER INDEX ... RENAME TO pgschemadiff_tmpidx_...`); that is safe and expected.

### Hazard annotations

pg-schema-diff annotates statements with hazards (e.g. `HAS_UNTRACKABLE_DEPENDENCIES` for plpgsql functions). Review hazard comments in plan output. Runtime `apply` (if used) requires `--allow-hazards` flags; Escalite will generate goose migrations from `plan` output and apply via goose instead.

## Spike evidence (task #142)

**Environment:** Postgres 16 Alpine (Docker), `pg-schema-diff` v1.0.7, Escalite schema as of 2026-07-26.

**Reproduce:**

```bash
go install github.com/stripe/pg-schema-diff/cmd/pg-schema-diff@v1.0.7
./services/api/scripts/pg-schema-diff-spike.sh
```

**Results:**

| Check | Result |
| ----- | ------ |
| `pg-schema-diff plan` (empty DB → full schema incl. NOTIFY triggers) | PASS — 3710 lines of SQL |
| Apply generated plan on empty database | PASS — `psql` with `ON_ERROR_STOP=1`, no errors |
| Tables created | 40 |
| NOTIFY functions | `notify_alert_change`, `notify_schedule_row_change` |
| NOTIFY triggers | 5 (`alerts_notify_*`, `schedules_notify_change`, `rotations_notify_change`, `overrides_notify_change`) |

**Sample plan head** (functions before tables — pg-schema-diff dependency ordering):

```
SET SESSION statement_timeout = 3000;
SET SESSION lock_timeout = 3000;
CREATE OR REPLACE FUNCTION public.notify_alert_change() ...
```

**Lexical-order failure** (documented, not used in spike):

```
$ pg-schema-diff plan --from-dsn ... --to-dir services/api/schema/sql
Error: ... realtime_notify.sql): ERROR: relation "alerts" does not exist
```

## Consequences

### Positive

- Single diff path for tables, functions, and triggers — no Atlas Pro paywall
- Canonical SQL is human-readable and matches sqlc input
- Stripe-maintained tool with plan validation (temp DB replay)
- Aligns with existing `realtime_notify.sql` source file

### Negative / trade-offs

- Must enforce SQL file load order (two dirs or numeric prefixes)
- Column/table renames require manual `RENAME` substitution in generated migrations
- Plan output includes per-statement `SET SESSION` timeouts — goose migrations will store these as-is or strip in generator (#144)
- Team must learn pg-schema-diff + goose workflow; Atlas HCL (`schema.hcl`) removed in #143

### Follow-up (epic #141)

| Task | Work |
| ---- | ---- |
| #143 | Remove Atlas artifacts; make `schema/sql/` canonical |
| #144 | Wire `task schema:diff` generator |
| #145 | Replace Atlas migrate apply with goose |
| #146 | Testcontainers integration tests |
| #147 | CI drift gate |
| #148 | Docs + agent rules |

## References

- [ADR 0001](./0001-atlas-declarative-schema.md) — superseded
- Doc 08 — Implementation Decisions & Conventions
- Epic [#141](https://github.com/mdg-labs/escalite/issues/141) — Drop Atlas; SQL schema with pg-schema-diff and goose
- Spike script: `services/api/scripts/pg-schema-diff-spike.sh`
- pg-schema-diff: https://github.com/stripe/pg-schema-diff (v1.0.7)
