# 05 — UI / Design System (COSS-first)

**Outline URL:** https://outline.mdg-labs.dev/doc/05-ui-design-system-coss-first-KAsncgrCHZ  
**Outline document id:** `56839d7c-268d-4003-91c3-f25245a2e4af`

The UI stack is shadcn/ui + Radix + Tailwind in `packages/ui`, with Tremor (or shadcn chart fallback) for analytics dashboards. Design direction favors dense, scan-friendly layouts, dark mode as default, colorblind-safe severity coding, and realtime updates via GraphQL subscriptions. Domain-specific composites (ScheduleCalendar, EscalationPolicyEditor, AlertCard, OnCallWidget) live under `packages/ui/domain/`; tokens are shared conceptually with mobile. Cal.com scheduling primitives are worth a spike for rotation calendar UI before a fully custom build.

> **Source of truth:** Outline MCP (collection: Escalite, doc id: `56839d7c-268d-4003-91c3-f25245a2e4af`). Fetch via Outline MCP before relying on this doc's content — this file is a pointer, not a mirror.
