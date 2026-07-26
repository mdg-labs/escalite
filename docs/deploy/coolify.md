# Deploy Escalite on Coolify

Escalite Community Edition is a **single-org, single-tenant** instance per deployment (see `docs/specs/04-licensing-and-editions.md` — tenant isolation). Coolify is a supported PaaS path for operators who want managed TLS, git-based deploys, and a web UI without maintaining their own reverse proxy.

This guide covers environment variables, TLS via the Coolify proxy, and a manual validation checklist. For backups, upgrades, and resource tuning, see [production.md](production.md).

## Prerequisites

| Requirement | Notes |
| ----------- | ----- |
| Coolify **4.x** server | Tested against Coolify 4.1.2; any 4.x instance with Docker Compose support should work |
| Target server | ≥ 2 CPU cores, ≥ 4 GB RAM (see [resource limits](production.md#resource-limits)) |
| Git source | Public or Coolify-connected private repo (`mdg-labs/escalite`) |
| DNS | `A`/`AAAA` record pointing your domain at the Coolify server |
| Encryption key | `openssl rand -hex 32` — generate once, store in Coolify secrets |

## Architecture

Coolify runs the same Compose stack as bare-metal Docker Compose. Only the **web** service should receive a public domain; `postgres`, `api`, and `engine` stay on the internal Docker network.

```text
Internet ──► Coolify proxy (Traefik, TLS) ──► web:5173
                                              │
                                              ├─► /api/*, /graphql → api:8080 (internal)
                                              │
postgres:5432 ◄── api, engine (internal only)
```

The `web` container (nginx) proxies `/api/` and `/graphql` to the API service. Coolify terminates TLS and forwards `X-Forwarded-Proto: https` so the app sees the correct public origin when `ESCALITE_APP_ORIGIN` is set.

## Deployment procedure

### 1. Create a Coolify project

1. In Coolify, create a new **Project** (e.g. `Escalite`).
2. Add a **Docker Compose** resource connected to the Escalite git repository.
3. Set the **base directory** to `deploy/docker-compose`.

### 2. Configure compose files

| Setting | Value |
| ------- | ----- |
| Compose file(s) | `docker-compose.yml`, `docker-compose.prod.yml` |
| Profile | `prod` |
| Build target | Set `ESCALITE_DOCKER_TARGET=prod` (see env vars below) |

For **registry-based** deploys (digest-pinned images from GHCR), add `docker-compose.release.yml` as a third file and set the `ESCALITE_*_IMAGE` variables. See [compose README](../../deploy/docker-compose/README.md#image-digest-pinning-release).

### 3. Set environment variables

Copy the [required variables](#environment-variable-matrix) into Coolify's environment editor. At minimum:

```bash
ESCALITE_DOCKER_TARGET=prod
ESCALITE_ENCRYPTION_KEY=<openssl rand -hex 32>
POSTGRES_PASSWORD=<strong-random-password>
ESCALITE_DATABASE_URL=postgres://escalite:<same-password>@postgres:5432/escalite?sslmode=disable
ESCALITE_APP_ORIGIN=https://escalite.example.com
```

`ESCALITE_DATABASE_URL` **must** use the same password as `POSTGRES_PASSWORD` and the internal hostname `postgres` (not `localhost`).

### 4. Expose only the web service

1. In Coolify, assign your FQDN (e.g. `escalite.example.com`) to the **web** service.
2. Set the container port to **5173**.
3. Enable **HTTPS** (Let's Encrypt) — Coolify handles certificate issuance and renewal.
4. Do **not** assign public domains or published ports to `postgres`, `api`, or `engine`.

### 5. Deploy

Trigger a deploy from Coolify. On first boot:

- Postgres starts and becomes healthy.
- API runs Atlas migrations automatically, then serves `/healthz`.
- Engine and web start after API is healthy.

Monitor logs in Coolify until all four services report healthy.

### 6. Post-deploy verification

Open `https://escalite.example.com` and complete the [manual test checklist](#manual-test-checklist) below.

## TLS via Coolify

Escalite does not terminate TLS inside the application containers. Coolify's Traefik proxy:

- Obtains and renews Let's Encrypt certificates for the assigned domain.
- Redirects HTTP → HTTPS.
- Forwards proxied requests to `web:5173`.

Set `ESCALITE_APP_ORIGIN` to the **HTTPS** public URL (e.g. `https://escalite.example.com`). This value is used for password-reset links, OAuth redirects, and CORS.

The optional in-compose `caddy` service remains commented out — Coolify replaces it.

## Single-org instance

Each Coolify deployment runs one isolated CE instance with one default organization. This matches the CE tenancy model: Cloud provisions **separate** CE instances per customer rather than multi-tenancy inside CE (`04-licensing-and-editions.md`). Do not run multiple unrelated teams on a single CE instance until multi-org ships (Phase 5 roadmap).

## Environment variable matrix

Variables consumed by the Compose stack. Set these in Coolify's environment UI (or a linked secrets store).

### Required

| Variable | Example | Used by | Notes |
| -------- | ------- | ------- | ----- |
| `ESCALITE_DOCKER_TARGET` | `prod` | api, engine, web | Selects prod Dockerfile stage (distroless / unprivileged nginx) |
| `ESCALITE_ENCRYPTION_KEY` | `a1b2…` (64 hex chars) | api, engine | 32-byte AES-256-GCM key; API refuses to start if empty |
| `POSTGRES_PASSWORD` | `<random>` | postgres | Change from default `escalite` |
| `ESCALITE_DATABASE_URL` | `postgres://escalite:<pw>@postgres:5432/escalite?sslmode=disable` | api, engine | Password must match `POSTGRES_PASSWORD`; host must be `postgres` |
| `ESCALITE_APP_ORIGIN` | `https://escalite.example.com` | api, web | Public URL users open; must match Coolify domain + `https` |

### Recommended

| Variable | Default | Used by | Notes |
| -------- | ------- | ------- | ----- |
| `POSTGRES_USER` | `escalite` | postgres | |
| `POSTGRES_DB` | `escalite` | postgres | |
| `ESCALITE_LOG_LEVEL` | `info` | api, engine | `debug` for troubleshooting only |

### Release deploy only (with `docker-compose.release.yml`)

| Variable | Used by | Notes |
| -------- | ------- | ----- |
| `ESCALITE_API_IMAGE` | api | Digest-pinned GHCR reference (`@sha256:…`) |
| `ESCALITE_ENGINE_IMAGE` | engine | Digest-pinned GHCR reference |
| `ESCALITE_WEB_IMAGE` | web | Digest-pinned GHCR reference |
| `ESCALITE_POSTGRES_IMAGE` | postgres | Optional; default pinned in release compose |

### Optional integrations

Set only when enabling the feature. Full list in [`.env.example`](../../.env.example).

| Group | Variables |
| ----- | --------- |
| OIDC / SSO | `ESCALITE_OIDC_ISSUER_URL`, `ESCALITE_OIDC_CLIENT_ID`, `ESCALITE_OIDC_CLIENT_SECRET` |
| SMTP | `ESCALITE_SMTP_HOST`, `ESCALITE_SMTP_PORT`, `ESCALITE_SMTP_USERNAME`, `ESCALITE_SMTP_PASSWORD`, `ESCALITE_SMTP_FROM` |
| Inbound email | `ESCALITE_INBOUND_EMAIL_RELAY_SECRET`, `ESCALITE_INBOUND_EMAIL_DOMAIN`, `ESCALITE_INBOUND_EMAIL_REQUIRE_AUTHENTICATED` |
| Twilio SMS/voice | `ESCALITE_SMS_PROVIDER`, `ESCALITE_TWILIO_ACCOUNT_SID`, `ESCALITE_TWILIO_AUTH_TOKEN`, `ESCALITE_TWILIO_FROM_NUMBER`, `ESCALITE_TWILIO_VOICE_FROM_NUMBER` |
| Slack OAuth | `ESCALITE_SLACK_CLIENT_ID`, `ESCALITE_SLACK_CLIENT_SECRET`, `ESCALITE_SLACK_REDIRECT_URL`, `ESCALITE_SLACK_SCOPES`, `ESCALITE_SLACK_SIGNING_SECRET` |
| Feature flags | `ESCALITE_FEATURE_*` |

### Do not set on Coolify (internal / dev-only)

| Variable | Reason |
| -------- | ------ |
| `ESCALITE_API_PORT`, `ESCALITE_ENGINE_PORT`, `ESCALITE_WEB_PORT`, `POSTGRES_PORT` | Host port mappings; Coolify routes to container ports directly. Leave unset so services use internal ports only. |
| `ESCALITE_API_PROXY_TARGET` | Hard-coded in compose for `web` → `api` internal proxy |

## Manual test checklist

Recorded during task `p4-coolify-validation` (session `EL129-20260726-b5d2`). Status key: **pass** = verified with evidence; **pending** = requires operator Coolify instance.

| # | Check | Status | Evidence / notes |
| - | ----- | ------ | ---------------- |
| 1 | Compose config validates (dev profile) | **pass** | `ESCALITE_DOCKER_TARGET=dev docker compose --profile dev config` exits 0 |
| 2 | Compose config validates (prod profile) | **pass** | `docker compose -f docker-compose.yml -f docker-compose.prod.yml --profile prod config` exits 0 |
| 3 | Compose config validates (release profile) | **pass** | Three-file merge with dummy digest-pinned images exits 0 (matches `task compose:config`) |
| 4 | Prod stack starts with required env vars | **pass** | `docker compose … --profile prod up -d` — all four services created; `ESCALITE_ENCRYPTION_KEY` + matching `ESCALITE_DATABASE_URL` required |
| 5 | Postgres reaches healthy state | **pass** | `escalite-postgres-1` healthy within ~10 s |
| 6 | API `/healthz` returns `{"status":"ok"}` | **pass** | `curl -fsS http://localhost:58080/healthz` after local prod smoke (2026-07-26) |
| 7 | Engine `/healthz` returns `{"status":"ok"}` | **pass** | `curl -fsS http://localhost:58081/healthz` |
| 8 | Web `/healthz` returns `ok` | **pass** | nginx `location = /healthz` returns 200 |
| 9 | Web UI loads at `/` | **pass** | `curl` returns Escalite HTML shell (HTTP 200) |
| 10 | API migrations run on first boot | **pass** | API log shows migration success when DB credentials match; fails fast with `password authentication failed` when mismatched (verified during smoke) |
| 11 | `web` proxies `/api/` to internal API | **pass** | Verified via `deploy/docker-compose/nginx/web.conf` — `proxy_pass http://api:8080` |
| 12 | Only `web` exposed publicly on Coolify | **pending** | **Operator:** confirm no public domain/port on `postgres`, `api`, or `engine` in Coolify service settings |
| 13 | HTTPS certificate issued (Let's Encrypt) | **pending** | **Operator:** open `https://<domain>` — browser shows valid cert; Coolify proxy logs show ACME success |
| 14 | `ESCALITE_APP_ORIGIN` matches public HTTPS URL | **pending** | **Operator:** set to `https://<domain>`; verify password-reset or auth redirect URLs use HTTPS (not `http://localhost`) |
| 15 | GraphQL endpoint reachable via public URL | **pending** | **Operator:** `curl -fsS https://<domain>/graphql -H 'Content-Type: application/json' -d '{"query":"{ __typename }"}'` |
| 16 | User can register / log in via public URL | **pending** | **Operator:** complete first-user signup or OIDC login through the HTTPS domain |
| 17 | Data persists across container restart | **pending** | **Operator:** create a test alert, `docker compose restart` (or Coolify redeploy without volume wipe), confirm data remains |
| 18 | Single-org: one default organization | **pending** | **Operator:** confirm fresh instance has one org; no cross-tenant data from other deployments (inherent per-instance isolation) |
| 19 | Coolify git-push redeploy succeeds | **pending** | **Operator:** push a no-op tag or trigger manual redeploy; all services return healthy |
| 20 | Resource limits applied (prod overlay) | **pass** | `docker-compose.prod.yml` sets `mem_limit` / `cpus` per service (postgres 1G, api/engine 512M, web 256M) |

### Operator steps for pending items

1. Create a Coolify project on a reachable server (staging or production).
2. Add Docker Compose resource → repo `mdg-labs/escalite`, base dir `deploy/docker-compose`, files `docker-compose.yml` + `docker-compose.prod.yml`, profile `prod`.
3. Set required env vars (see [§ 3](#3-set-environment-variables)).
4. Assign domain + HTTPS to **web** port **5173** only.
5. Deploy and walk checklist rows 12–19.
6. Record results in this table (or a linked runbook) for your environment.

## Related docs

- [Production deploy, backups, upgrades](production.md)
- [Compose profiles and digest-pinned release images](../../deploy/docker-compose/README.md)
- [Security and auth (TLS guidance)](../specs/07-security-and-auth.md)
