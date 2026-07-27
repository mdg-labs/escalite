# Doc index — Escalite

> Local spec shorthand map. Expand per project.

## Spec documents

| Shorthand | Path | Topics |
| --------- | ---- | ------ |
| `spec` | `docs/specs/README.md` | Spec index, Outline pointers |
| `spec-00` | `docs/specs/00-vision-and-scope.md` | Vision, MVP scope, non-goals |
| `spec-01` | `docs/specs/01-architecture-and-monorepo.md` | Stack, monorepo layout, deployment |
| `spec-08` | `docs/specs/08-implementation-decisions-and-conventions.md` | Pinned tooling, DoD, bootstrap order |
| `frontend-plan` | `docs/frontend-refactor-plan.md` | Active frontend refactor epics/leaves |
| `frontend-plan-yaml` | `docs/frontend-refactor-plan.yaml` | Machine-readable plan for Phasical import |
| `frontend-gaps` | `docs/frontend-gap-analysis.md` | Gap analysis (input to refactor plan) |
| `roadmap-archived` | `docs/roadmap/README.md` | MVP roadmap archived — use Phasical |
| `adr-0001` | `docs/adr/0001-atlas-declarative-schema.md` | Atlas declarative schema decision |

## Verification commands

| Scope | Command |
| ----- | ------- |
| Default lint | `pnpm turbo run lint --filter=...` (pending Phase 0 scaffold) |
| Default typecheck | `pnpm turbo run typecheck --filter=...` |
| Default test | `pnpm turbo run test:unit --filter=...` |
| Go (full) | `bash scripts/go-test.sh` |
| Go (scoped) | `cd services/<svc> && go test ./...` |

Map committed paths → package filters per `.cursor/rules/06-local-ci-before-commit.mdc`.

## Phase gates (optional)

| Gate | Blocks |
| ---- | ------ |
| `phase-0` | Phases 1–5 until monorepo scaffold + compose + CI skeleton land |

## Hot files (never parallelize)

- `docs/frontend-refactor-plan.yaml` — source of truth for frontend refactor backlog; human index in `frontend-refactor-plan.md`
- `docs/import-frontend-refactor-plan.py` — Phasical API importer; run `--dry-run` before `--apply`
