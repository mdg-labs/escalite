# 03 — Mobile App Spec (Minimal)

**Outline URL:** https://outline.mdg-labs.dev/doc/03-mobile-app-spec-minimal-v3pleAhwlp  
**Outline document id:** `82b6b1dc-75df-4354-8e2f-077dcd51cdc1`

A minimal Expo (React Native) app is required for iOS Critical Alerts and reliable on-call push; scope is locked to receive push, acknowledge, and escalate/re-assign — all other UX stays in the responsive web/PWA via deep links. Push flows from the Go engine through Expo Push API (or direct APNs/FCM later); notification payloads define actionable ack/escalate buttons. Auth reuses the web login flow with device-bound refresh tokens in Keychain/Keystore. Critical Alerts entitlement requires Apple Developer review (discretionary); Time-Sensitive notifications are the fallback while entitlement is pending.

> **Source of truth:** Outline MCP (collection: Escalite, doc id: `82b6b1dc-75df-4354-8e2f-077dcd51cdc1`). Fetch via Outline MCP before relying on this doc's content — this file is a pointer, not a mirror.
