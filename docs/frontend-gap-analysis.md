# Frontend gap analysis — `apps/web` & `apps/status-page`

**Date:** 2026-07-27  
**Revision:** 2 (sub-agent audit round)  
**Plan coverage:** Round 3 verified 2026-07-27 (pre-import) — see [`frontend-refactor-plan.md` § Round 3 verification](./frontend-refactor-plan.md#round-3-verification-pre-import-2026-07-27)  
**Active plan:** [`frontend-refactor-plan.md`](./frontend-refactor-plan.md) (74 leaf tasks)  
**Sources:** Four parallel codebase audits + Outline Escalite collection (docs 00–06)

Sub-agents audited: `apps/web` routes/components/GraphQL usage, `apps/status-page` public app + admin gaps, Outline spec UI requirements, and full API-vs-UI matrix (`packages/schema` vs `packages/ts-types` vs apps).

---

## Executive summary

**Yes — the current frontend is a scaffold with selective vertical slices, not a product-complete operator console.**

| Layer | Maturity |
|-------|----------|
| **GraphQL API** (`packages/schema`) | ~90% of MVP operator surface |
| **Frontend codegen** (`packages/ts-types`) | ~60% of schema ops |
| **`apps/web` admin UI** | ~40% of codegen ops actually used; many screens partial |
| **`apps/status-page` public UI** | ~80% of roadmap acceptance |
| **`apps/web` status-page admin** | **0%** |

The largest blockers are not just missing pages — several gaps require **new GraphQL operations** (team CRUD, user directory) before UI can be built at all.

### Headline gaps (confirmed by sub-agents)

| Area | Verdict |
|------|---------|
| **Teams management** | Missing UI **and** missing GraphQL mutations |
| **User directory / invite / notification prefs** | Missing UI; several API ops lack frontend codegen |
| **Status page admin** | Missing entirely; 8 GraphQL ops have no codegen and zero web usage |
| **Escalation policy UI** | Editor exists but `users: []`, `schedules: []`; orphaned route; no delete |
| **Heartbeat monitors** | API complete; no codegen, no UI |
| **Schedule/rotation setup** | Calendar + overrides only; 6 CRUD mutations have no codegen |
| **Dashboard** | Placeholder — no `OnCallWidget`, no open-alert summary |
| **Integrations standalone page** | Requires pasting raw Service UUID |

---

## Methodology (round 2)

1. **Web audit** — all 15 routes, 12 components, 13 lib modules; GraphQL hook usage vs `packages/ts-types`; i18n gaps; UX anti-patterns.
2. **Status-page audit** — public app features, REST integration, admin gaps in `apps/web`, subscription email delivery gap.
3. **Spec extraction** — Outline docs 00/02/05/06 UI requirements by feature area with roadmap phases and acceptance criteria.
4. **API matrix** — all 36 Query / 55 Mutation / 3 Subscription schema fields mapped to codegen coverage and `apps/web` usage.

---

## `apps/web` — route inventory

| Route | Auth | Maturity | Notes |
|-------|------|----------|-------|
| `/login` | guest | **Functional** | Email/password + SAML/OIDC links |
| `/login/mobile` | none | **Functional** | REST code exchange → deep link |
| `/setup` | guest | **Functional** | First-admin bootstrap |
| `/dashboard` | protected | **Partial** | Placeholder; explicitly says features "arrive in later phases" |
| `/alerts/:alertId?` | protected | **Functional** | List/detail, ack/close/promote, `alertUpdated` subscription |
| `/incidents/:incidentId?` | protected | **Functional** | Timeline, roles, notes, postmortem export, subscription |
| `/services`, `/services/:id` | protected | **Functional** | CRUD, tabs (general, escalation, integrations, maintenance, schedules) |
| `/schedules/:id` | protected | **Functional** | `ScheduleCalendar` + overrides; not in nav |
| `/services/:id/escalation-policies/:id` | protected | **Partial** | Custom shell (not `AppShell`); empty target options |
| `/integrations` | protected | **Partial** | Same panel as service tab but requires **Service UUID** input |
| `/analytics` | protected | **Functional** | MTTA/MTTR Tremor charts, team/service filters |
| `/audit-log` | admin | **Functional** | Filter, paginate, CSV export |
| `/settings` | protected | **Partial** | SAML, SCIM, Slack, mobile devices; no notification rules |
| `*`, `/` | — | redirect | Silent redirect to `/dashboard` — no 404 page |

### Nav vs routes

**In nav:** Dashboard, Alerts, Incidents, Services, Integrations, Analytics, Audit log (admin), Settings.

**Implemented but not in nav:** `/schedules/:id`, escalation policy editor.

**Missing from nav and routes:** Teams, Users, Schedules index, Status pages, Heartbeats, Notification rules.

---

## `apps/web` — components & lib

### Components (`src/components/`)

| Component | Purpose |
|-----------|---------|
| `app-shell.tsx` | Nav, org switcher, page title |
| `auth-layout.tsx` | Login/setup card wrapper |
| `org-switcher.tsx` | Multi-org switch (`switchOrganization`) |
| `integration-keys-panel.tsx` | Key CRUD; standalone mode needs Service UUID |
| `integration-picker.tsx` | Preset picker — **Beszel only** |
| `maintenance-windows-panel.tsx` | Maintenance window CRUD |
| `saml-settings-panel.tsx` | SAML metadata + enable |
| `scim-settings-panel.tsx` | SCIM token rotate + one-time reveal |
| `slack-settings-panel.tsx` | OAuth link + manual bot token |
| `alert-*-badge.tsx`, `copy-button.tsx` | Small utilities |

### Lib modules — usage notes

| Module | Status |
|--------|--------|
| `notification-channel-form.ts` | **Unused in app** (test only) — schema-driven channel forms never wired |
| `schedule.ts` | User labels = truncated UUIDs (`id.slice(0, 8)`), not names |
| `escalation-policy.ts` | Maps API → editor; API query omits step targets |
| `integration-presets.ts` | Single preset: Beszel |

### `packages/ui/domain` composites

| Spec component | In `packages/ui` | Used in web |
|----------------|------------------|-------------|
| `EscalationPolicyEditor` | Yes | Yes — but broken options |
| `ScheduleCalendar` | Yes | Yes |
| `OnCallWidget` | **No** | **No** |
| `AlertCard` | **No** | **No** — alerts use table + inline detail |

---

## `apps/status-page` — public app

### What works (~80% of `p5-status-page-app` acceptance)

| Feature | Status |
|---------|--------|
| `GET /api/v1/public/status/{slug}` | Functional |
| Component grid (4-status enum) | Functional |
| Active incidents + update timeline | Functional |
| Overall status banner | Functional |
| Email subscribe form | Functional (stores email) |
| Polling (default 60s) | Functional |
| CSP `frame-ancestors` (per-org + nginx) | Functional |
| Public API excludes internal fields | Verified by API integration tests |
| Disabled pages return 404 | Functional |

### Public app gaps

| Gap | Severity |
|-----|----------|
| **Subscription email delivery** | High — emails stored in DB; no worker sends notifications on incident create/update |
| Resolved incident history | Medium — API returns only non-resolved incidents |
| `resolvedAt` not rendered | Low |
| Dynamic `<title>` / OG meta per slug | Low |
| Unsubscribe flow | Missing |
| Auto component status from linked `serviceId` | Not implemented |
| E2E browser tests | Missing |
| COSS particle alignment (`p-card-1` etc.) | Uses `Frame` instead |
| Dark mode as first-class default (spec 05) | Not explicit |

### Admin UI (`apps/web`) — **0%**

Sub-agent confirmed **zero matches** in `apps/web` for `statusPage`, `saveStatusPage`, `publishIncidentToStatusPage`, etc.

| Expected admin surface | Status |
|------------------------|--------|
| Route / nav entry | Missing |
| `statusPage` query + all 7 mutations | No frontend codegen |
| Slug, title, enabled, `frameAncestorsCsp` editor | Missing |
| Component CRUD + service linking + reorder | Missing |
| Publish incident → status page | Missing on `/incidents` |
| Post public updates / resolve | Missing |
| View subscriptions | Missing |
| Preview public URL | Missing |
| `packages/ui/domain` status-page composite | Missing |

**Impact:** Status pages only configurable via GraphQL API, integration tests, or direct DB.

---

## API vs UI matrix

### Coverage summary

| Layer | Count |
|-------|------:|
| Schema Query fields | 36 |
| Schema Mutation fields | 55 |
| Schema Subscription fields | 3 |
| In `packages/ts-types` codegen | 33 Q / 38 M / 3 S |
| Schema ops **without** codegen | 3 Q / **17 M** / 0 S |
| ts-types hooks **used** in `apps/web` | 52 |
| ts-types hooks **unused** in `apps/web` | 3 (`Health`, `RegisterMobileDevice`, `OnCallUpdated`) |

### Schema operations with NO frontend codegen (blocks UI without ts-types work)

**Queries:** `heartbeatMonitor`, `heartbeatMonitors`, `maintenanceWindow`, `notificationChannels`, `notificationRules`, `statusPage`, `analyticsSettings`

**Mutations:** All heartbeat CRUD; `deleteEscalationPolicy`; `snoozeAlert`, `reEscalateAlert`; all schedule/rotation CRUD (`createSchedule` … `deleteRotation`); all notification pref ops; `createIncident`; all incident role definition CRUD; `unassignIncidentRole`; **all 7 status-page admin mutations**; `saveAnalyticsSettings`

### Missing from GraphQL entirely (blocks UI at API layer)

| Capability | Impact |
|------------|--------|
| `createTeam`, `updateTeam`, `deleteTeam` | Cannot manage teams in-app |
| Team membership mutations | Cannot add/remove members |
| `users` / `organizationMembers` query | No user picker for escalation, schedules, roles |
| User invite / role assignment | Cannot onboard teammates |
| Password reset mutation | No self-serve reset (roadmap `p0-auth-password-reset`) |
| Org settings mutations | No org rename/profile |

`teams` query is **read-only** — used only for service-creation dropdown.

### Highest-leverage API-ready gaps (schema exists, UI absent)

| Priority | Area | Blocker |
|----------|------|---------|
| P0 | Team + user management | **New GraphQL ops needed** |
| P1 | Status page admin | Codegen + full admin route |
| P1 | Heartbeat monitors | Codegen + service tab |
| P1 | Schedule/rotation CRUD | Codegen + `/schedules` index + create flows |
| P1 | Notification rules + contact methods | Codegen + user settings |
| P1 | Wire escalation editor | User/schedule queries + populate options |
| P2 | Alert snooze / re-escalate | Codegen + alert detail buttons |
| P2 | `deleteEscalationPolicy` | Codegen + service tab |
| P2 | Incident role definition admin | Codegen + settings route |
| P2 | `onCallUpdated` subscription | Codegen exists — wire to dashboard/widget |
| P2 | `saveAnalyticsSettings` | Codegen + analytics settings toggle |

---

## Gap analysis by feature area

### 1. Teams & organization — **MISSING (API + UI)**

**Spec:** Org → Team hierarchy; services/schedules/incidents team-scoped; admin sees all teams, members scoped.

**Today:**
- `teams` query populates service-creation dropdown only.
- Teams created at `setup` or via SCIM — not manageable in product UI.
- If `teams` is empty, create-service flow has empty team select with no recovery path.

---

### 2. Users & access — **MISSING**

**Spec:** Contact methods, per-priority notification rules, device management, user directory for pickers.

**Today:**
- Mobile device list + revoke in settings.
- `notification-channel-form.ts` exists but unused.
- No user list, invite, profile, or notification rules UI.
- No password reset UI.
- No logout button in app shell.

---

### 3. Dashboard — **PLACEHOLDER**

**Spec (05):** `OnCallWidget`, open alert summary, dense ops landing, realtime on-call.

**Today:** Static welcome text + three links. Dashboard copy: *"Incident and alerting features arrive in later phases."*

---

### 4. Alerts — **~60%**

**Good:** Status filters with counts, master-detail, ack/close/promote, `alertUpdated` subscription, `eventCount`/dedup key.

**Gaps:**
- `snoozeAlert`, `reEscalateAlert` — API exists, no codegen, no UI.
- Service shown as raw UUID in detail (not linked to `/services/:id`).
- No escalation step / `next_escalation_at` display.
- No notification attempt delivery status.
- No `AlertCard` domain component.

---

### 5. Incidents — **~55%**

**Good:** List/detail, status workflow, role assignment, timeline + notes, `incidentTimelineUpdated` subscription, Markdown postmortem download.

**Gaps:**
- No `createIncident` UI (promote-from-alert only).
- No incident role definition admin (`createIncidentRoleDefinition` etc.).
- `unassignIncidentRole` not wired — roles can be assigned but not removed.
- No publish-to-status-page from incident detail.
- No Slack channel/thread UI (Phase 3).

---

### 6. Services — **~70%**

**Good:** Searchable list, CRUD, tabs, auto-promote rule, maintenance windows, embedded integration keys.

**Gaps:**
- Cannot change team after creation.
- No heartbeat tab.
- Escalation tab = plain policy name links only.
- No `deleteEscalationPolicy` UI.

---

### 7. Escalation policies — **PARTIAL / NOT PRODUCT-READY**

**Good:** `EscalationPolicyEditor` with DnD reorder; create/update mutations wired.

**Gaps:**
- `options={{ users: [], schedules: [] }}` — targets cannot be configured.
- API `EscalationPolicy` query omits step targets; editor warns on empty load.
- Orphaned route outside `AppShell`.
- `deleteEscalationPolicy` — no codegen, no UI.
- Repeat/max-repeat settings not exposed.

---

### 8. Schedules & on-call — **~35%**

**Good:** `ScheduleCalendar` on `/schedules/:id`; on-call layers; override create/delete.

**Gaps:**
- No `/schedules` index, no create schedule UI.
- No rotation CRUD (`createRotation` … `deleteRotation` — no codegen).
- No iCal export button.
- `onCallUpdated` subscription unused — manual refresh only.
- User labels = truncated UUIDs.
- No `OnCallWidget` on dashboard.

---

### 9. Heartbeat monitors — **MISSING UI**

**Spec (day 1):** Per-service dead man's switch with interval, grace, ping URL (shown once), status.

**Today:** Full GraphQL CRUD in schema; **no codegen, no route, no service tab**.

---

### 10. Integrations — **~40%**

**Good:** Key create/rotate/revoke with one-time secret reveal; embedded panel on service page.

**Gaps:**
- Standalone `/integrations` requires pasting Service UUID.
- Only Beszel preset — spec calls for Alertmanager, generic-webhook, Grafana, Datadog, Uptime Kuma, etc.
- No JSON Schema-driven plugin config forms (spec: `ConfigSchema()` drives UI automatically).
- `notification-channel-form.ts` unused.

---

### 11. Notifications — **~10%**

**Spec:** Schema-driven channel config forms; per-user rules by priority; Slack DM, email, push, webhook.

**Today:** Slack bot token panel only. No `notificationChannels`, `notificationRules`, `saveUserContactMethod`, `saveNotificationRule` in codegen or UI.

---

### 12. Analytics — **~50%**

**Good:** MTTA/MTTR + volume charts, team/service filters.

**Gaps:** `analyticsSettings` / `saveAnalyticsSettings` — no maintenance-exclusion toggle UI. Chart legend strings hardcoded English.

---

### 13. Settings & enterprise — **~60%**

| Setting | UI |
|---------|-----|
| SAML | Panel |
| SCIM | Token rotate |
| Slack | OAuth link + manual token |
| Mobile devices | List + revoke |
| OIDC | Login button only |
| Org profile | Missing |
| Notification rules | Missing |

---

### 14. Auth — **~70%**

| Flow | UI |
|------|-----|
| Setup | Yes |
| Login (email/OIDC/SAML links) | Yes |
| Mobile deep link | Yes |
| Password reset | **Missing** |
| Logout | **Missing from shell** |
| Session management | Missing |

---

### 15. Design system & UX polish — **GAPS**

Per Outline doc 05:

| Pattern | Spec | Today |
|---------|------|-------|
| Dark mode default | Required | Not verified in app shell |
| Breadcrumbs | `p-breadcrumb-3` | Minimal on service page only |
| Skeleton / empty states | Standard | Plain "Loading…" text |
| Toast feedback | All mutations | Not used in routes |
| i18n all strings | Required | **~40% hardcoded English** (dashboard, login, setup, integrations, escalation, integration-keys, analytics charts, auth loading) |
| Realtime on-call | `onCallUpdated` | Unused |
| Command palette | Post-MVP | Not started |
| Severity tokens | Colorblind-safe | Partial |

### UX anti-patterns (code evidence)

1. **Raw Service UUID** on `/integrations` (`integration-keys-panel.tsx`).
2. **Empty escalation target options** (`escalation-policy.tsx`).
3. **Truncated UUID user labels** in schedule calendar (`schedule.ts`).
4. **Service ID as UUID** in alert detail, not a link (`alerts.tsx`).
5. **Duplicate integrations flow** — nav `/integrations` inferior to service tab.
6. **Wildcard 404 → dashboard** — broken deep links give no error (`App.tsx`).
7. **Escalation page outside AppShell** — loses nav context.

---

## Roadmap task status (`p1-ui-*`, `p3-*`, `p5-*`)

| Task | UI status | Key gaps |
|------|-----------|----------|
| `p1-ui-integration-keys` | Partial | UUID input on standalone page; Beszel-only |
| `p1-ui-service-config` | Partial | No heartbeat tab; minimal escalation list |
| `p1-ui-alert-list-detail` | ~60% | No snooze/re-escalate; service as UUID |
| `p1-ui-escalation-editor` | Partial | Empty options; orphaned route; no delete |
| `p1-ui-schedule-calendar` | ~35% | View/overrides only; no CRUD |
| `p3-incident-timeline-ui` | ~55% | No role admin; no unassign; no status-page publish |
| `p3-postmortem-export` | Done | `postmortem-export.ts` |
| `p5-status-page-app` | Public ~80% / Admin 0% | No admin UI or codegen |
| `p5-analytics-dashboard-ui` | ~50% | No settings toggle |
| `p5-saml-sso` | ~70% | Config panel exists |
| `p5-scim-provisioning` | Partial | Token rotate only |
| `p5-multi-org-ui` | ~40% | Switcher only |
| `p5-audit-log-ui` | Functional | — |

---

## What works (genuine vertical slices)

1. Auth bootstrap — setup, login, protected routes, org switcher.
2. Service CRUD — list/search/create/delete (given teams exist).
3. Alert list/detail — subscriptions, ack/close/promote, dedup count.
4. Incident timeline — notes, status, roles, postmortem export, subscription.
5. Maintenance windows — full CRUD on service page.
6. Integration keys — per-service create/rotate/revoke with secret reveal.
7. Schedule calendar — on-call + overrides for existing schedules.
8. Audit log — admin filter + CSV export.
9. Analytics — basic MTTA/MTTR charts.
10. Public status page — component grid, incidents, subscribe, polling.
11. E2E — `z-alert-flow.spec.ts` (service → API alert → ack). Login smoke only otherwise.

---

## Suggested build order

### Phase A — Unblock self-serve (requires API work)

1. **GraphQL team CRUD + membership mutations**
2. **GraphQL user directory query** (for pickers everywhere)
3. **User invite / role assignment**

### Phase B — Wire existing API (codegen + UI)

4. **Escalation editor** — populate users/schedules; integrate in service tab with `AppShell`; add delete.
5. **Schedule index + CRUD** — add missing ops to ts-types; `/schedules` route.
6. **Heartbeat tab** on service detail.
7. **Notification rules + contact methods** in user settings.
8. **Status page admin** — codegen all 8 ops; settings route or dedicated page.
9. **Service picker** on `/integrations` (remove UUID input).

### Phase C — Phase 1 parity + polish

10. Alert snooze / re-escalate.
11. Dashboard `OnCallWidget` + open alerts; wire `onCallUpdated`.
12. Dynamic integration config forms from JSON Schema.
13. iCal export on schedule page.
14. i18n sweep + toasts + skeletons + breadcrumbs.
15. Incident role definition admin; unassign role; publish to status page from incident.

---

## Appendix — sub-agent references

| Audit | Agent ID | Scope |
|-------|----------|-------|
| Web app | `f93f1710-0ce8-4483-be89-3225fdd8e66b` | Routes, components, GraphQL usage, i18n, UX |
| Status page | `0c7d63e6-2396-4302-bb12-020e70d08030` | Public app + admin gaps |
| Spec requirements | `7c406405-f7c4-488e-9b94-1e2e53334296` | Outline 00/02/05/06 UI requirements |
| API matrix | `ec5e9256-6a5a-463e-a9e7-ea8192e3bb3e` | Schema vs ts-types vs apps |

### File inventory

**`apps/web/src/routes/` (15):** `alerts`, `analytics`, `audit-log`, `dashboard`, `escalation-policy`, `incidents`, `integrations`, `login`, `login-mobile`, `schedule`, `service`, `services`, `settings`, `setup`

**`apps/status-page/src/routes/`:** `status-page.tsx` (+ landing in `App.tsx`)

**`packages/ui/domain/`:** `EscalationPolicyEditor/`, `ScheduleCalendar/` only
