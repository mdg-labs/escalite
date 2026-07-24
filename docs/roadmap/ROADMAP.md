# Escalite MVP Roadmap (Phases 0–5)

> **Source of truth:** [`roadmap.yaml`](./roadmap.yaml). This file is generated — do not edit by hand. Regenerate with `python docs/roadmap/generate_roadmap.py`.

## Summary

- **Phases:** 6 (0–5)
- **Epics:** 29
- **Tasks:** 109

## External dependencies (human action required)

- **Blocks:** `p2-ios-critical-alerts-entitlement`
  - **Action:** Apple Developer account + submit Critical Alerts entitlement request; approval is discretionary and timing is external

- **Blocks:** `p2-ios-critical-alerts-impl`
  - **Action:** Cannot enable Critical Alerts in production until Apple approves entitlement

- **Blocks:** `p2-twilio-integration`
  - **Action:** Twilio account + credentials for SMS/voice testing and production send

- **Blocks:** `p4-coolify-validation`
  - **Action:** Coolify instance or VPS for manual validation deploy; DNS/TLS domain decisions by operator

- **Blocks:** `p4-pagerduty-importer`
  - **Action:** PagerDuty API token or export file from operator account for test/import runs

- **Blocks:** `p5-status-page-app`
  - **Action:** Custom domain DNS and TLS certificate provisioning for public status page (operator responsibility)

## Phases

### phase-0: Foundations

#### Epic: Repo scaffolding & CI skeleton (`phase-0-repo-scaffold`)

Initialize the public AGPL monorepo structure, pinned tooling, and CI gates per doc 08 bootstrap
step 1.

- **`p0-scaffold-monorepo`** — Initialize pnpm workspace + Turborepo + go.work + Taskfile
  - Size: S | Depends on: _none_
  - Acceptance criteria:
    - pnpm-workspace.yaml defines apps/*, services/*, packages/*
    - turbo.json configured with build, test, and lint pipelines
    - go.work includes services/api, services/engine, services/integrations
    - Taskfile.yml exposes: dev, build, test, lint, migrate
    - Root README documents clone → task dev prerequisites

- **`p0-packages-config-shared`** — Add packages/config shared ESLint, Prettier, and tsconfig
  - Size: S | Depends on: `p0-scaffold-monorepo`
  - Acceptance criteria:
    - packages/config exports eslint and prettier configs consumable by apps/web
    - TypeScript strict mode enabled in base tsconfig
    - turbo lint task runs ESLint across JS/TS workspaces

- **`p0-license-agpl-dco`** — Add AGPL-3.0 LICENSE and DCO enforcement in CI
  - Size: S | Depends on: `p0-scaffold-monorepo`
  - Acceptance criteria:
    - LICENSE file is AGPL-3.0
    - CI fails PRs missing DCO sign-off per doc 04
    - CONTRIBUTING.md references conventional commits

- **`p0-ci-skeleton`** — GitHub Actions CI: lint, test, build, vuln scan
  - Size: M | Depends on: `p0-scaffold-monorepo`, `p0-packages-config-shared`
  - Acceptance criteria:
    - CI runs on pull_request and push to main
    - govulncheck and pnpm audit fail on known-critical vulns per doc 07
    - Renovate or Dependabot config present

- **`p0-go-service-skeletons`** — Scaffold Go service modules (api, engine, integrations)
  - Size: M | Depends on: `p0-scaffold-monorepo`
  - Acceptance criteria:
    - Each service builds to a static binary via go build
    - Config package fails fast with actionable errors for missing required env
    - Structured logs include org_id/service_id keys when available

- **`p0-compose-dev-prod`** — Docker Compose dev + prod profiles (Postgres, api, engine, web)
  - Size: M | Depends on: `p0-go-service-skeletons`
  - Acceptance criteria:
    - docker compose --profile dev up starts postgres, api, engine, web
    - docker compose --profile prod up uses built images tagged locally
    - .env.example documents ESCALITE_ENCRYPTION_KEY generation one-liner
    - Optional commented Caddy reverse-proxy service per doc 07

- **`p0-dockerfiles-nonroot`** — Distroless/slim Dockerfiles with non-root user and pinned bases
  - Size: S | Depends on: `p0-compose-dev-prod`
  - Acceptance criteria:
    - Containers run as non-root UID
    - Release compose documents image digest pinning approach

#### Epic: Postgres schema v1 (`phase-0-schema`)

Atlas declarative schema, generated migrations, and sqlc foundation for core domain tables per doc
08 bootstrap step 3.

- **`p0-atlas-migrations-setup`** — Wire Atlas declarative schema and migration tooling on API startup
  - Size: S | Depends on: `p0-compose-dev-prod`
  - Acceptance criteria:
    - services/api/schema/ is canonical; zero hand-written DDL in migrations/
    - atlas migrate apply applies all migrations idempotently
    - Concurrent API starts do not corrupt migrations (advisory lock)
    - task migrate runs migrations locally without starting full stack
    - CI atlas migrate lint / schema drift gate passes

- **`p0-schema-core-entities`** — Schema: organizations, teams, users, team_memberships
  - Size: M | Depends on: `p0-atlas-migrations-setup`
  - Acceptance criteria:
    - All PKs are UUID (v7 generated in app layer)
    - users.email unique per organization
    - team_memberships enforces user belongs to org of team
    - Migration generated via atlas migrate diff from schema change (zero hand-written DDL)

- **`p0-schema-service-integration`** — Schema: services, integration_keys
  - Size: M | Depends on: `p0-schema-core-entities`
  - Acceptance criteria:
    - integration_keys.token stores only hash; prefix column for UI last-4 style
    - integration_keys.plugin_name references compile-time registry names
    - Migration generated via atlas migrate diff from schema change (zero hand-written DDL)

- **`p0-schema-scheduling`** — Schema: schedules, rotations, overrides
  - Size: M | Depends on: `p0-schema-core-entities`
  - Acceptance criteria:
    - schedules.timezone stores valid IANA zone string
    - overrides support soft-delete with deleted_at where product requires audit retention
    - Migration generated via atlas migrate diff from schema change (zero hand-written DDL)

- **`p0-schema-escalation-alerts`** — Schema: escalation_policies, steps, alerts, notification_attempts
  - Size: M | Depends on: `p0-schema-service-integration`
  - Acceptance criteria:
    - alerts.status enum: triggered, acknowledged, closed
    - alerts.dedup_key indexed per service_id
    - escalation_steps.order unique per policy
    - Migration generated via atlas migrate diff from schema change (zero hand-written DDL)

- **`p0-schema-sessions-audit`** — Schema: sessions, audit_events, refresh_tokens
  - Size: M | Depends on: `p0-schema-core-entities`
  - Acceptance criteria:
    - sessions store expires_at and user_agent fingerprint
    - audit_events capture actor_id, action, target_type, target_id, metadata JSON
    - refresh_tokens individually revocable
    - Migration generated via atlas migrate diff from schema change (zero hand-written DDL)

- **`p0-sqlc-foundation`** — sqlc queries for auth, org, and health checks
  - Size: M | Depends on: `p0-schema-sessions-audit`, `p0-schema-escalation-alerts`
  - Acceptance criteria:
    - sqlc generate produces compiles Go code
    - Integration test uses testcontainers-go Postgres

#### Epic: Auth vertical slice (`phase-0-auth-slice`)

First-admin setup, sessions, RBAC, optional OIDC, encryption at rest per doc 08 bootstrap step 4.

- **`p0-config-encryption-key`** — App config validation and ESCALITE_ENCRYPTION_KEY enforcement
  - Size: S | Depends on: `p0-go-service-skeletons`
  - Acceptance criteria:
    - API and engine refuse start when ESCALITE_ENCRYPTION_KEY missing or placeholder
    - .env.example shows openssl rand -hex 32 one-liner

- **`p0-auth-password-argon2id`** — Password hashing with argon2id
  - Size: S | Depends on: `p0-schema-core-entities`
  - Acceptance criteria:
    - Unit tests verify hash round-trip and rejection of wrong password
    - Password hashes stored only in users.password_hash

- **`p0-auth-first-admin-setup`** — First-admin setup flow (single-org bootstrap)
  - Size: M | Depends on: `p0-auth-password-argon2id`, `p0-sqlc-foundation`
  - Acceptance criteria:
    - Setup endpoint returns 403 when any user exists
    - Successful setup creates org, admin user (role=admin), and session
    - Setup covered by integration test

- **`p0-auth-login-sessions`** — Email+password login and Postgres session cookies
  - Size: M | Depends on: `p0-auth-first-admin-setup`
  - Acceptance criteria:
    - Invalid credentials return GraphQL error code UNAUTHENTICATED without user enumeration leak
    - Session cookie not accessible to JS (HttpOnly)
    - Revoked session returns UNAUTHENTICATED on next request

- **`p0-auth-rbac-guards`** — RBAC guards: admin vs member, team scoping
  - Size: M | Depends on: `p0-auth-login-sessions`
  - Acceptance criteria:
    - Non-member accessing team resource returns FORBIDDEN
    - Admin can access all teams in org
    - GraphQL errors use stable code extensions per doc 08

- **`p0-auth-oidc-optional`** — Optional generic OIDC login (ESCALITE_OIDC_*)
  - Size: M | Depends on: `p0-auth-login-sessions`
  - Acceptance criteria:
    - OIDC disabled when env vars unset
    - OIDC login creates/links user and session same as password flow
    - Uses coreos/go-oidc and golang.org/x/oauth2

- **`p0-auth-password-reset`** — Password reset via email with rate limiting
  - Size: M | Depends on: `p0-auth-login-sessions`
  - Acceptance criteria:
    - Reset email sent only if user exists (same response either way)
    - Expired reset tokens rejected with VALIDATION code
    - Rate limit returns RATE_LIMITED after threshold

- **`p0-audit-events-auth`** — Audit log capture for auth and privileged actions
  - Size: S | Depends on: `p0-auth-rbac-guards`
  - Acceptance criteria:
    - Failed login attempts logged with IP and user agent
    - audit_events table append-only (no UPDATE/DELETE in app code)

- **`p0-secrets-encryption-at-rest`** — AES-256-GCM encryption helper for provider credentials
  - Size: M | Depends on: `p0-config-encryption-key`, `p0-sqlc-foundation`
  - Acceptance criteria:
    - Round-trip encrypt/decrypt integration test
    - Decrypt fails with wrong key id with actionable error

#### Epic: GraphQL/codegen skeleton + web shell (`phase-0-graphql-skeleton`)

SDL, gqlgen, codegen to ts-types, packages/ui primitives, Vite web shell per doc 08 bootstrap step
5.

- **`p0-schema-sdl-foundation`** — packages/schema GraphQL SDL foundation
  - Size: S | Depends on: `p0-auth-rbac-guards`
  - Acceptance criteria:
    - SDL is single source of truth in packages/schema
    - Schema validates with gqlgen init

- **`p0-gqlgen-wiring`** — gqlgen server wiring in services/api
  - Size: M | Depends on: `p0-schema-sdl-foundation`
  - Acceptance criteria:
    - Depth and complexity limits configured
    - GraphQL errors include extensions.code per doc 08

- **`p0-graphql-codegen-ts`** — graphql-codegen → packages/ts-types
  - Size: S | Depends on: `p0-schema-sdl-foundation`
  - Acceptance criteria:
    - pnpm codegen produces packages/ts-types without manual edits
    - turbo build depends on codegen output

- **`p0-packages-ui-primitives`** — packages/ui primitives (shadcn base + tokens + dark mode)
  - Size: M | Depends on: `p0-packages-config-shared`
  - Acceptance criteria:
    - Dark mode toggles via class on html element
    - Severity colors documented as colorblind-safe in tokens

- **`p0-web-app-shell`** — apps/web Vite shell: login, setup, empty dashboard
  - Size: M | Depends on: `p0-gqlgen-wiring`, `p0-graphql-codegen-ts`, `p0-packages-ui-primitives`
  - Acceptance criteria:
    - Unauthenticated users redirect to /login
    - Successful login lands on /dashboard with user name displayed
    - CORS locked to configured app origin

#### Epic: river job queue wiring (`phase-0-queue`)

Prove Postgres-backed job queue path per doc 08 bootstrap step 6.

- **`p0-river-client-setup`** — river client and worker registration in engine service
  - Size: M | Depends on: `p0-atlas-migrations-setup`, `p0-go-service-skeletons`
  - Acceptance criteria:
    - river migrations applied on engine startup
    - Worker logs job start/finish with structured slog fields

- **`p0-river-noop-heartbeat-job`** — First noop heartbeat scan job to prove queue path
  - Size: S | Depends on: `p0-river-client-setup`
  - Acceptance criteria:
    - Job enqueued and processed within 60s in dev compose
    - Failed jobs retry per river defaults with logged error

#### Epic: First E2E smoke in CI (`phase-0-ci-smoke`)

Playwright smoke and compose-up gate per doc 08 bootstrap step 7.

- **`p0-playwright-setup`** — Playwright test harness against compose stack
  - Size: S | Depends on: `p0-web-app-shell`, `p0-compose-dev-prod`
  - Acceptance criteria:
    - pnpm e2e runs Playwright headless in CI
    - Tests wait for /healthz before interactions

- **`p0-e2e-smoke-setup-login`** — E2E smoke: first-admin setup → login → dashboard
  - Size: M | Depends on: `p0-playwright-setup`, `p0-auth-first-admin-setup`
  - Acceptance criteria:
    - Test runs against ephemeral Postgres in CI
    - Test artifacts uploaded on failure

- **`p0-ci-compose-gate`** — CI job: docker compose up end-to-end gate
  - Size: M | Depends on: `p0-e2e-smoke-setup-login`, `p0-ci-skeleton`
  - Acceptance criteria:
    - CI fails if docker compose up exits non-zero
    - CI runs E2E smoke as required check on main

---

### phase-1: Core alerting (GoAlert parity)

#### Epic: Escalation engine (`phase-1-escalation-engine`)

Trigger → notify → escalate → ack/close state machine with river timers per doc 02 and Phase 1
scope.

- **`p1-escalation-policy-crud`** — Escalation policy and step CRUD API
  - Size: M | Depends on: `p0-ci-compose-gate`
  - Acceptance criteria:
    - Admin can create policy with ordered steps and delay_minutes
    - Step order gaps rejected with VALIDATION
    - Policy changes emit audit_events

- **`p1-escalation-trigger-notify`** — Alert trigger schedules step-1 notifications
  - Size: L | Depends on: `p1-escalation-policy-crud`, `p0-river-noop-heartbeat-job`
  - Acceptance criteria:
    - Triggered alert creates notification_attempt rows per target/channel
    - Integration test with testcontainers verifies step-1 enqueue within 5s

- **`p1-escalation-step-timers`** — Per-step delay timers via river
  - Size: L | Depends on: `p1-escalation-trigger-notify`
  - Acceptance criteria:
    - Timer fires and advances escalation_state.current_step
    - Acknowledged alert cancels pending escalation jobs

- **`p1-escalation-repeat-cap`** — Repeat-last-step with max repeat count
  - Size: M | Depends on: `p1-escalation-step-timers`
  - Acceptance criteria:
    - After max repeats, escalation stops and alert flagged escalated_exhausted
    - Unit test covers repeat boundary

- **`p1-escalation-ack-close`** — Acknowledge and close mutations
  - Size: M | Depends on: `p1-escalation-step-timers`
  - Acceptance criteria:
    - ack sets status=acknowledged and records acked_by/at
    - close sets status=closed; closed alerts reject re-ack with VALIDATION

- **`p1-escalation-manual-snooze`** — Manual snooze and re-escalate from UI/API
  - Size: M | Depends on: `p1-escalation-ack-close`
  - Acceptance criteria:
    - snooze shifts next_escalation_at by requested duration
    - reEscalate resets to step 1 and enqueues notifications

#### Epic: Schedules, rotations, and overrides (`phase-1-scheduling`)

RRULE-based on-call scheduling with overrides and DST tests.

- **`p1-schedule-crud`** — Schedule and rotation CRUD with RRULE storage
  - Size: M | Depends on: `p0-ci-compose-gate`
  - Acceptance criteria:
    - Invalid RRULE rejected with VALIDATION and human-readable message
    - Schedules require valid IANA timezone

- **`p1-schedule-on-call-now`** — onCallNow query for schedule layers
  - Size: L | Depends on: `p1-schedule-crud`
  - Acceptance criteria:
    - onCallNow returns primary and secondary layers when configured
    - Integration test covers DST spring-forward and fall-back boundaries

- **`p1-schedule-overrides`** — Override CRUD with audit trail
  - Size: M | Depends on: `p1-schedule-crud`
  - Acceptance criteria:
    - Active override replaces rotation participant for overlapping window
    - Override create/delete writes audit_events

- **`p1-schedule-ical-export`** — iCal export for schedule rotations
  - Size: M | Depends on: `p1-schedule-on-call-now`
  - Acceptance criteria:
    - Exported ICS validates against RFC 5545 test fixture
    - DTSTART/TZID match schedule timezone

- **`p1-escalation-target-rotation`** — Escalation targets resolve rotation schedules to users
  - Size: M | Depends on: `p1-schedule-on-call-now`, `p1-escalation-trigger-notify`
  - Acceptance criteria:
    - Step targeting rotation notifies current on-call user(s)
    - Empty rotation returns logged skip, not panic

#### Epic: Heartbeat monitors (dead man's switch) (`phase-1-heartbeat-monitors`)

GoAlert-equivalent heartbeat monitors per doc 02.

- **`p1-heartbeat-crud`** — HeartbeatMonitor CRUD per service
  - Size: M | Depends on: `p0-ci-compose-gate`
  - Acceptance criteria:
    - Monitors belong to exactly one service
    - interval and grace must be positive durations

- **`p1-heartbeat-ping-endpoint`** — Heartbeat ping endpoint with token auth
  - Size: M | Depends on: `p1-heartbeat-crud`, `p0-secrets-encryption-at-rest`
  - Acceptance criteria:
    - Invalid token returns 404 without leaking existence
    - Valid ping updates last_ping_at and sets status healthy
    - Token logged only as prefix per doc 07

- **`p1-heartbeat-deadline-scan`** — River periodic scan for overdue heartbeats
  - Size: M | Depends on: `p1-heartbeat-ping-endpoint`, `p1-escalation-trigger-notify`
  - Acceptance criteria:
    - Overdue monitor creates alert with source=heartbeat and dedup_key=monitor id
    - Scan interval configurable via ESCALITE_HEARTBEAT_SCAN_INTERVAL

#### Epic: Notification channels (push basic, email, webhook, Slack DM) (`phase-1-notification-channels`)

Day-1 outbound channels without Twilio or Critical Alerts (Phase 2).

- **`p1-channel-registry`** — NotificationChannel plugin registry in engine
  - Size: M | Depends on: `p1-escalation-trigger-notify`
  - Acceptance criteria:
    - Unknown channel name returns clear error at config save
    - ConfigSchema drives web UI form generation

- **`p1-channel-email`** — Email notification channel via SMTP
  - Size: M | Depends on: `p1-channel-registry`
  - Acceptance criteria:
    - SMTP misconfig surfaces actionable startup warning
    - Send failure records notification_attempt status failed with error

- **`p1-channel-webhook-outbound`** — Generic outbound webhook channel
  - Size: M | Depends on: `p1-channel-registry`
  - Acceptance criteria:
    - Payload includes alert id, service, status, title, body
    - Timeout default 10s; failure retried per engine policy

- **`p1-channel-slack-dm`** — Slack per-user DM notification channel
  - Size: M | Depends on: `p1-channel-registry`, `p0-secrets-encryption-at-rest`
  - Note: Operator must create Slack app and provide bot token — per-install config, not blocking development with test token.
  - Acceptance criteria:
    - Slack token never returned in full after save (last-4 hint)
    - Missing slack_user_id skips with logged reason

- **`p1-channel-push-basic`** — Basic mobile push via Expo Push API (no Critical Alerts)
  - Size: L | Depends on: `p1-channel-registry`
  - Acceptance criteria:
    - Push payload matches contract in doc 03 minus critical:true default
    - Invalid Expo token marks notification_attempt failed

- **`p1-notification-rules`** — Per-user notification rules by priority
  - Size: M | Depends on: `p1-channel-email`, `p1-channel-push-basic`
  - Acceptance criteria:
    - High-priority alert uses user's high rule ordering
    - Rules applied when resolving contact methods at send time

#### Epic: Inbound plugin system and day-1 plugins (`phase-1-inbound-plugins`)

Compile-time InboundPlugin registry and secured webhook router.

- **`p1-inbound-registry`** — InboundPlugin registry in services/integrations
  - Size: M | Depends on: `p0-ci-compose-gate`
  - Acceptance criteria:
    - Plugins registered at init via single registry.go import list
    - ParseAlert resolved event type documented in interface

- **`p1-inbound-webhook-router`** — Secured webhook ingestion router
  - Size: L | Depends on: `p1-inbound-registry`, `p0-auth-rbac-guards`
  - Acceptance criteria:
    - Returns 429 with {error, code: RATE_LIMITED} when >120 req/min per IntegrationKey
    - Returns 413 when body exceeds 256KB
    - Invalid token returns 404; token never logged in full

- **`p1-inbound-alertmanager`** — prometheus-alertmanager plugin (reference implementation)
  - Size: L | Depends on: `p1-inbound-webhook-router`
  - Acceptance criteria:
    - Firing creates triggered alert with dedup_key=fingerprint
    - Resolved auto-closes matching open alert by dedup_key
    - Fixture-based unit tests from upstream Alertmanager samples

- **`p1-inbound-generic-webhook`** — generic-webhook plugin with JSON field mapping
  - Size: M | Depends on: `p1-inbound-webhook-router`
  - Acceptance criteria:
    - Mapping config validated against JSON Schema
    - Missing required mapped fields returns 400 with VALIDATION body

- **`p1-inbound-generic-rest`** — generic-rest-api authenticated alert create
  - Size: M | Depends on: `p1-inbound-webhook-router`
  - Acceptance criteria:
    - Missing Authorization returns 401 {error, code: UNAUTHENTICATED}
    - Valid payload creates alert and returns 201 with alert id

- **`p1-inbound-email-to-alert`** — email-to-alert inbound parsing
  - Size: L | Depends on: `p1-inbound-registry`
  - Acceptance criteria:
    - Inbound email endpoint rejects unsigned/unauthenticated mail per SMTP relay config
    - Parsed email creates alert with source=email

- **`p1-inbound-grafana`** — Grafana inbound plugin
  - Size: M | Depends on: `p1-inbound-alertmanager`
  - Acceptance criteria:
    - Grafana firing and resolved states map to EventType correctly
    - Integration test uses captured Grafana fixture JSON

- **`p1-inbound-datadog`** — Datadog inbound plugin
  - Size: M | Depends on: `p1-inbound-alertmanager`
  - Acceptance criteria:
    - Datadog RECOVERY event resolves matching alert
    - Invalid signature when configured returns 401

- **`p1-inbound-uptime-kuma`** — uptime-kuma dedicated plugin
  - Size: M | Depends on: `p1-inbound-generic-webhook`
  - Acceptance criteria:
    - monitor.id used as dedup_key scoped to service
    - Up event resolves prior down alert with same dedup_key

#### Epic: Beszel integration preset (`phase-1-beszel-preset`)

UI preset atop generic-webhook for Shoutrrr generic:// JSON payloads.

- **`p1-beszel-preset-ui`** — Beszel preset in integration picker + setup docs
  - Size: S | Depends on: `p1-inbound-generic-webhook`
  - Acceptance criteria:
    - Selecting Beszel creates IntegrationKey with correct plugin and mapping
    - Docs page shows exact generic:// URL format for Beszel notification settings
    - No new backend plugin package added

#### Epic: Basic dedup-key matching (`phase-1-dedup-basic`)

Collapse repeat alerts on same service+dedup_key; enable auto-resolve from integrations.

- **`p1-dedup-key-collapse`** — Dedup-key collapse into open alert
  - Size: M | Depends on: `p1-inbound-alertmanager`
  - Acceptance criteria:
    - Second firing increments alert.event_count
    - Collapsed alert does not re-notify already-acked targets in Phase 1 basic mode

- **`p1-dedup-auto-resolve`** — Auto-resolve path for resolved integration events
  - Size: M | Depends on: `p1-dedup-key-collapse`
  - Acceptance criteria:
    - Resolve for unknown dedup_key returns 200 no-op logged at debug
    - Closed alert records resolved_at and source integration name

#### Epic: Realtime GraphQL subscriptions (`phase-1-realtime`)

LISTEN/NOTIFY to graphql-ws for live alert and on-call updates.

- **`p1-listen-notify-bridge`** — Postgres LISTEN/NOTIFY bridge in API
  - Size: M | Depends on: `p1-escalation-ack-close`
  - Acceptance criteria:
    - Alert status change triggers NOTIFY within same transaction commit
    - Reconnect logic on listener disconnect

- **`p1-graphql-subscriptions`** — graphql-ws subscriptions for alerts and onCallNow
  - Size: M | Depends on: `p1-listen-notify-bridge`, `p0-web-app-shell`
  - Acceptance criteria:
    - urql subscription receives event within 2s of DB change in integration test
    - Unauthenticated subscription rejected

#### Epic: Web app core screens (`phase-1-web-screens`)

Alert list/detail, service config, schedule calendar, escalation editor per Phase 1.

- **`p1-ui-integration-keys`** — Integration key management UI
  - Size: M | Depends on: `p1-inbound-webhook-router`
  - Acceptance criteria:
    - New key shown once on create; thereafter only prefix visible
    - Revoked keys return 404 on webhook within 60s

- **`p1-ui-service-config`** — Service configuration screens
  - Size: M | Depends on: `p1-escalation-policy-crud`, `p1-ui-integration-keys`
  - Acceptance criteria:
    - Service list searchable by name
    - Deleting service soft-deletes with audit event

- **`p1-ui-alert-list-detail`** — Alert list and detail views with ack/close
  - Size: L | Depends on: `p1-escalation-ack-close`, `p1-graphql-subscriptions`
  - Acceptance criteria:
    - Ack and close buttons call mutations and update via subscription
    - Alert list shows event_count for deduped alerts

- **`p1-ui-escalation-editor`** — Escalation policy editor component
  - Size: L | Depends on: `p1-escalation-policy-crud`
  - Acceptance criteria:
    - Drag reorder updates step order persisted to API
    - Invalid step without targets blocked in UI and API

- **`p1-ui-schedule-calendar`** — Schedule calendar and override UI
  - Size: L | Depends on: `p1-schedule-overrides`, `p1-schedule-on-call-now`
  - Acceptance criteria:
    - Calendar displays viewer-local times with timezone label
    - Override creation requires admin or member with team access

- **`p1-e2e-alert-flow`** — E2E: create service → fire test alert → ack
  - Size: M | Depends on: `p1-ui-alert-list-detail`, `p1-inbound-generic-rest`
  - Acceptance criteria:
    - Test creates service via UI, triggers alert via generic-rest API, acks in UI
    - CI runs on main alongside compose gate

---

### phase-2: Mobile + reliable delivery

#### Epic: Mobile app v1 (Expo) (`phase-2-mobile`)

Minimal native app: push, ack, escalate per doc 03.

- **`p2-mobile-expo-scaffold`** — Expo app scaffold in apps/mobile
  - Size: M | Depends on: `p0-graphql-codegen-ts`
  - Acceptance criteria:
    - apps/mobile builds with eas.json stub for future builds
    - Deep link scheme escalite:// configured

- **`p2-mobile-auth-deep-link`** — Mobile auth via web login + deep link token exchange
  - Size: M | Depends on: `p2-mobile-expo-scaffold`, `p0-auth-login-sessions`
  - Acceptance criteria:
    - Refresh token stored in Keychain/Keystore only
    - Revoked refresh token returns UNAUTHENTICATED on refresh

- **`p2-mobile-device-token-register`** — Device push token registration mutation
  - Size: M | Depends on: `p2-mobile-auth-deep-link`
  - Acceptance criteria:
    - Duplicate token upserts same device row
    - User settings lists devices with revoke action

- **`p2-mobile-push-receive`** — Receive and display alert push notifications
  - Size: M | Depends on: `p2-mobile-device-token-register`, `p1-channel-push-basic`
  - Acceptance criteria:
    - Notification displays title/body from payload
    - Tap navigates to alert detail route

- **`p2-mobile-ack-escalate`** — Ack and escalate from notification actions and in-app
  - Size: L | Depends on: `p2-mobile-push-receive`, `p1-escalation-manual-snooze`
  - Acceptance criteria:
    - Ack from lock screen succeeds without opening app (platform permitting)
    - Escalate prompts for target selection minimal list

#### Epic: iOS Critical Alerts entitlement (`phase-2-ios-critical-alerts`)

Apple entitlement application and implementation; start early parallel to Phase 1 per doc 06.

- **`p2-ios-critical-alerts-entitlement`** — Submit Apple Critical Alerts entitlement request ⚠️ **EXTERNAL DEPENDENCY**
  - Size: S | Depends on: `p0-scaffold-monorepo`
  - Note: Doc 06: start in parallel with Phase 1 (not blocked on Phase 2). Bundle ID documented in repo once chosen; mobile scaffold not required to submit entitlement request.
  - Acceptance criteria:
    - Entitlement request submitted in Apple Developer portal
    - App ID configured with Critical Alerts capability pending approval
    - Fallback documented: Time-Sensitive notifications ship regardless of approval status

- **`p2-ios-critical-alerts-impl`** — Implement Critical Alerts push when entitlement approved ⚠️ **EXTERNAL DEPENDENCY**
  - Size: M | Depends on: `p2-ios-critical-alerts-entitlement`, `p2-mobile-push-receive`
  - Acceptance criteria:
    - When critical:true in payload and entitlement present, iOS delivers as Critical Alert
    - Without entitlement, falls back to Time-Sensitive per doc 03

#### Epic: SMS and voice via Twilio (`phase-2-twilio-channels`)

Pluggable provider with Twilio adapter; operator-supplied credentials.

- **`p2-twilio-provider-interface`** — Pluggable SMS/voice provider interface
  - Size: M | Depends on: `p1-notification-rules`, `p0-secrets-encryption-at-rest`
  - Acceptance criteria:
    - Provider selection via ESCALITE_SMS_PROVIDER=twilio
    - Missing Twilio creds disable SMS/voice channels with startup warning

- **`p2-twilio-integration`** — Twilio SMS and voice channel implementation ⚠️ **EXTERNAL DEPENDENCY**
  - Size: L | Depends on: `p2-twilio-provider-interface`
  - Acceptance criteria:
    - SMS send succeeds against Twilio test credentials in integration test
    - Voice call TwiML speaks alert title and service name

#### Epic: Advanced deduplication and maintenance windows (`phase-2-dedup-advanced`)

Time-window dedup refinements and per-service mute windows.

- **`p2-dedup-time-window`** — Time-window dedup collapse with counter bumps
  - Size: M | Depends on: `p1-dedup-key-collapse`
  - Acceptance criteria:
    - Alerts outside window create new alert row
    - Window default 5 minutes configurable per service

- **`p2-dedup-no-renotify-acked`** — Suppress re-notify of already-acked targets
  - Size: M | Depends on: `p2-dedup-time-window`
  - Acceptance criteria:
    - Integration test: second firing does not create notification_attempt for acked user
    - Unacked targets still receive repeat per notification rules

- **`p2-maintenance-windows`** — Maintenance windows / per-service mute
  - Size: M | Depends on: `p1-ui-service-config`
  - Acceptance criteria:
    - Alert during active maintenance does not enqueue notifications
    - Admin UI shows active maintenance banner on service

---

### phase-3: Incident response layer

#### Epic: Incident aggregate and workflow (`phase-3-incident-aggregate`)

Promote alerts to incidents with timeline and roles per doc 02.

- **`p3-incident-schema`** — Incident, timeline_events, roles schema and API
  - Size: M | Depends on: `p2-dedup-no-renotify-acked`
  - Acceptance criteria:
    - incidents.status enum: investigating, identified, monitoring, resolved
    - timeline_events append-only with actor and body

- **`p3-incident-promote-manual`** — Manual promote alerts to incident
  - Size: M | Depends on: `p3-incident-schema`
  - Acceptance criteria:
    - Promote moves alerts under incident_id foreign key
    - Promote writes timeline_event type=declared

- **`p3-incident-promote-auto`** — Auto-promote rule (N alerts same service in window)
  - Size: M | Depends on: `p3-incident-promote-manual`
  - Acceptance criteria:
    - Rule default disabled; enabling creates incidents without duplicate if one open
    - Auto-promote suppresses per-alert escalation per severity config

- **`p3-incident-timeline-ui`** — Incident timeline UI with notes and state changes
  - Size: L | Depends on: `p3-incident-promote-manual`, `p1-graphql-subscriptions`
  - Acceptance criteria:
    - Adding note creates timeline_event visible to subscribers in <2s
    - Role assignment shows IC and Comms Lead labels

- **`p3-incident-suppress-escalation`** — Suppress per-alert escalation when incident active
  - Size: M | Depends on: `p3-incident-promote-auto`, `p1-escalation-step-timers`
  - Acceptance criteria:
    - New alert auto-attached to open incident skips step notifications when configured
    - Closing incident resumes normal escalation only for still-triggered alerts if configured

#### Epic: Slack channel-per-incident integration (`phase-3-slack-incidents`)

Rich Slack integration beyond Phase 1 DMs.

- **`p3-slack-app-install`** — Slack app OAuth install flow for workspace
  - Size: M | Depends on: `p3-incident-schema`, `p0-secrets-encryption-at-rest`
  - Acceptance criteria:
    - OAuth callback stores token with key-id encryption
    - Reinstall updates token without duplicate workspace rows

- **`p3-slack-channel-per-incident`** — Create Slack channel per declared incident
  - Size: L | Depends on: `p3-slack-app-install`, `p3-incident-promote-manual`
  - Acceptance criteria:
    - Channel naming template configurable with default incident-{short_id}
    - Failure to create channel logs error but does not roll back incident

- **`p3-slack-interactive-actions`** — Slack interactive ack/escalate buttons
  - Size: M | Depends on: `p3-slack-channel-per-incident`, `p1-escalation-ack-close`
  - Acceptance criteria:
    - Ack button acknowledges alert linked to incident
    - Invalid Slack signature returns 401

- **`p3-slack-timeline-mirror`** — Mirror incident timeline to Slack thread
  - Size: M | Depends on: `p3-slack-channel-per-incident`, `p3-incident-timeline-ui`
  - Acceptance criteria:
    - Note added in web appears in Slack thread within 5s
    - Resolve status posts summary message to channel

#### Epic: Postmortem Markdown export (`phase-3-postmortem`)

Generate postmortem doc from incident timeline.

- **`p3-postmortem-export`** — Markdown postmortem export from timeline
  - Size: M | Depends on: `p3-incident-timeline-ui`
  - Acceptance criteria:
    - Exported MD includes ISO8601 timestamps in UTC
    - Download returns text/markdown attachment

---

### phase-4: Migration & adoption tooling

#### Epic: Migration importers (`phase-4-migration-importers`)

GoAlert and PagerDuty export importers per doc 06 Phase 4.

- **`p4-goalert-importer-cli`** — GoAlert Postgres importer CLI
  - Size: L | Depends on: `p3-postmortem-export`
  - Acceptance criteria:
    - Dry-run mode prints mapping summary without writes
    - Import creates id mapping file for audit
    - Documented in docs/migration/goalert.md

- **`p4-pagerduty-importer`** — PagerDuty export API importer ⚠️ **EXTERNAL DEPENDENCY**
  - Size: L | Depends on: `p4-goalert-importer-cli`
  - Acceptance criteria:
    - API token stored only via env at import time, not persisted
    - Import report lists skipped unsupported objects

#### Epic: Hardened Docker Compose + Coolify validation (`phase-4-deployment-hardening`)

Production compose path and PaaS validation; explicitly no Helm.

- **`p4-compose-hardening`** — Hardened docker-compose.yml and production docs
  - Size: M | Depends on: `p4-goalert-importer-cli`
  - Acceptance criteria:
    - README deployment section starts with docker compose up
    - Services define healthcheck and restart policy
    - SBOM generated on release per doc 07

- **`p4-coolify-validation`** — Validate deployment on Coolify ⚠️ **EXTERNAL DEPENDENCY**
  - Size: M | Depends on: `p4-compose-hardening`
  - Acceptance criteria:
    - docs/deploy/coolify.md lists required env vars
    - Manual test checklist completed and recorded in doc

---

### phase-5: Remaining CE features

#### Epic: SSO / SAML / SCIM (`phase-5-sso-scim`)

Full enterprise auth beyond Phase 0 generic OIDC.

- **`p5-saml-sso`** — SAML SSO login and config UI
  - Size: L | Depends on: `p4-compose-hardening`
  - Acceptance criteria:
    - SAML response validated against IdP cert
    - Disabled SAML falls back to password/OIDC only

- **`p5-scim-provisioning`** — SCIM user and group provisioning
  - Size: L | Depends on: `p5-saml-sso`
  - Acceptance criteria:
    - SCIM bearer token rotatable from admin UI
    - Deprovisioned user cannot authenticate within 60s

#### Epic: Status pages app (`phase-5-status-pages`)

Public status page renderer in apps/status-page.

- **`p5-status-page-schema`** — Status page components and incident visibility model
  - Size: M | Depends on: `p4-compose-hardening`
  - Acceptance criteria:
    - Public page does not expose internal alert payloads
    - Component status enum: operational, degraded, partial_outage, major_outage

- **`p5-status-page-app`** — apps/status-page public renderer ⚠️ **EXTERNAL DEPENDENCY**
  - Size: L | Depends on: `p5-status-page-schema`, `p3-incident-timeline-ui`
  - Acceptance criteria:
    - Status page renders component grid and active incidents
    - frame-ancestors CSP distinct from main app per doc 07

#### Epic: MTTA/MTTR analytics dashboard (`phase-5-analytics`)

Tremor-based analytics in web app.

- **`p5-analytics-metrics`** — MTTA/MTTR metric computation and API
  - Size: L | Depends on: `p3-incident-timeline-ui`
  - Acceptance criteria:
    - Metrics API returns 7d and 30d rollups
    - Computation excludes maintenance-window alerts when configured

- **`p5-analytics-dashboard-ui`** — Analytics dashboard UI with Tremor charts
  - Size: M | Depends on: `p5-analytics-metrics`
  - Acceptance criteria:
    - Charts use packages/ui/charts wrappers
    - Dashboard loads under 3s with 10k alert seed dataset

#### Epic: Multi-org support (MSP use case) (`phase-5-multi-org`)

Multiple organizations in one self-hosted deployment.

- **`p5-multi-org-model`** — Multi-org data model and org switcher API
  - Size: L | Depends on: `p5-saml-sso`
  - Acceptance criteria:
    - Member in org A cannot query org B resources (FORBIDDEN)
    - Admin role is per-organization not global

- **`p5-multi-org-ui`** — Org switcher UI and per-org settings
  - Size: M | Depends on: `p5-multi-org-model`
  - Acceptance criteria:
    - Switcher lists only orgs user belongs to
    - Switching org refetches urql cache

#### Epic: Audit log UI and ticketing integrations (`phase-5-audit-enterprise`)

Admin audit UI and optional Jira/ServiceNow outbound.

- **`p5-audit-log-ui`** — Admin audit log list and export
  - Size: M | Depends on: `p0-audit-events-auth`
  - Acceptance criteria:
    - Audit list filterable by action type and date range
    - Export produces CSV with RFC3339 timestamps

- **`p5-jira-servicenow-outbound`** — Jira/ServiceNow ticket on incident declare (optional)
  - Size: L | Depends on: `p3-incident-promote-manual`
  - Note: Build cost permitting per doc 02; may defer if schedule pressure.
  - Acceptance criteria:
    - Ticket URL stored on incident record
    - Missing integration config skips silently with info log

---
