<p align="center">
  <img src="assets/escalite_app_icon_w_dark_blue_bg.svg" width="120" alt="Escalite app icon" />
</p>

<h1 align="center">Escalite</h1>

<p align="center">
  Open-core, self-hostable on-call, alerting, and incident-response platform.<br />
  Community Edition · AGPL · built for teams who want GoAlert-grade reliability with a modern operator experience.
</p>

<p align="center">
  <a href="docs/specs/README.md">Specs</a> ·
  <a href="docs/deploy/production.md">Deploy</a> ·
  <a href="CONTRIBUTING.md">Contributing</a> ·
  <a href="LICENSE">License</a>
</p>

---

## Why Escalite

Escalite targets platform and SRE teams that need **reliable paging**, **on-call scheduling**, and **incident workflows** without surrendering control to a SaaS-only vendor. Run it on your own infrastructure with Docker Compose, integrate via GraphQL, and extend through an open-core model.

| Pillar | What you get |
| ------ | -------------- |
| Alerting | Escalation policies, dedup, heartbeats, inbound integrations |
| On-call | Schedules, rotations, overrides, mobile push (incl. iOS Critical Alerts path) |
| Incidents | Timeline, roles, postmortem export |
| Ops UX | Dense dark-mode web console, status pages, self-hosted from day one |

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

## Quick start (developers)

```bash
git clone https://github.com/mdg-labs/escalite.git
cd escalite
pnpm install
task dev
```

`task dev` starts Turborepo `dev` tasks across JS workspaces. For the full stack (Postgres, API, engine, web), use `task compose:dev`.

### Prerequisites

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

### Docker Compose stack (development)

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

## Monorepo layout

| Path | Purpose |
| ---- | ------- |
| `apps/web` | Operator web console (React, Vite, COSS UI) |
| `apps/mobile` | Minimal Expo app — push, ack, escalate |
| `apps/status-page` | Public status page frontend |
| `services/api` | GraphQL API (Go, chi, pgx, sqlc) |
| `services/engine` | Alerting engine and on-call compute |
| `services/integrations` | Inbound/outbound integration workers |
| `packages/` | Shared schema, types, UI primitives, tokens |
| `deploy/` | Docker Compose and deployment assets |
| `assets/` | Canonical branding (app icon, logo marks) |

Regenerate raster icons and favicons from SVG sources:

```bash
node scripts/generate-branding-assets.mjs
```

See [`assets/README.md`](assets/README.md) for which file to use where.

## Common tasks

```bash
task dev       # Start JS dev workflows (Turborepo)
task build     # Build JS workspaces and Go services
task test      # Run JS and Go tests
task lint      # Lint JS workspaces
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
- Frontend refactor plan: [`docs/frontend-refactor-plan.md`](docs/frontend-refactor-plan.md)
- Mobile releases: [`docs/deploy/mobile-releases.md`](docs/deploy/mobile-releases.md)

## Repository activity

![Repobeats analytics](https://repobeats.axiom.co/api/embed/1227279aebf7b43b60bddb00b675e5f853fe4fa4.svg)
