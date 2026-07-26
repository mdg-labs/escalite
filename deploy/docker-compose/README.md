# Docker Compose deployment

Local development and production-style stacks for Escalite (MVP path).

## Quick start

```bash
cp ../../.env.example .env
task compose:dev          # from repo root — hot-reload dev profile
task compose:prod:build   # build prod-target images locally
task compose:prod         # run prod profile (no bind-mounts)
```

## Compose files

| File | Purpose |
| ---- | ------- |
| `docker-compose.yml` | Base stack (Postgres, api, engine, web) with dev + prod profiles, healthchecks, and `restart: unless-stopped` |
| `docker-compose.prod.yml` | Prod overrides — removes dev bind-mounts, sets resource limits, and distroless/wget health probes |
| `docker-compose.release.yml` | Release overrides — pull digest-pinned images from a registry (no local build) |

Operator procedures (backups, upgrades, resource tuning): [`docs/deploy/production.md`](../../docs/deploy/production.md).

Merge order for release deploys:

```bash
docker compose \
  -f docker-compose.yml \
  -f docker-compose.prod.yml \
  -f docker-compose.release.yml \
  --profile prod up -d
```

## Image digest pinning (release)

Per `07-security-and-auth` (dependency & supply-chain hygiene), production deploys should pin every runtime image to an immutable digest:

```text
ghcr.io/mdg-labs/escalite-api@sha256:abc123…
```

### Workflow

1. **Build and push** versioned images from CI (e.g. `:v0.1.0` or a git SHA tag).
2. **Resolve the manifest digest** after push:

   ```bash
   docker buildx imagetools inspect ghcr.io/mdg-labs/escalite-api:v0.1.0 \
     --format '{{.Manifest.Digest}}'
   ```

3. **Set environment variables** in `.env` (see root `.env.example`):

   ```bash
   ESCALITE_API_IMAGE=ghcr.io/mdg-labs/escalite-api@sha256:<digest>
   ESCALITE_ENGINE_IMAGE=ghcr.io/mdg-labs/escalite-engine@sha256:<digest>
   ESCALITE_WEB_IMAGE=ghcr.io/mdg-labs/escalite-web@sha256:<digest>
   ESCALITE_POSTGRES_IMAGE=postgres:16-alpine@sha256:<digest>  # optional; default pinned in release compose
   ```

4. **Deploy** with the release override (step above). Compose pulls by digest; retagging upstream cannot change what runs.

### Updating digests

Re-run step 2 after each image rebuild. Update operator `.env` or your secrets store; verify with `docker compose … config` before `up`.

### Dockerfile base pins

Service Dockerfiles pin builder and runtime base images (`@sha256:…` on `FROM` lines). Bump those pins when upgrading Go, Node, distroless, or nginx bases — rebuild and re-record application image digests for release compose.

## Container users (non-root)

| Service | Prod runtime | UID |
| ------- | ------------ | --- |
| api | `gcr.io/distroless/static-debian12:nonroot` | 65532 |
| engine | `gcr.io/distroless/static-debian12:nonroot` | 65532 |
| web | `nginxinc/nginx-unprivileged` | 101 (nginx) |
