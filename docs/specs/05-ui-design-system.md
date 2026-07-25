# 05 — UI / Design System (COSS-first)

**Outline URL:** https://outline.mdg-labs.dev/doc/05-ui-design-system-coss-first-KAsncgrCHZ  
**Outline document id:** `56839d7c-268d-4003-91c3-f25245a2e4af`

The UI stack is COSS UI (Base UI + Tailwind v4) in `packages/ui`, installed via `@coss/*` shadcn registry, with Tremor (or Recharts fallback) for analytics dashboards. Semantic design tokens live in `packages/tokens` (shared with mobile Tamagui). Design direction favors dense, scan-friendly layouts, dark mode as default, colorblind-safe severity coding, and realtime updates via GraphQL subscriptions. Domain composites live under `packages/ui/domain/`; see Outline for per-screen COSS particle selection guide.

> **Source of truth:** Outline MCP (collection: Escalite, doc id: `56839d7c-268d-4003-91c3-f25245a2e4af`). Fetch via Outline MCP before relying on this doc's content — this file is a pointer, not a mirror.
