# Escalite

Open-core, self-hostable on-call, alerting, and incident-response platform (AGPL Community Edition).

## Deployment

Production is a **standalone Docker Compose template** — pull GHCR images only. **No git clone.**

### Docker Compose (VPS / bare metal)

```bash
mkdir escalite && cd escalite
curl -fsSL -o docker-compose.yml https://raw.githubusercontent.com/mdg-labs/escalite/main/docker-compose.yml
curl -fsSL -o .env https://raw.githubusercontent.com/mdg-labs/escalite/main/docker-compose.env.example
# Edit .env: ESCALITE_ENCRYPTION_KEY (openssl rand -hex 32), POSTGRES_PASSWORD, ESCALITE_DATABASE_URL, ESCALITE_APP_ORIGIN
docker compose pull && docker compose up -d
```

### Coolify

Create a **Docker Compose** resource, use the compose file from the [raw URL](https://raw.githubusercontent.com/mdg-labs/escalite/main/docker-compose.yml) or paste [`docker-compose.yml`](docker-compose.yml), set env vars in the Coolify UI — see [Coolify guide](docs/deploy/coolify.md).

Expose only **web** (port 5173) publicly for TLS. Pin `ESCALITE_*_IMAGE` to a release tag instead of `:latest` for production — [production ops](docs/deploy/production.md).

### Developers (build from source)

```bash
cp .env.example deploy/docker-compose/.env
# Set ESCALITE_ENCRYPTION_KEY in deploy/docker-compose/.env
cd deploy/docker-compose
docker compose -f docker-compose.yml -f docker-compose.prod.yml --profile prod up -d --build
```

From the repo root: `task compose:prod:build` then `task compose:prod -- -d`.

Operator guides:

- [Production deploy, backups, and upgrades](docs/deploy/production.md)
- [Coolify deployment (env vars, TLS, validation)](docs/deploy/coolify.md)
- [Compose profiles and digest-pinned release images](deploy/docker-compose/README.md)

## Prerequisites

Install these tools before developing locally:

| Tool | Version | Notes |
| ---- | ------- | ----- |
| [Git](https://git-scm.com/) | latest | Clone and contribute |
| [Node.js](https://nodejs.org/) | 20 LTS or newer | JavaScript toolchain |
| [pnpm](https://pnpm.io/) | 9.x | Monorepo package manager (`corepack enable` recommended) |
| [Go](https://go.dev/) | ≥ 1.22 | API, engine, and integrations services |
| [Task](https://taskfile.dev/) | 3.x | Root task runner (`go install github.com/go-task/task/v3/cmd/task@latest`) |
| [pg-schema-diff](https://github.com/stripe/pg-schema-diff) | v1.0.7 | Migration generation (`go install github.com/stripe/pg-schema-diff/cmd/pg-schema-diff@v1.0.7`) |
| [goose](https://github.com/pressly/goose) | v3 | Migration apply at dev/startup (via `task migrate`) |

Docker is required for the full local stack via `deploy/docker-compose/` (see [Deployment](#deployment)).

## Docker Compose stack (development)

```bash
cp .env.example deploy/docker-compose/.env
task compose:dev
```

Production-style local images (distroless / unprivileged nginx, non-root, resource limits, healthchecks):

```bash
task compose:prod:build
task compose:prod
```

Release deployments with digest-pinned registry images: see [`deploy/docker-compose/README.md`](deploy/docker-compose/README.md) and [production ops](docs/deploy/production.md).

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
task migrate   # Apply Postgres migrations with goose (requires DATABASE_URL)
task schema:diff -- <name>  # Generate migration from schema/sql (pg-schema-diff)
task compose:dev        # Docker Compose dev profile
task compose:prod:build # Build prod-profile images locally
task compose:prod       # Run prod-profile stack
task compose:config     # Validate compose YAML (dev + prod)
```

Set `DATABASE_URL` or `ESCALITE_DATABASE_URL` before running `task migrate`.

## Documentation

- Spec index: [`docs/specs/README.md`](docs/specs/README.md)
- Frontend refactor plan: [`docs/frontend-refactor-plan.md`](docs/frontend-refactor-plan.md) (Phasical import via `docs/import-frontend-refactor-plan.py`)
