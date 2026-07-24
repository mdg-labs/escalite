# 06 — Roadmap / MVP Phasing

**Outline URL:** https://outline.mdg-labs.dev/doc/06-roadmap-mvp-phasing-CE27IW9vNv  
**Outline document id:** `4b5b2710-5ceb-489a-9bc5-65e1033d9f0f`

MVP scope is Phases 0–5 only: Community Edition, self-hosted via Docker Compose — no Phase 6 Cloud work. Phase 0 covers repo scaffolding, compose, schema v1, auth, GraphQL/codegen shell, river queue, and CI E2E smoke. Phase 1 delivers GoAlert-parity alerting (escalation, schedules, heartbeats, channels, inbound plugins, basic dedup, web screens). Phase 2 adds mobile, Critical Alerts, Twilio SMS/voice, advanced dedup, and maintenance windows. Phase 3 adds incidents, Slack channel-per-incident, and postmortem export. Phase 4 covers importers and hardened compose/Coolify validation (no Helm). Phase 5 adds SSO/SAML/SCIM, status pages, MTTA/MTTR analytics, and multi-org.

> **Source of truth:** Outline MCP (collection: Escalite, doc id: `4b5b2710-5ceb-489a-9bc5-65e1033d9f0f`). Fetch via Outline MCP before relying on this doc's content — this file is a pointer, not a mirror.
