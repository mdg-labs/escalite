# 08 — Implementation Decisions & Conventions (agent-binding)

**Outline URL:** https://outline.mdg-labs.dev/doc/08-implementation-decisions-conventions-agent-binding-VCZBeGw3HB  
**Outline document id:** `19d87fc5-7bbd-45ba-be42-f40eeb253783`

This doc pins every previously open "X or Y" choice for autonomous implementers: chi, gqlgen, pgx+sqlc, goose, river, pnpm, Turborepo, Vite, urql, Expo, GitHub Actions, UUIDv7, slog, and more. It defines API error codes, naming conventions, env prefix `ESCALITE_`, conventional commits with DCO, and ADR requirements for deviations. The Definition of Done requires code, tests (integration Postgres for timing/scheduling/auth), working `docker compose up`, docs/ADR updates, and clean lint/vuln scans. Phase 0 bootstrap order is explicit: scaffold → compose → schema → auth → GraphQL/codegen/web shell → river → E2E smoke.

> **Source of truth:** Outline MCP (collection: Escalite, doc id: `19d87fc5-7bbd-45ba-be42-f40eeb253783`). Fetch via Outline MCP before relying on this doc's content — this file is a pointer, not a mirror.
