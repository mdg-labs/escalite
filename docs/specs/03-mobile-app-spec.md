# 03 — Mobile App Spec (Minimal)

**Outline URL:** https://outline.mdg-labs.dev/doc/03-mobile-app-spec-minimal-v3pleAhwlp  
**Outline document id:** `82b6b1dc-75df-4354-8e2f-077dcd51cdc1`

A minimal Expo + Tamagui app is required for iOS Critical Alerts and reliable on-call push; scope is locked to receive push, acknowledge, and escalate/re-assign — all other UX stays in the responsive web/PWA via deep links. In-app UI uses Tamagui; semantic tokens from `packages/tokens` match web severity styling. Push flows from the Go engine through Expo Push API; auth reuses the web login flow with device-bound refresh tokens. **Self-hosted instances:** users configure their Escalite server URL in-app on first launch (see Outline `server-configuration`). **Versioning:** mobile semver lives in `apps/mobile/package.json` (independent of root `VERSION`); Git tags use `mobile-v*` (see `docs/deploy/mobile-releases.md`). Android release setup: Outline doc 09. See Outline for per-screen Tamagui component guide.

> **Source of truth:** Outline MCP (collection: Escalite, doc id: `82b6b1dc-75df-4354-8e2f-077dcd51cdc1`). Fetch via Outline MCP before relying on this doc's content — this file is a pointer, not a mirror.
