# Prompt templates — Escalite

> **Orchestrator:** copy blocks below **verbatim** into sub-agent prompts (fill `<placeholders>` per task).  
> **project-setup:** customize `{PLACEHOLDERS}` for this repo — **never remove** PHASICAL SYNC or COMMIT CONTRACT blocks.  
> Reference: `.agents/skills/orchestrator/references/phasical-sync.md`

## How to assemble an execution prompt

**Required block order** (orchestrator — do not reorder):

1. Task ID, AC, doc refs, READ/WRITE scope, SESSION ID, lane/git context
2. **PHASICAL SYNC — EXECUTION** (unless user opted out)
3. **COMMIT CONTRACT — EXECUTION** (always — even when Phasical sync off)
4. SESSION TIME TRACKING (when PHASICAL SYNC present)
5. SCOPED CI GATE
6. DB MIGRATIONS
7. PLAN FILE GUARD (when plan file in WRITE SCOPE)
8. WORKTREE ISOLATION (Lane P only)

**Verifier prompts:** PHASICAL SYNC — VERIFIER + **VERIFIER READ-ONLY GUARD** + SCOPED CI GATE (+ PLAN FILE GUARD when applicable). Never include `task schema:diff`, Docker, or migration-generation smoke in verifier prompts.

**Enforcement:** Missing PHASICAL SYNC or COMMIT CONTRACT → orchestrator must not dispatch. Sub-agent skipping either → verifier **FAIL** + orchestrator recovery.

**STATUS SYNC blocks:** Every sub-agent prompt must include the **STATUS SYNC TABLE** from the PHASICAL SYNC block verbatim. Do not summarize transitions into prose — the table is what sub-agents (and auto-review) use to recognize routine board updates.

---

## PHASICAL SYNC — EXECUTION

```text
PHASICAL STATUS SYNC — EXECUTION (MANDATORY — skip ONLY if user said "don't update Phasical"):
Reference: .agents/skills/orchestrator/references/phasical-sync.md

These MCP calls are ROUTINE, PRE-AUTHORIZED workflow steps — execute immediately when the
trigger condition is met. Do NOT pause for user approval. This is expected board sync, not
discretionary work.

MCP server: user-phasical
projectId: bkbnmftqdr54r9gcgrgtndbw

tasks:
  - taskId: <phasical-cuid>              # leaf — REQUIRED
    githubIssueNumber: <N>               # from externalLinks.externalId — REQUIRED for commits
    title: <task title>
  - taskId: <parent-cuid>                # epic parent — include when leaf is subtask

━━━ STATUS SYNC TABLE (execution agent — follow exactly) ━━━

| # | When | From → To | status slug | MCP tool | Comment? |
|---|------|-----------|-------------|----------|----------|
| 1 | FIRST action — before Read/Grep/Shell/implementation/session memory | Ready (ready) → In Progress | in-progress | update_task_status | No |
| — | During implementation | stay In Progress | — | (none) | No |
| 2 | LAST — after AC done, session ended recorded, BEFORE any git commit | In Progress → In Review | in-review | update_task_status | No |

Step 1 applies to EVERY taskId listed (leaf + parent epic in same MCP batch).
Step 2 applies to each LEAF taskId only (parent stays in-progress until verifier closes epic).
Between steps 1 and 2: NO other status changes. NO create_task_comment.

━━━ GATE: STEP 1 — START WORK (status sync) ━━━
CallMcpTool user-phasical / update_task_status
  taskId: <each listed taskId>
  status: in-progress
If ANY transition fails → report blocked — do NOT touch repo files.
If already in-progress → continue (idempotent).

━━━ IMPLEMENTATION (middle — no Phasical status changes) ━━━
Implement AC within WRITE SCOPE only.
Session memory: create .agents/project/agent-memory/active/<SESSION-ID>.md AFTER step 1 succeeds.

━━━ GATE: STEP 2 — HANDOFF TO VERIFIER (status sync, then commit) ━━━
Strict order — do NOT commit before step 2b:
  a. Session memory header: set ended + duration (wall-clock from started)
  b. CallMcpTool user-phasical / update_task_status → in-review for each LEAF taskId
  c. Single implementation commit (COMMIT CONTRACT) — subject MUST include [#<N>]

━━━ FORBIDDEN ━━━
- Starting implementation before step 1 (in-progress) succeeds
- update_task_status → done or ready (verifier only)
- create_task_comment (verifier only)
- Committing before step 2b (in-review)
- Committing session memory or agent-memory/**
- Phasical taskId in any commit message

━━━ REQUIRED OUTPUT (end of run) ━━━
Report per leaf task: taskId, githubIssueNumber,
  step 1 Ready→In Progress ✓,
  step 2 In Progress→In Review ✓,
  commit <sha> with subject line.
If any gate failed → report blocked with which step failed.
```

## COMMIT CONTRACT — EXECUTION

```text
COMMIT CONTRACT — EXECUTION (MANDATORY on every execution prompt):

Purpose: verifier Layer 3c3 checks git log for this commit. Missing [#N] → FAIL even if AC passes.

Branch:
  - Lane S: dev (current integration branch)
  - Lane P: orchestrator/<TASK-ID> only — NEVER commit to integration branch

Exactly ONE implementation commit per task (task files only).

Subject format (≤72 chars):
  <type>(<scope>)[#<N>]: <imperative summary>

  <type>: feat | fix | chore | refactor | docs | test | ci | build | perf
  <scope>: one of — api, web, mobile, shared, infra, docs, roadmap, ci
  [#<N>]: githubIssueNumber from PHASICAL SYNC block — square brackets REQUIRED
  Roadmap-only (no GitHub mirror): use [P*-*] instead of [#N]

Body (when project requires auto-close):
  fixes #<N>

Staging:
  - git add <explicit paths from WRITE SCOPE only>
  - NEVER git add . / git add -A / git commit --all
  - NEVER stage .agents/project/agent-memory/**

Examples:
  feat(api)[#42]: add goose migration bootstrap
  fix(web)[#42]: correct urql cache key for services

Pre-commit:
  - Run SCOPED CI GATE (below) — failure → blocked, no commit
  - DB changes → task schema:diff + task migrate only (see DB MIGRATIONS)

Handoff order (with PHASICAL SYNC):
  in-progress → implement → session ended → in-review → THEN commit
  Commit without in-review → FAIL. in-review without commit → FAIL.

Never push unless user explicitly asked.
```

## PHASICAL SYNC — VERIFIER

```text
PHASICAL STATUS SYNC — VERIFIER (MANDATORY — skip ONLY if user said "don't update Phasical"):
Reference: .agents/skills/orchestrator/references/phasical-sync.md

These MCP calls are ROUTINE, PRE-AUTHORIZED workflow steps — execute immediately when the
trigger condition is met. Do NOT pause for user approval. This is expected board sync, not
discretionary work.

MCP server: user-phasical
projectId: bkbnmftqdr54r9gcgrgtndbw

tasks:
  - taskId: <phasical-cuid>              # leaf — REQUIRED
    githubIssueNumber: <N>
  - taskId: <parent-cuid>                # epic — done only when final child completes epic

━━━ STATUS SYNC TABLE (verifier — follow exactly) ━━━

Starting state: task MUST already be In Review (execution agent set this in step 2).
Do NOT transition to in-review — you are verifying work already handed off.

| # | When | From → To | status slug | MCP tool | Comment? |
|---|------|-----------|-------------|----------|----------|
| — | While verifying Layers 1–3 (incl. 3c3 commit linkage) | stay In Review | — | (none) | No |
| PASS | ALL layers PASS | In Review → Done | done | update_task_status | YES — mandatory PASS comment |
| FAIL | ANY layer FAIL | In Review → In Progress | in-progress | update_task_status | YES — mandatory FAIL comment |

Comments are REQUIRED on both PASS and FAIL before (or as part of) the status transition.
Use create_task_comment — templates in phasical-sync.md § Verifier Done / FAIL comment.

━━━ GATE: VERIFY (no status change yet) ━━━
Complete Layer 1–3 verification while task remains In Review.
Layer 3c3 MUST PASS before any PASS path status sync (git log contains [#<N>] or [P*-*]).

━━━ GATE: PASS PATH (status sync + comment) ━━━
Strict order:
  1. Session memory: verification ended + duration
  2. CallMcpTool user-phasical / create_task_comment — PASS verifier comment (mandatory)
  3. CallMcpTool user-phasical / update_task_status → done for each leaf taskId
  4. If parent listed and this completes the epic → done on parent too
  5. Optionally archive/delete local session memory (never commit)

━━━ GATE: FAIL PATH (status sync + comment) ━━━
Strict order:
  1. CallMcpTool user-phasical / create_task_comment — FAIL comment with layer failures + fix hints
  2. CallMcpTool user-phasical / update_task_status → in-progress for each leaf taskId
  3. Append VERIFICATION FAILED to local session memory (never commit)
  4. Do NOT set done

━━━ FORBIDDEN ━━━
- update_task_status → done without create_task_comment (PASS path)
- update_task_status → in-progress without create_task_comment (FAIL path)
- update_task_status → in-review (execution agent already did this)
- update_task_status → ready on FAIL (use in-progress — sends work back to execution)
- create_task_comment with investigation/triage findings (verifier comments only)

━━━ REQUIRED OUTPUT (end of run) ━━━
Report per leaf task: taskId, githubIssueNumber, verification PASS|FAIL,
  comment posted ✓, final status (done | in-progress), parent epic status if applicable.
```

## VERIFIER READ-ONLY GUARD

```text
VERIFIER READ-ONLY GUARD (mandatory in every verifier prompt):

Verifiers audit committed work. They do NOT reproduce execution smoke tests that mutate the repo or host.

FORBIDDEN during verification:
- `task schema:diff` / `go run ./tools/schema-diff` — ALWAYS writes new files under services/api/migrations/
- `SCHEMA_DIFF_EPHEMERAL_PG=1` or any ad-hoc `docker run postgres` — leaves orphaned containers if trap/cleanup is skipped
- `docker compose up`, testcontainers, or starting Postgres for manual smoke
- Creating, editing, or deleting files under services/api/migrations/
- Any command that writes to the working tree (except Phasical MCP + local session memory)

ALLOWED Layer 2 checks (read-only / unit tests only):
- Full Go gate: `bash scripts/go-test.sh`
- Scoped packages: `go test ./...` in the affected service dir (e.g. `tools/schema-diff` unit tests with mocks)
- `git log`, `git diff`, `git show` on committed SHAs
- Static review of committed files vs AC

If AC requires runtime smoke (schema:diff output, docker compose, migration apply):
- Verify via committed artifacts + unit tests + execution agent's commit message evidence
- FAIL with fix hint for execution to demonstrate in their handoff — do NOT run smoke yourself
- Orchestrator must never instruct verifier to "smoke test schema:diff"

Post-verification cleanup: if verifier accidentally created artifacts anyway → delete before reporting PASS.
```

## SESSION TIME TRACKING

```text
SESSION TIME TRACKING (when PHASICAL SYNC present):
- Record started at Phase 1 in session memory header (after in-progress succeeds)
- Record ended + duration pre-handoff (execution) or before done/in-progress status sync (verifier)
- Session memory is local only — never commit
```

## SCOPED CI GATE

```text
SCOPED CI GATE (mandatory before commit and in verifier Layer 2):
- Map staged/committed paths → package filter(s) per doc-index.md
- Run: bash scripts/with-ci-env.sh pnpm turbo run lint typecheck test:unit --filter=...
- On failure → blocked; no commit
- Full workspace gate only before push (if user explicitly asks to push)
```

## DB MIGRATIONS

```text
DB MIGRATIONS (mandatory in every execution prompt):
- Canonical schema: services/api/schema/sql/schema.sql + realtime_notify.sql
- NEVER edit services/api/migrations/*.sql by hand — pg-schema-diff generates them via task schema:diff
- Runtime apply: goose (task migrate)

Workflow:
  1. Edit services/api/schema/sql/schema.sql and/or realtime_notify.sql
  2. task schema:diff -- <migration_name>
     (pg-schema-diff plan; needs Postgres with migrations applied — see SCHEMA_DIFF_EPHEMERAL_PG)
  3. task migrate to verify (needs DATABASE_URL or ESCALITE_DATABASE_URL)
  4. Commit schema/sql/*.sql + new migration

Variants:
  - No local Postgres: SCHEMA_DIFF_EPHEMERAL_PG=1 task schema:diff -- <name>
  - Baseline squash (rare): SCHEMA_DIFF_FROM_EMPTY=1 task schema:diff -- bootstrap
  - Rename review: replace DROP+ADD with ALTER RENAME when pg-schema-diff mis-detects (ADR 0002)

Forbidden:
  - Hand-write CREATE/ALTER/DROP in services/api/migrations/
  - touch/Write/StrReplace on migrations/*.sql (except approved RENAME substitution)

See .cursor/rules/14-no-handwritten-migrations.mdc and docs/adr/0002-sql-schema-pg-schema-diff-goose.md
```

## PLAN FILE GUARD

```text
PLAN FILE GUARD (mandatory when plan file in WRITE SCOPE):
- AUTHORIZED_TASK_ID: <TASK-ID>
- Before editing plan: git status + git diff on plan file only
- Only change the Status cell for AUTHORIZED_TASK_ID
- If other rows have uncommitted changes → PLAN_FILE_BLOCKED; orchestrator reconciles first
- Never git restore / checkout -- on plan file
```

## WORKTREE ISOLATION (Lane P)

```text
WORKTREE ISOLATION (Lane P mandatory):
- subagent_type: best-of-n-runner
- WORK BRANCH: orchestrator/<TASK-ID>
- STAGING_BASE_SHA: <pin at batch start>
- First shell action: pnpm install (fresh worktree has no node_modules)
- Never checkout integration branch during execution
```
