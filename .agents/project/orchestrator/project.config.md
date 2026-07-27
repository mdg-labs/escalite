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
| Plan file | `docs/frontend-refactor-plan.md` |
| Plan file (machine) | `docs/frontend-refactor-plan.yaml` |
| Archived MVP roadmap | Phasical + GitHub issues; see `docs/roadmap/README.md` |
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
| `frontend-refactor` | Frontend operator console refactor (see `docs/frontend-refactor-plan.yaml`) |

## Area prefixes (titles)

- `fe-` — Frontend refactor plan (`docs/frontend-refactor-plan.yaml`)
- `p0-` … `p5-` — **Archived** MVP roadmap (Phasical/GitHub only; do not use for new work)

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
