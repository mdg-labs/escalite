# Workspace notes — Escalite

Durable project learnings for the orchestrator. Not session-specific.

## Conventions

- Phasical is the board source of truth; GitHub issues mirror via sync.
- Commits use `[#N]` from Phasical `externalLinks`.
- Outline MCP is authoritative for spec bodies; `docs/specs/*.md` are pointers only.
- Active plan: `docs/frontend-refactor-plan.yaml` (+ human index `frontend-refactor-plan.md`). MVP roadmap removed from git — see `docs/roadmap/README.md`.

---

## Atlas removed (use goose + pg-schema-diff only)

Atlas was dropped in favor of [ADR 0002](../../docs/adr/0002-sql-schema-pg-schema-diff-goose.md). **Never** run `atlas migrate *`, `atlas migrate hash`, or create/edit `atlas.sum`.

| Step | Command / module |
| ---- | ---------------- |
| Edit canonical schema | `services/api/schema/sql/schema.sql` (+ `realtime_notify.sql`) |
| Generate migration | `task schema:diff -- <name>` |
| Apply locally | `task migrate` |
| Runtime / tests apply | `services/dbmigrate` (goose + advisory lock) |

Integration tests use goose via `dbmigrate`, not the Atlas CLI.
_added: 2026-07-26_

---

Verification agents must **never** run `task schema:diff`, `SCHEMA_DIFF_EPHEMERAL_PG=1`, or ad-hoc `docker run postgres` — those write migration files and/or leave orphaned containers (`escalite-vrf-*`, `escalite-schema-diff-*`). Verifiers use `go test` (unit), `git log`, and static review only. See `prompt-templates.md` § VERIFIER READ-ONLY GUARD.
_added: 2026-07-26_


## Lane P `best-of-n-runner` background stalls

Several `run_in_background: true` best-of-n-runner dispatches (EL-245, batch 3) produced transcript files with only the user prompt — no assistant turns. Recovery: re-dispatch as Lane S `generalPurpose` (serial on `dev`). EL-243/EL-244 Lane P succeeded when agents started promptly.
_added: 2026-07-28_

## <topic>

<2-4 lines>
_added: YYYY-MM-DD_

-->
