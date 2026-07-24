# 01 — Architecture & Monorepo Layout

**Outline URL:** https://outline.mdg-labs.dev/doc/01-architecture-monorepo-layout-n6jv6GaYmH  
**Outline document id:** `52050af1-4739-4441-9a1e-a4e07e721301`

This doc pins the technical stack: Go backend, GraphQL primary API with REST for webhook ingestion, PostgreSQL, TypeScript/React web, Postgres-backed job queue (river), and a public AGPL monorepo layout (`apps/`, `services/`, `packages/`, `deploy/`). It recommends building a new codebase with GoAlert as reference (not a fork), documents realtime via LISTEN/NOTIFY → GraphQL subscriptions, and states that `docker compose up` is the only MVP-supported deployment path — Helm is cut, not deprioritized. A separate private `escalite-cloud` repo holds the control plane; the public repo is 100% CE with no license-boundary splits.

> **Source of truth:** Outline MCP (collection: Escalite, doc id: `52050af1-4739-4441-9a1e-a4e07e721301`). Fetch via Outline MCP before relying on this doc's content — this file is a pointer, not a mirror.
