# 02 — Core Domain & Feature Spec

**Outline URL:** https://outline.mdg-labs.dev/doc/02-core-domain-feature-spec-mxzObn2e2D  
**Outline document id:** `3aba6e39-b5eb-4e89-b444-fdd59c38215c`

The domain model covers Organization → Team → Service (with IntegrationKey, HeartbeatMonitor, EscalationPolicy) and the first-class Incident aggregate (alerts, timeline, roles, postmortem) as the main departure from GoAlert. It specifies escalation engine semantics, RRULE-based scheduling with overrides, heartbeat dead-man's-switch monitors, compile-time inbound/outbound plugin registries, day-1 notification channels (push, email, webhook, Slack, later Twilio SMS/voice), deduplication, maintenance windows, and integration presets (e.g. Beszel via generic-webhook). Alert lifecycle includes dedup_key, auto-resolve from integrations, and priority levels kept simple (low/high).

> **Source of truth:** Outline MCP (collection: Escalite, doc id: `3aba6e39-b5eb-4e89-b444-fdd59c38215c`). Fetch via Outline MCP before relying on this doc's content — this file is a pointer, not a mirror.
