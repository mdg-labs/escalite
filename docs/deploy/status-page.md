# Status page deployment

Public status pages are served by the `apps/status-page` Vite/React app. Each page loads data from the unauthenticated REST API (`GET /api/v1/public/status/{slug}`) and polls for updates.

## Compose service

The `status-page` service is included in the default Docker Compose stack:

```bash
cd deploy/docker-compose
cp ../../.env.example .env
docker compose --profile dev up
```

| Variable | Default | Purpose |
| -------- | ------- | ------- |
| `ESCALITE_STATUS_PAGE_PORT` | `5174` | Host port for the status page UI |
| `ESCALITE_STATUS_PAGE_FRAME_ANCESTORS` | `*` | Dev-server / operator override for `frame-ancestors` CSP |

Open a configured slug at `http://localhost:5174/{slug}` (for example `http://localhost:5174/acme-status` after creating a status page in Escalite).

## Custom domain and TLS (operator responsibility)

Escalite does not provision DNS or TLS certificates for customer-facing status domains. Operators point a hostname (for example `status.example.com`) at the `status-page` service and terminate HTTPS at a reverse proxy (Caddy, nginx, Cloudflare, etc.).

### Recommended setup

1. Create the status page in Escalite (admin UI / GraphQL) and note the **slug**.
2. Point DNS `A`/`AAAA` or `CNAME` for your public hostname to the load balancer or host running Compose.
3. Terminate TLS on the reverse proxy and forward HTTP to the `status-page` container on port `5174`.
4. Proxy `/api` to the Escalite API (`api:8080` inside the compose network) so the browser can call public status endpoints on the same origin.

### Path-based vs dedicated hostname

- **Dedicated hostname** (common): map `status.example.com` → status-page service; use slug routing (`/acme-status`) or rewrite `/` to a fixed slug if you host one org per domain.
- **Subpath on main app origin**: possible but not the default; prefer a separate hostname so CSP and caching stay isolated from the authenticated app.

### frame-ancestors CSP (doc 07)

The main Escalite web app sets `Content-Security-Policy: frame-ancestors 'none'` so the admin UI cannot be embedded in third-party sites.

The status page app uses a **distinct** policy so pages may be embedded (for example in a corporate intranet iframe):

| Layer | Policy |
| ----- | ------ |
| `status-page` nginx (prod image) | `frame-ancestors *` by default |
| Vite dev server | `frame-ancestors` from `ESCALITE_STATUS_PAGE_FRAME_ANCESTORS` |
| Public status API JSON responses | Per-org `frameAncestorsCsp` from status page settings (when set) |

To restrict embedding to specific parent origins, set `frameAncestorsCsp` in Escalite (for example `https://intranet.example.com`) and mirror that value at your reverse proxy for the HTML shell if you need strict enforcement beyond the API responses.

Example Caddy snippet:

```caddyfile
status.example.com {
    reverse_proxy status-page:5174
    header Content-Security-Policy "frame-ancestors https://intranet.example.com"
}
```

## Environment variables (runtime)

Set on the **status-page** container at deploy time (no image rebuild). Defaults work when the status UI and API share a public origin (nginx proxies `/api/` to `ESCALITE_API_UPSTREAM`).

| Variable | Default | Purpose |
| -------- | ------- | ------- |
| `ESCALITE_API_PUBLIC_URL` | *(empty → same origin)* | Public API origin when the UI and API are on different hosts |
| `ESCALITE_API_UPSTREAM` | `http://api:8080` | nginx `proxy_pass` target for `/api/` |
| `ESCALITE_STATUS_POLL_INTERVAL_MS` | `60000` | Poll interval for incident updates |

Build-time `VITE_*` variables remain supported as a fallback when `/runtime-config.js` is not injected.

## Email subscriptions

Visitors can subscribe via `POST /api/v1/public/status/{slug}/subscribe`. Delivery of notification email is handled by the Escalite API/worker stack once outbound SMTP is configured for the instance.
