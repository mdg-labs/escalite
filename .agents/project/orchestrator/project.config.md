# Project config — Escalite

> Supporting file — created by project-setup (layer **phasical**). Lives under `.agents/project/` — **not** inside `.agents/skills/` (`npx skills update` wipes skill directories).

## Repository

| Field | Value |
| ----- | ----- |
| Project name | Escalite |
| Repo path | /home/mdguggenbichler/projects/escalite |
| GitHub repo | `mdg-labs/escalite` |
| Integration branch | `dev` |
| Production branch | `main` — agents must not push here |
| Task branch (Lane P) | `orchestrator/<TASK-ID>` |
| Worktree (Lane P) | `../escalite-worktrees/orchestrator-<TASK-ID>` |
| Plan file | `docs/roadmap/ROADMAP.md` |
| Spec doc glob | `docs/specs/*.md` |

## Phasical

| Field | Value |
| ----- | ----- |
| MCP server | `user-phasical` |
| Workspace | MDG-Labs (`X3VbytvC7pKgazK2dAsOQIFtdGYRzdGH`) |
| Project | Escalite (`bkbnmftqdr54r9gcgrgtndbw`) |
| Ready status slug | `ready` |
| GitHub MCP (read) | `user-github` |

**Commits:** use GitHub `[#N]` from `externalLinks.externalId`. Never Phasical task IDs in git.

## Domain labels

| Label | Scope |
| ----- | ----- |
| `phase-0` | Foundations — repo scaffold, compose, CI skeleton |
| `phase-1` | Core alerting engine |
| `phase-2` | On-call scheduling & notifications |
| `phase-3` | Incident response |
| `phase-4` | Integrations & importers |
| `phase-5` | Status pages & polish |

## Area prefixes (titles)

- `p0-` — Phase 0 (Foundations)
- `p1-` — Phase 1 (Core alerting)
- `p2-` — Phase 2 (Scheduling & notifications)
- `p3-` — Phase 3 (Incident response)
- `p4-` — Phase 4 (Integrations)
- `p5-` — Phase 5 (Status pages)

## Commit conventions

- Phasical/GitHub tasks: `[#N]` in subject
- Roadmap-only (no Phasical mirror): `[P*-*]` in subject
- Body: `fixes #N` when project rules require it (see `.cursor/rules/`)

## Optional

| Field | Value |
| ----- | ----- |
| Multi-repo workspace | none |
| Slack session-end | not configured |
| Phase gates | see `doc-index.md` § Phase gates |
