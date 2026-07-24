# 04 — Licensing & Editions (Community Edition + Cloud)

**Outline URL:** https://outline.mdg-labs.dev/doc/04-licensing-editions-community-edition-cloud-bsCCX8dNhd  
**Outline document id:** `44307d04-7eef-44f7-94e1-0ed0031bf8db`

Community Edition is AGPL-3.0, self-hosted, and ships full feature parity with Cloud — no gated features or EE modules in the public repo. Cloud is a proprietary control plane in a separate `escalite-cloud` repo (billing, provisioning, tenant admin, metering, auth gateway) that orchestrates isolated CE instances per customer rather than embedding multi-tenancy in CE. DCO sign-off is preferred over a CLA. SSO, status pages, analytics, and multi-org are CE features on the product roadmap, not license gates. Cloud orchestration need not start on Kubernetes; Docker/Coolify-style provisioning is acceptable initially.

> **Source of truth:** Outline MCP (collection: Escalite, doc id: `44307d04-7eef-44f7-94e1-0ed0031bf8db`). Fetch via Outline MCP before relying on this doc's content — this file is a pointer, not a mirror.
