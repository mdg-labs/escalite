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

Docker is required for the full local stack via `deploy/docker-compose/` (see below).

## Docker Compose stack

```bash
cp .env.example deploy/docker-compose/.env
task compose:dev
```

Production-style local images (distroless / unprivileged nginx, non-root):

```bash
task compose:prod:build
task compose:prod
```

Release deployments with digest-pinned registry images: see [`deploy/docker-compose/README.md`](deploy/docker-compose/README.md).

## Quick start

```bash
git clone https://github.com/mdg-labs/escalite.git
cd escalite
pnpm install
task dev
```

`task dev` starts Turborepo `dev` tasks across JS workspaces. For the full stack (Postgres, API, engine, web), use `task compose:dev`.

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
task lint       # Lint JS workspaces
task migrate   # Apply Postgres migrations (requires DATABASE_URL)
task compose:dev        # Docker Compose dev profile
task compose:prod:build # Build prod-profile images locally
task compose:prod       # Run prod-profile stack
task compose:config     # Validate compose YAML (dev + prod)
```

Set `DATABASE_URL` or `ESCALITE_DATABASE_URL` before running `task migrate`.

## Documentation

- Spec index: [`docs/specs/README.md`](docs/specs/README.md)
- Roadmap: [`docs/roadmap/ROADMAP.md`](docs/roadmap/ROADMAP.md)
