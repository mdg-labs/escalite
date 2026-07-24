# 07 — Security, AuthN/AuthZ & Hardening

**Outline URL:** https://outline.mdg-labs.dev/doc/07-security-authnauthz-hardening-a7Jjcl2IhR  
**Outline document id:** `f88f431b-874d-46cb-880d-3091734ca36c`

MVP auth uses email+password (argon2id) with server-side Postgres sessions (HttpOnly cookies), optional generic OIDC, and mobile device-bound refresh tokens. RBAC is minimal: org `admin` vs `member`, team scoping enforced server-side on every mutation. Inbound webhooks use per-IntegrationKey URL tokens (≥128-bit), optional HMAC, rate limiting (default 120 req/min per key), 256 KB payload cap, and no secrets in logs. Provider credentials are AES-256-GCM encrypted at rest with `ESCALITE_ENCRYPTION_KEY`. Transport hardening, CSRF/CORS, GraphQL depth limits, audit_events from day 1, and CI vuln scanning (govulncheck, pnpm audit) are required. SAML/SCIM and Vault integration are explicitly post–Phase 5 / out of MVP.

> **Source of truth:** Outline MCP (collection: Escalite, doc id: `f88f431b-874d-46cb-880d-3091734ca36c`). Fetch via Outline MCP before relying on this doc's content — this file is a pointer, not a mirror.
