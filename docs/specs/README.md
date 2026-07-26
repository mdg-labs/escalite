# Escalite specification pointers

The authoritative specifications for Escalite live in the **Escalite** collection in Outline (collection id: `699e1fff-a593-4767-9f29-0a9a96bf48cc`). Files in this directory are **pointers only** — they are not mirrors of the full spec content.

## Why pointers, not copies

Outline is the single source of truth. Full copies in git would drift from the live specs and create conflicting guidance for humans and agents. Each file here gives you enough context to know *which* doc to fetch and *what it covers*; always pull the latest body via the Outline MCP before implementing or reviewing behavior.

## Pointer convention

Every `NN-*.md` file contains:

- Document title
- Outline URL and document id
- A short summary (3–5 sentences)
- A source-of-truth note with the Outline collection and doc id

**For contributors and agents:** use `fetch` on the Outline MCP with the document id before relying on any spec detail. Treat these markdown files as an index, not the spec itself.

## Documents

| File | Title | Summary |
|------|-------|---------|
| [00-vision-and-scope.md](./00-vision-and-scope.md) | Vision & Scope | Product vision, gap analysis, pillars, non-goals, target users, MVP success criteria |
| [01-architecture-and-monorepo.md](./01-architecture-and-monorepo.md) | Architecture & Monorepo | Stack, monorepo layout, build-vs-fork, deployment targets, realtime |
| [02-core-domain-and-features.md](./02-core-domain-and-features.md) | Core Domain & Features | Domain model, escalation, scheduling, integrations, notifications |
| [03-mobile-app-spec.md](./03-mobile-app-spec.md) | Mobile App (Minimal) | Native app scope, Expo + Tamagui, push/Critical Alerts, auth |
| [04-licensing-and-editions.md](./04-licensing-and-editions.md) | Licensing & Editions | AGPL CE, Cloud split, two-repo model, tenant isolation |
| [05-ui-design-system.md](./05-ui-design-system.md) | UI / Design System | COSS UI, packages/tokens, Tremor, component/particle guide |
| [06-roadmap-mvp-phasing.md](./06-roadmap-mvp-phasing.md) | Roadmap / MVP Phasing | Phases 0–6; MVP = Phases 0–5 CE + Docker Compose only |
| [07-security-and-auth.md](./07-security-and-auth.md) | Security, AuthN/AuthZ & Hardening | Sessions, RBAC, webhook security, secrets, audit, CI hygiene |
| [08-implementation-decisions-and-conventions.md](./08-implementation-decisions-and-conventions.md) | Implementation Decisions (agent-binding) | Pinned tools, conventions, Definition of Done, Phase 0 bootstrap order |

Operator-only docs in Outline (not mirrored here): **09 — Mobile Android Release Setup (operator)** — Expo/EAS, `EXPO_TOKEN`, Android keystore, Obtainium. See [`../deploy/mobile-releases.md`](../deploy/mobile-releases.md).

## Machine-readable roadmap

Implementation sequencing derived from doc 06 and doc 08 lives in [`../roadmap/roadmap.yaml`](../roadmap/roadmap.yaml). Human-readable rendering: [`../roadmap/ROADMAP.md`](../roadmap/ROADMAP.md).
