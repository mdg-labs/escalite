# Escalite

Open-core, self-hostable on-call, alerting, and incident-response platform (AGPL Community Edition).

## Prerequisites

Install these tools before developing locally:

| Tool | Version | Notes |
| ---- | ------- | ----- |
| [Git](https://git-scm.com/) | latest | Clone and contribute |
| [Node.js](https://nodejs.org/) | 20 LTS or newer | JavaScript toolchain |
| [pnpm](https://pnpm.io/) | 9.x | Monorepo package manager (`corepack enable` recommended) |
| [Go](https://go.dev/) | ≥ 1.22 | API, engine, and integrations services |
| [Task](https://taskfile.dev/) | 3.x | Root task runner (`go install github.com/go-task/task/v3/cmd/task@latest`) |

Docker is required for the full local stack once `deploy/docker-compose/` lands (Phase 0, task #36).

## Quick start

```bash
git clone https://github.com/mdg-labs/escalite.git
cd escalite
pnpm install
task dev
```

`task dev` starts Turborepo `dev` tasks across JS workspaces. Go services and Docker Compose profiles are wired in follow-on Phase 0 tasks.

## Monorepo layout

| Path | Purpose |
| ---- | ------- |
| `apps/` | User-facing applications (`apps/web`, later `apps/mobile`) |
| `services/` | Go microservices (`api`, `engine`, `integrations`) |
| `packages/` | Shared libraries (schema, config, TypeScript types) |
| `deploy/` | Docker Compose and deployment assets |

## Common tasks

```bash
task dev       # Start JS dev workflows (Turborepo)
task build     # Build JS workspaces and Go services
task test      # Run JS and Go tests
task lint      # Lint JS workspaces
task migrate   # Apply Postgres migrations (requires DATABASE_URL)
```

Set `DATABASE_URL` or `ESCALITE_DATABASE_URL` before running `task migrate`.

## Documentation

- Spec index: [`docs/specs/README.md`](docs/specs/README.md)
- Roadmap: [`docs/roadmap/ROADMAP.md`](docs/roadmap/ROADMAP.md)
