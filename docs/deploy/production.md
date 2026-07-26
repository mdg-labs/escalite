# Production deployment (single-node Docker Compose)

Escalite's MVP deployment path is a single-node Docker Compose stack. TLS termination, backups, and upgrades are operator responsibilities; this guide covers the recommended procedures.

## Prerequisites

- Docker Engine 24+ with the Compose plugin (`docker compose version`)
- A host with at least 2 CPU cores and 4 GB RAM (see [resource limits](#resource-limits))
- A 32-byte encryption key: `openssl rand -hex 32`

## Initial deploy

From a clean checkout:

```bash
git clone https://github.com/mdg-labs/escalite.git
cd escalite
cp .env.example deploy/docker-compose/.env
```

Edit `deploy/docker-compose/.env` and set at minimum:

- `ESCALITE_ENCRYPTION_KEY` — required; the API refuses to start without it
- `POSTGRES_PASSWORD` — change from the example default
- `ESCALITE_APP_ORIGIN` — public URL users will open (e.g. `https://escalite.example.com`)

Build and start the production profile:

```bash
cd deploy/docker-compose
docker compose -f docker-compose.yml -f docker-compose.prod.yml --profile prod up -d --build
```

Or from the repo root:

```bash
task compose:prod:build
task compose:prod -- -d
```

Wait until all services are healthy:

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml --profile prod ps
```

Open the web UI at the host port mapped by `ESCALITE_WEB_PORT` (default `5173`). Place a reverse proxy (Caddy, Traefik, nginx, or a PaaS such as Coolify) in front for TLS — see `docs/specs/07-security-and-auth.md`.

### Release images (digest-pinned)

For registry-based deploys instead of local builds, see [`deploy/docker-compose/README.md`](../../deploy/docker-compose/README.md#image-digest-pinning-release).

## Resource limits

`docker-compose.prod.yml` sets conservative per-service limits suitable for a small team on a single VPS:

| Service  | Memory | CPUs |
| -------- | ------ | ---- |
| postgres | 1 GB   | 1.0  |
| api      | 512 MB | 1.0  |
| engine   | 512 MB | 1.0  |
| web      | 256 MB | 0.5  |

Adjust `mem_limit`, `cpus`, and `deploy.resources.limits` in `docker-compose.prod.yml` for larger installs. All services use `restart: unless-stopped` and HTTP healthchecks so Compose can restart unhealthy containers and honour `depends_on` conditions.

## Backups

Escalite stores all application state in PostgreSQL. Back up the database volume regularly.

### Logical backup (recommended)

While the stack is running:

```bash
cd deploy/docker-compose
docker compose -f docker-compose.yml -f docker-compose.prod.yml --profile prod \
  exec -T postgres pg_dump -U "${POSTGRES_USER:-escalite}" -d "${POSTGRES_DB:-escalite}" \
  --format=custom > "escalite-$(date +%Y%m%d-%H%M%S).dump"
```

Store dumps off-host (object storage, another machine). Test restores on a staging instance before relying on a backup schedule.

### Restore

```bash
# Stop app services so nothing writes during restore
docker compose -f docker-compose.yml -f docker-compose.prod.yml --profile prod stop api engine web

docker compose -f docker-compose.yml -f docker-compose.prod.yml --profile prod \
  exec -T postgres pg_restore -U "${POSTGRES_USER:-escalite}" -d "${POSTGRES_DB:-escalite}" \
  --clean --if-exists < escalite-YYYYMMDD-HHMMSS.dump

docker compose -f docker-compose.yml -f docker-compose.prod.yml --profile prod start api engine web
```

### Volume snapshot

The named volume `postgres_data` can also be snapshotted at the filesystem or cloud-provider level. Stop Postgres first for a crash-consistent snapshot, or use `pg_dump` for a portable logical backup.

## Upgrades

1. **Back up** the database (see above).
2. **Pull or build** new images:
   - Local build: `task compose:prod:build`
   - Registry release: update digest-pinned `ESCALITE_*_IMAGE` values in `.env` (see compose README).
3. **Review** release notes and `.env.example` for new required variables.
4. **Apply migrations** — the API container runs Atlas migrations on startup; ensure only one API instance migrates during rolling upgrades on a single node.
5. **Recreate containers**:

   ```bash
   cd deploy/docker-compose
   docker compose -f docker-compose.yml -f docker-compose.prod.yml --profile prod up -d
   ```

6. **Verify** health and smoke-test login:

   ```bash
   docker compose -f docker-compose.yml -f docker-compose.prod.yml --profile prod ps
   curl -fsS "http://localhost:${ESCALITE_API_PORT:-8080}/healthz"
   ```

To roll back, restore the database backup and redeploy the previous image digests or tags.

## Supply chain

Release builds publish SPDX SBOMs (syft) attached to GitHub Releases. Pin runtime images to `@sha256:…` digests in production — details in [`deploy/docker-compose/README.md`](../../deploy/docker-compose/README.md).
