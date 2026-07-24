# Prompt templates — Escalite

> **Orchestrator:** copy blocks below **verbatim** into sub-agent prompts (fill `<placeholders>` per task).  
> **project-setup:** customize `{PLACEHOLDERS}` for this repo — **never remove** KANEO SYNC or COMMIT CONTRACT blocks.  
> Reference: `.agents/skills/orchestrator/references/kaneo-sync.md`

## How to assemble an execution prompt

**Required block order** (orchestrator — do not reorder):

1. Task ID, AC, doc refs, READ/WRITE scope, SESSION ID, lane/git context
2. **KANEO SYNC — EXECUTION** (unless user opted out)
3. **COMMIT CONTRACT — EXECUTION** (always — even when Kaneo sync off)
4. SESSION TIME TRACKING (when KANEO SYNC present)
5. SCOPED CI GATE
6. DB MIGRATIONS
7. PLAN FILE GUARD (when plan file in WRITE SCOPE)
8. WORKTREE ISOLATION (Lane P only)

**Verifier prompts:** KANEO SYNC — VERIFIER + SCOPED CI GATE (+ PLAN FILE GUARD when applicable).

**Enforcement:** Missing KANEO SYNC or COMMIT CONTRACT → orchestrator must not dispatch. Sub-agent skipping either → verifier **FAIL** + orchestrator recovery.

---

## KANEO SYNC — EXECUTION

```text
KANEO SYNC — EXECUTION (MANDATORY — skip ONLY if user said "don't update Kaneo"):
Reference: .agents/skills/orchestrator/references/kaneo-sync.md

MCP server: user-kaneo
projectId: bkbnmftqdr54r9gcgrgtndbw

tasks:
  - taskId: <kaneo-cuid>              # leaf — REQUIRED
    githubIssueNumber: <N>             # from externalLinks.externalId — REQUIRED for commits
    title: <task title>
  - taskId: <parent-cuid>              # epic parent — include when leaf is subtask

━━━ GATE: FIRST ACTIONS (before Read/Grep/implementation/session memory) ━━━
CallMcpTool user-kaneo / update_task_status
  → status: in-progress
  → for EVERY taskId listed above (leaf + parent in same batch)
If ANY transition fails → status: blocked — report error — do NOT touch repo files.
If already in-progress → continue (idempotent).

━━━ IMPLEMENTATION (middle) ━━━
Implement AC within WRITE SCOPE only.
Session memory: create .agents/project/agent-memory/active/<SESSION-ID>.md after in-progress succeeds.

━━━ GATE: LAST ACTIONS (strict order — do NOT commit before step 2) ━━━
1. Session memory header: set ended + duration (wall-clock from started)
2. CallMcpTool user-kaneo / update_task_status → in-review for each LEAF taskId
3. Single implementation commit (see COMMIT CONTRACT below) — subject MUST include [#<N>]

━━━ FORBIDDEN ━━━
- Starting implementation before in-progress MCP succeeds
- update_task_status → done (verifier only)
- create_task_comment (verifier only)
- Committing before in-review transition
- Committing session memory or agent-memory/**
- Kaneo taskId in any commit message

━━━ REQUIRED OUTPUT (end of run) ━━━
Report per task: taskId, githubIssueNumber, in-progress ✓, in-review ✓, commit <sha> with subject line.
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
  [#<N>]: githubIssueNumber from KANEO SYNC block — square brackets REQUIRED
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
  - DB changes → goose -dir services/api/migrations postgres "$DATABASE_URL" up only (see DB MIGRATIONS)

Handoff order (with KANEO SYNC):
  in-progress → implement → session ended → in-review → THEN commit
  Commit without in-review → FAIL. in-review without commit → FAIL.

Never push unless user explicitly asked.
```

## KANEO SYNC — VERIFIER

```text
KANEO SYNC — VERIFIER (MANDATORY — skip ONLY if user said "don't update Kaneo"):
Reference: .agents/skills/orchestrator/references/kaneo-sync.md

MCP server: user-kaneo
projectId: bkbnmftqdr54r9gcgrgtndbw

tasks:
  - taskId: <kaneo-cuid>
    githubIssueNumber: <N>
  - taskId: <parent-cuid>              # epic — done only when final child completes epic

━━━ BEFORE status changes ━━━
Layer 1–3 verification must PASS (including 3c3 commit linkage — git log contains [#<N>] or [P*-*]).

━━━ AFTER PASS (strict order) ━━━
1. Session memory: verification ended + duration
2. create_task_comment — mandatory structured Done summary (see kaneo-sync.md § Verifier Done comment)
3. update_task_status → done for each leaf taskId
4. If parent listed and epic complete → done on parent
5. Optionally archive/delete local session memory (never commit)

━━━ AFTER FAIL ━━━
1. create_task_comment — FAIL template with layer failures + fix hints
2. update_task_status → to-do (Ready) for leaf taskId
3. Append VERIFICATION FAILED to local session memory
4. Do NOT set done

━━━ FORBIDDEN ━━━
- done without create_task_comment
- create_task_comment with investigation findings (triage only)
```

## SESSION TIME TRACKING

```text
SESSION TIME TRACKING (when KANEO SYNC present):
- Record started at Phase 1 in session memory header (after in-progress succeeds)
- Record ended + duration pre-handoff (execution) or before Done/Ready (verifier)
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
- Schema changes → use project migration CLI only (goose -dir services/api/migrations postgres "$DATABASE_URL" up)
- Never hand-write migration.sql or create migration directories manually
- If CLI cannot run → report blocked; no SQL workaround
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
