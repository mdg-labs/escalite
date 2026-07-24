# Doc index — Escalite

> Local spec shorthand map. Expand per project.

## Spec documents

| Shorthand | Path | Topics |
| --------- | ---- | ------ |
| `spec` | `docs/specs/README.md` | Spec index, Outline pointers |
| `spec-00` | `docs/specs/00-vision-and-scope.md` | Vision, MVP scope, non-goals |
| `spec-01` | `docs/specs/01-architecture-and-monorepo.md` | Stack, monorepo layout, deployment |
| `spec-08` | `docs/specs/08-implementation-decisions-and-conventions.md` | Pinned tooling, DoD, bootstrap order |
| `roadmap` | `docs/roadmap/ROADMAP.md` | Phases, task IDs (generated) |
| `roadmap-yaml` | `docs/roadmap/roadmap.yaml` | Machine-readable roadmap source |
| `adr-0001` | `docs/adr/0001-atlas-declarative-schema.md` | Atlas declarative schema decision |

## Verification commands

| Scope | Command |
| ----- | ------- |
| Default lint | `pnpm turbo run lint --filter=...` (pending Phase 0 scaffold) |
| Default typecheck | `pnpm turbo run typecheck --filter=...` |
| Default test | `pnpm turbo run test:unit --filter=...` |
| Go API | `cd services/api && go test ./...` (pending scaffold) |

Map committed paths → package filters per `.cursor/rules/06-local-ci-before-commit.mdc`.

## Phase gates (optional)

| Gate | Blocks |
| ---- | ------ |
| `phase-0` | Phases 1–5 until monorepo scaffold + compose + CI skeleton land |

## Hot files (never parallelize)

- `docs/roadmap/roadmap.yaml` — source of truth for roadmap; regenerate `ROADMAP.md` via `python docs/roadmap/generate_roadmap.py`
- `docs/roadmap/generate_roadmap.py` — roadmap generator; coordinate edits with `roadmap.yaml`
