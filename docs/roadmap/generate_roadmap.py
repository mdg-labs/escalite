#!/usr/bin/env python3
"""Generate roadmap.yaml and ROADMAP.md. Run from repo root: python docs/roadmap/generate_roadmap.py"""

from __future__ import annotations

import textwrap
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parent
YAML_PATH = ROOT / "roadmap.yaml"
MD_PATH = ROOT / "ROADMAP.md"

DOD = "Follows the DoD checklist in 08-implementation-decisions-and-conventions"
DOD_COMPOSE = (
    "Follows the DoD checklist in 08-implementation-decisions-and-conventions; "
    "docker compose up works from a clean checkout"
)
DOD_COMPOSE_NA = (
    "Follows the DoD checklist in 08-implementation-decisions-and-conventions; "
    "docker compose up gate N/A until compose stack exists (step 2 of Phase 0 bootstrap)"
)


def task(
    id: str,
    title: str,
    description: str,
    acceptance_criteria: list[str],
    depends_on: list[str] | None = None,
    spec_refs: list[str] | None = None,
    estimated_size: str = "M",
    labels: list[str] | None = None,
    definition_of_done: list[str] | None = None,
    external_dependency: dict | None = None,
    notes: str | None = None,
) -> dict:
    t: dict = {
        "id": id,
        "title": title,
        "description": description,
        "acceptance_criteria": acceptance_criteria,
        "depends_on": depends_on or [],
        "definition_of_done": definition_of_done or [DOD, DOD_COMPOSE],
        "estimated_size": estimated_size,
        "labels": labels or [],
        "spec_refs": spec_refs or [],
    }
    if external_dependency:
        t["external_dependency"] = external_dependency
    if notes:
        t["notes"] = notes
    return t


def epic(id: str, title: str, description: str, tasks: list[dict], spec_refs: list[str] | None = None) -> dict:
    return {
        "id": id,
        "title": title,
        "description": description,
        "spec_refs": spec_refs or [],
        "tasks": tasks,
    }


ROADMAP = {
    "version": 1,
    "project": "escalite",
    "phases": [
        {
            "id": "phase-0",
            "name": "Foundations",
            "source_doc": "06-roadmap-mvp-phasing",
            "epics": [
                epic(
                    "phase-0-repo-scaffold",
                    "Repo scaffolding & CI skeleton",
                    "Initialize the public AGPL monorepo structure, pinned tooling, and CI gates per doc 08 bootstrap step 1.",
                    [
                        task(
                            "p0-scaffold-monorepo",
                            "Initialize pnpm workspace + Turborepo + go.work + Taskfile",
                            "Create the root monorepo skeleton: pnpm-workspace.yaml (apps/*, services/*, packages/*), turbo.json with build/test/lint pipelines, go.work referencing all Go service modules, and Taskfile.yml exposing dev/build/test/lint/migrate.",
                            [
                                "pnpm-workspace.yaml defines apps/*, services/*, packages/*",
                                "turbo.json configured with build, test, and lint pipelines",
                                "go.work includes services/api, services/engine, services/integrations",
                                "Taskfile.yml exposes: dev, build, test, lint, migrate",
                                "Root README documents clone → task dev prerequisites",
                            ],
                            depends_on=[],
                            definition_of_done=[DOD, DOD_COMPOSE_NA],
                            estimated_size="S",
                            labels=["phase-0", "infra", "monorepo"],
                            spec_refs=[
                                "08-implementation-decisions-and-conventions#pinned-tool-choices",
                                "01-architecture-and-monorepo",
                            ],
                        ),
                        task(
                            "p0-packages-config-shared",
                            "Add packages/config shared ESLint, Prettier, and tsconfig",
                            "Create packages/config with shared eslint.config, prettier config, and base tsconfig extended by apps and packages.",
                            [
                                "packages/config exports eslint and prettier configs consumable by apps/web",
                                "TypeScript strict mode enabled in base tsconfig",
                                "turbo lint task runs ESLint across JS/TS workspaces",
                            ],
                            depends_on=["p0-scaffold-monorepo"],
                            definition_of_done=[DOD, DOD_COMPOSE_NA],
                            estimated_size="S",
                            labels=["phase-0", "infra", "tooling"],
                            spec_refs=["08-implementation-decisions-and-conventions#pinned-tool-choices"],
                        ),
                        task(
                            "p0-license-agpl-dco",
                            "Add AGPL-3.0 LICENSE and DCO enforcement in CI",
                            "Add LICENSE (AGPL-3.0), CONTRIBUTING.md with DCO sign-off instructions, and a CI check that rejects commits without Signed-off-by on PRs.",
                            [
                                "LICENSE file is AGPL-3.0",
                                "CI fails PRs missing DCO sign-off per doc 04",
                                "CONTRIBUTING.md references conventional commits",
                            ],
                            depends_on=["p0-scaffold-monorepo"],
                            definition_of_done=[DOD, DOD_COMPOSE_NA],
                            estimated_size="S",
                            labels=["phase-0", "legal", "ci"],
                            spec_refs=["04-licensing-and-editions"],
                        ),
                        task(
                            "p0-ci-skeleton",
                            "GitHub Actions CI: lint, test, build, vuln scan",
                            "Add .github/workflows/ci.yml running golangci-lint, go test, pnpm lint/test/build, govulncheck, and pnpm audit/OSV on PRs.",
                            [
                                "CI runs on pull_request and push to main",
                                "govulncheck and pnpm audit fail on known-critical vulns per doc 07",
                                "Renovate or Dependabot config present",
                            ],
                            depends_on=["p0-scaffold-monorepo", "p0-packages-config-shared"],
                            definition_of_done=[DOD, DOD_COMPOSE_NA],
                            estimated_size="M",
                            labels=["phase-0", "ci", "security"],
                            spec_refs=[
                                "08-implementation-decisions-and-conventions#pinned-tool-choices",
                                "07-security-and-auth#dependency-supply-chain",
                            ],
                        ),
                        task(
                            "p0-go-service-skeletons",
                            "Scaffold Go service modules (api, engine, integrations)",
                            "Create minimal main packages for services/api, services/engine, services/integrations with chi router stub, slog JSON logging, ESCALITE_ config parsing, and /healthz endpoints.",
                            [
                                "Each service builds to a static binary via go build",
                                "Config package fails fast with actionable errors for missing required env",
                                "Structured logs include org_id/service_id keys when available",
                            ],
                            depends_on=["p0-scaffold-monorepo"],
                            estimated_size="M",
                            labels=["phase-0", "backend", "go"],
                            spec_refs=[
                                "01-architecture-and-monorepo",
                                "08-implementation-decisions-and-conventions#conventions",
                            ],
                        ),
                        task(
                            "p0-compose-dev-prod",
                            "Docker Compose dev + prod profiles (Postgres, api, engine, web)",
                            "Add deploy/docker-compose/docker-compose.yml with dev profile (hot reload mounts) and prod profile (built images). Services: postgres, api, engine, web. Include .env.example with all ESCALITE_ vars documented.",
                            [
                                "docker compose --profile dev up starts postgres, api, engine, web",
                                "docker compose --profile prod up uses built images tagged locally",
                                ".env.example documents ESCALITE_ENCRYPTION_KEY generation one-liner",
                                "Optional commented Caddy reverse-proxy service per doc 07",
                            ],
                            depends_on=["p0-go-service-skeletons"],
                            estimated_size="M",
                            labels=["phase-0", "infra", "compose"],
                            spec_refs=[
                                "01-architecture-and-monorepo#deployment-targets",
                                "08-implementation-decisions-and-conventions#bootstrap-order",
                            ],
                        ),
                        task(
                            "p0-dockerfiles-nonroot",
                            "Distroless/slim Dockerfiles with non-root user and pinned bases",
                            "Add multi-stage Dockerfiles for api, engine, web using slim/distroless bases, non-root USER, and document digest pinning for release compose.",
                            [
                                "Containers run as non-root UID",
                                "Release compose documents image digest pinning approach",
                            ],
                            depends_on=["p0-compose-dev-prod"],
                            estimated_size="S",
                            labels=["phase-0", "infra", "security"],
                            spec_refs=["07-security-and-auth#dependency-supply-chain"],
                        ),
                    ],
                    spec_refs=["08-implementation-decisions-and-conventions#bootstrap-order", "01-architecture-and-monorepo"],
                ),
                epic(
                    "phase-0-schema",
                    "Postgres schema v1",
                    "Goose migrations and sqlc foundation for core domain tables per doc 08 bootstrap step 3.",
                    [
                        task(
                            "p0-goose-migrations-setup",
                            "Wire goose migrations with advisory lock on API startup",
                            "Add services/api/migrations/ with goose; API runs migrations on startup guarded by Postgres advisory lock (single-node MVP).",
                            [
                                "goose up applies all SQL migrations idempotently",
                                "Concurrent API starts do not corrupt migrations (advisory lock)",
                                "task migrate runs migrations locally without starting full stack",
                            ],
                            depends_on=["p0-compose-dev-prod"],
                            definition_of_done=[DOD, DOD_COMPOSE],
                            estimated_size="S",
                            labels=["phase-0", "database", "migrations"],
                            spec_refs=["08-implementation-decisions-and-conventions#pinned-tool-choices"],
                        ),
                        task(
                            "p0-schema-core-entities",
                            "Migration: organizations, teams, users, team_memberships",
                            "Create tables: organizations, teams, users, team_memberships with UUIDv7 PKs, timestamptz created_at/updated_at, snake_case naming.",
                            [
                                "All PKs are UUID (v7 generated in app layer)",
                                "users.email unique per organization",
                                "team_memberships enforces user belongs to org of team",
                            ],
                            depends_on=["p0-goose-migrations-setup"],
                            estimated_size="M",
                            labels=["phase-0", "database", "schema"],
                            spec_refs=["02-core-domain-and-features#domain-model", "08-implementation-decisions-and-conventions#conventions"],
                        ),
                        task(
                            "p0-schema-service-integration",
                            "Migration: services, integration_keys",
                            "Add services and integration_keys tables; integration key stores hashed token, display prefix, plugin name, JSON config, revoked_at.",
                            [
                                "integration_keys.token stores only hash; prefix column for UI last-4 style",
                                "integration_keys.plugin_name references compile-time registry names",
                            ],
                            depends_on=["p0-schema-core-entities"],
                            estimated_size="M",
                            labels=["phase-0", "database", "schema"],
                            spec_refs=["02-core-domain-and-features#domain-model", "07-security-and-auth#inbound-webhook-ingestion"],
                        ),
                        task(
                            "p0-schema-scheduling",
                            "Migration: schedules, rotations, overrides",
                            "Add schedules (IANA timezone), rotations (RRULE text, layer, participants JSON), overrides (start/end, user swap, audit fields).",
                            [
                                "schedules.timezone stores valid IANA zone string",
                                "overrides support soft-delete with deleted_at where product requires audit retention",
                            ],
                            depends_on=["p0-schema-core-entities"],
                            estimated_size="M",
                            labels=["phase-0", "database", "schema"],
                            spec_refs=["02-core-domain-and-features#scheduling"],
                        ),
                        task(
                            "p0-schema-escalation-alerts",
                            "Migration: escalation_policies, steps, alerts, notification_attempts",
                            "Add escalation_policies, escalation_steps (ordered, delay_minutes, repeat config), alerts (status, dedup_key, escalation_state JSON), notification_attempts.",
                            [
                                "alerts.status enum: triggered, acknowledged, closed",
                                "alerts.dedup_key indexed per service_id",
                                "escalation_steps.order unique per policy",
                            ],
                            depends_on=["p0-schema-service-integration"],
                            estimated_size="M",
                            labels=["phase-0", "database", "schema"],
                            spec_refs=["02-core-domain-and-features#escalation-engine-semantics"],
                        ),
                        task(
                            "p0-schema-sessions-audit",
                            "Migration: sessions, audit_events, refresh_tokens",
                            "Add server-side sessions (HttpOnly cookie backing), audit_events append-only, refresh_tokens for mobile devices with revocation.",
                            [
                                "sessions store expires_at and user_agent fingerprint",
                                "audit_events capture actor_id, action, target_type, target_id, metadata JSON",
                                "refresh_tokens individually revocable",
                            ],
                            depends_on=["p0-schema-core-entities"],
                            estimated_size="M",
                            labels=["phase-0", "database", "schema", "security"],
                            spec_refs=["07-security-and-auth#authn-mvp", "07-security-and-auth#audit-log"],
                        ),
                        task(
                            "p0-sqlc-foundation",
                            "sqlc queries for auth, org, and health checks",
                            "Configure sqlc for services/api; generate typed queries for user lookup, session CRUD, org bootstrap, and migration health.",
                            [
                                "sqlc generate produces compiles Go code",
                                "Integration test uses testcontainers-go Postgres",
                            ],
                            depends_on=["p0-schema-sessions-audit", "p0-schema-escalation-alerts"],
                            estimated_size="M",
                            labels=["phase-0", "database", "sqlc"],
                            spec_refs=["08-implementation-decisions-and-conventions#pinned-tool-choices"],
                        ),
                    ],
                    spec_refs=["08-implementation-decisions-and-conventions#bootstrap-order"],
                ),
                epic(
                    "phase-0-auth-slice",
                    "Auth vertical slice",
                    "First-admin setup, sessions, RBAC, optional OIDC, encryption at rest per doc 08 bootstrap step 4.",
                    [
                        task(
                            "p0-config-encryption-key",
                            "App config validation and ESCALITE_ENCRYPTION_KEY enforcement",
                            "Config package requires 32-byte ESCALITE_ENCRYPTION_KEY; refuse startup with placeholder/empty key and actionable error referencing setup docs.",
                            [
                                "API and engine refuse start when ESCALITE_ENCRYPTION_KEY missing or placeholder",
                                ".env.example shows openssl rand -hex 32 one-liner",
                            ],
                            depends_on=["p0-go-service-skeletons"],
                            estimated_size="S",
                            labels=["phase-0", "security", "config"],
                            spec_refs=["07-security-and-auth#secrets-handling"],
                        ),
                        task(
                            "p0-auth-password-argon2id",
                            "Password hashing with argon2id",
                            "Implement password hash/verify using alexedwards/argon2id with safe defaults; never log passwords.",
                            [
                                "Unit tests verify hash round-trip and rejection of wrong password",
                                "Password hashes stored only in users.password_hash",
                            ],
                            depends_on=["p0-schema-core-entities"],
                            estimated_size="S",
                            labels=["phase-0", "auth", "security"],
                            spec_refs=["07-security-and-auth#authn-mvp", "08-implementation-decisions-and-conventions#pinned-tool-choices"],
                        ),
                        task(
                            "p0-auth-first-admin-setup",
                            "First-admin setup flow (single-org bootstrap)",
                            "HTTP/GraphQL flow when zero users exist: create org + first admin user; disabled after first admin exists.",
                            [
                                "Setup endpoint returns 403 when any user exists",
                                "Successful setup creates org, admin user (role=admin), and session",
                                "Setup covered by integration test",
                            ],
                            depends_on=["p0-auth-password-argon2id", "p0-sqlc-foundation"],
                            estimated_size="M",
                            labels=["phase-0", "auth"],
                            spec_refs=["07-security-and-auth#authn-mvp"],
                        ),
                        task(
                            "p0-auth-login-sessions",
                            "Email+password login and Postgres session cookies",
                            "Login mutation sets HttpOnly+Secure+SameSite=Lax session cookie; logout invalidates session; session middleware on GraphQL.",
                            [
                                "Invalid credentials return GraphQL error code UNAUTHENTICATED without user enumeration leak",
                                "Session cookie not accessible to JS (HttpOnly)",
                                "Revoked session returns UNAUTHENTICATED on next request",
                            ],
                            depends_on=["p0-auth-first-admin-setup"],
                            estimated_size="M",
                            labels=["phase-0", "auth"],
                            spec_refs=["07-security-and-auth#authn-mvp"],
                        ),
                        task(
                            "p0-auth-rbac-guards",
                            "RBAC guards: admin vs member, team scoping",
                            "Implement org roles admin/member; members scoped to team_memberships; every mutation verifies membership server-side.",
                            [
                                "Non-member accessing team resource returns FORBIDDEN",
                                "Admin can access all teams in org",
                                "GraphQL errors use stable code extensions per doc 08",
                            ],
                            depends_on=["p0-auth-login-sessions"],
                            estimated_size="M",
                            labels=["phase-0", "auth", "rbac"],
                            spec_refs=["07-security-and-auth#authz-rbac"],
                        ),
                        task(
                            "p0-auth-oidc-optional",
                            "Optional generic OIDC login (ESCALITE_OIDC_*)",
                            "When OIDC_ISSUER_URL and client credentials set, enable Sign in with OIDC; no SCIM/group mapping (Phase 5).",
                            [
                                "OIDC disabled when env vars unset",
                                "OIDC login creates/links user and session same as password flow",
                                "Uses coreos/go-oidc and golang.org/x/oauth2",
                            ],
                            depends_on=["p0-auth-login-sessions"],
                            estimated_size="M",
                            labels=["phase-0", "auth", "oidc"],
                            spec_refs=["07-security-and-auth#authn-mvp"],
                        ),
                        task(
                            "p0-auth-password-reset",
                            "Password reset via email with rate limiting",
                            "Token-based password reset using SMTP config; rate limit reset requests per email/IP.",
                            [
                                "Reset email sent only if user exists (same response either way)",
                                "Expired reset tokens rejected with VALIDATION code",
                                "Rate limit returns RATE_LIMITED after threshold",
                            ],
                            depends_on=["p0-auth-login-sessions"],
                            estimated_size="M",
                            labels=["phase-0", "auth"],
                            spec_refs=["07-security-and-auth#authn-mvp"],
                        ),
                        task(
                            "p0-audit-events-auth",
                            "Audit log capture for auth and privileged actions",
                            "Write audit_events for login, failed login, logout, role changes, integration key lifecycle (stub hooks for later).",
                            [
                                "Failed login attempts logged with IP and user agent",
                                "audit_events table append-only (no UPDATE/DELETE in app code)",
                            ],
                            depends_on=["p0-auth-rbac-guards"],
                            estimated_size="S",
                            labels=["phase-0", "audit", "security"],
                            spec_refs=["07-security-and-auth#audit-log"],
                        ),
                        task(
                            "p0-secrets-encryption-at-rest",
                            "AES-256-GCM encryption helper for provider credentials",
                            "Implement encrypt/decrypt with key-id column for provider secrets; UI shows last-4 hints only after save.",
                            [
                                "Round-trip encrypt/decrypt integration test",
                                "Decrypt fails with wrong key id with actionable error",
                            ],
                            depends_on=["p0-config-encryption-key", "p0-sqlc-foundation"],
                            estimated_size="M",
                            labels=["phase-0", "security"],
                            spec_refs=["07-security-and-auth#secrets-handling"],
                        ),
                    ],
                    spec_refs=["07-security-and-auth", "08-implementation-decisions-and-conventions#bootstrap-order"],
                ),
                epic(
                    "phase-0-graphql-skeleton",
                    "GraphQL/codegen skeleton + web shell",
                    "SDL, gqlgen, codegen to ts-types, packages/ui primitives, Vite web shell per doc 08 bootstrap step 5.",
                    [
                        task(
                            "p0-schema-sdl-foundation",
                            "packages/schema GraphQL SDL foundation",
                            "Add packages/schema with Query (me, health), Mutation (login, setup), and core types User, Organization, Team stub fields.",
                            [
                                "SDL is single source of truth in packages/schema",
                                "Schema validates with gqlgen init",
                            ],
                            depends_on=["p0-auth-rbac-guards"],
                            estimated_size="S",
                            labels=["phase-0", "graphql", "schema"],
                            spec_refs=["01-architecture-and-monorepo#api-layer", "08-implementation-decisions-and-conventions#pinned-tool-choices"],
                        ),
                        task(
                            "p0-gqlgen-wiring",
                            "gqlgen server wiring in services/api",
                            "Wire gqlgen with chi; session middleware; GraphQL POST at /graphql; production disables introspection for unauthenticated requests.",
                            [
                                "Depth and complexity limits configured",
                                "GraphQL errors include extensions.code per doc 08",
                            ],
                            depends_on=["p0-schema-sdl-foundation"],
                            estimated_size="M",
                            labels=["phase-0", "graphql", "api"],
                            spec_refs=["08-implementation-decisions-and-conventions#pinned-tool-choices", "07-security-and-auth#transport-hardening"],
                        ),
                        task(
                            "p0-graphql-codegen-ts",
                            "graphql-codegen → packages/ts-types",
                            "Configure graphql-codegen to emit typed urql hooks/document nodes into packages/ts-types consumed by web (and later mobile).",
                            [
                                "pnpm codegen produces packages/ts-types without manual edits",
                                "turbo build depends on codegen output",
                            ],
                            depends_on=["p0-schema-sdl-foundation"],
                            estimated_size="S",
                            labels=["phase-0", "graphql", "typescript"],
                            spec_refs=["08-implementation-decisions-and-conventions#pinned-tool-choices"],
                        ),
                        task(
                            "p0-packages-ui-primitives",
                            "packages/ui primitives (shadcn base + tokens + dark mode)",
                            "Stand up packages/ui with shadcn-derived Button, Dialog, Input, Toast; Tailwind tokens; dark mode default.",
                            [
                                "Dark mode toggles via class on html element",
                                "Severity colors documented as colorblind-safe in tokens",
                            ],
                            depends_on=["p0-packages-config-shared"],
                            estimated_size="M",
                            labels=["phase-0", "ui", "design-system"],
                            spec_refs=["05-ui-design-system", "06-roadmap-mvp-phasing#phase-0"],
                        ),
                        task(
                            "p0-web-app-shell",
                            "apps/web Vite shell: login, setup, empty dashboard",
                            "React Router app with login page, first-admin setup page, authenticated dashboard shell using urql and packages/ui.",
                            [
                                "Unauthenticated users redirect to /login",
                                "Successful login lands on /dashboard with user name displayed",
                                "CORS locked to configured app origin",
                            ],
                            depends_on=["p0-gqlgen-wiring", "p0-graphql-codegen-ts", "p0-packages-ui-primitives"],
                            estimated_size="M",
                            labels=["phase-0", "web", "frontend"],
                            spec_refs=["05-ui-design-system", "08-implementation-decisions-and-conventions#bootstrap-order"],
                        ),
                    ],
                    spec_refs=["08-implementation-decisions-and-conventions#bootstrap-order", "05-ui-design-system"],
                ),
                epic(
                    "phase-0-queue",
                    "river job queue wiring",
                    "Prove Postgres-backed job queue path per doc 08 bootstrap step 6.",
                    [
                        task(
                            "p0-river-client-setup",
                            "river client and worker registration in engine service",
                            "Add river queue client using same Postgres DSN; worker process in services/engine with graceful shutdown.",
                            [
                                "river migrations applied on engine startup",
                                "Worker logs job start/finish with structured slog fields",
                            ],
                            depends_on=["p0-goose-migrations-setup", "p0-go-service-skeletons"],
                            estimated_size="M",
                            labels=["phase-0", "queue", "river"],
                            spec_refs=["01-architecture-and-monorepo#job-scheduling", "08-implementation-decisions-and-conventions#pinned-tool-choices"],
                        ),
                        task(
                            "p0-river-noop-heartbeat-job",
                            "First noop heartbeat scan job to prove queue path",
                            "Register a periodic noop HeartbeatScan job that logs and exits; enqueue on schedule to validate end-to-end river wiring.",
                            [
                                "Job enqueued and processed within 60s in dev compose",
                                "Failed jobs retry per river defaults with logged error",
                            ],
                            depends_on=["p0-river-client-setup"],
                            estimated_size="S",
                            labels=["phase-0", "queue", "river"],
                            spec_refs=["08-implementation-decisions-and-conventions#bootstrap-order"],
                        ),
                    ],
                    spec_refs=["08-implementation-decisions-and-conventions#bootstrap-order"],
                ),
                epic(
                    "phase-0-ci-smoke",
                    "First E2E smoke in CI",
                    "Playwright smoke and compose-up gate per doc 08 bootstrap step 7.",
                    [
                        task(
                            "p0-playwright-setup",
                            "Playwright test harness against compose stack",
                            "Add apps/web e2e/ with Playwright config targeting docker compose dev profile URL.",
                            [
                                "pnpm e2e runs Playwright headless in CI",
                                "Tests wait for /healthz before interactions",
                            ],
                            depends_on=["p0-web-app-shell", "p0-compose-dev-prod"],
                            estimated_size="S",
                            labels=["phase-0", "e2e", "testing"],
                            spec_refs=["08-implementation-decisions-and-conventions#pinned-tool-choices"],
                        ),
                        task(
                            "p0-e2e-smoke-setup-login",
                            "E2E smoke: first-admin setup → login → dashboard",
                            "Playwright test completes setup flow on fresh DB, logs out, logs in, sees dashboard heading.",
                            [
                                "Test runs against ephemeral Postgres in CI",
                                "Test artifacts uploaded on failure",
                            ],
                            depends_on=["p0-playwright-setup", "p0-auth-first-admin-setup"],
                            estimated_size="M",
                            labels=["phase-0", "e2e", "auth"],
                            spec_refs=["08-implementation-decisions-and-conventions#bootstrap-order"],
                        ),
                        task(
                            "p0-ci-compose-gate",
                            "CI job: docker compose up end-to-end gate",
                            "CI brings up compose prod profile, runs migrations, executes E2E smoke, tears down.",
                            [
                                "CI fails if docker compose up exits non-zero",
                                "CI runs E2E smoke as required check on main",
                            ],
                            depends_on=["p0-e2e-smoke-setup-login", "p0-ci-skeleton"],
                            estimated_size="M",
                            labels=["phase-0", "ci", "compose"],
                            spec_refs=["08-implementation-decisions-and-conventions#definition-of-done"],
                        ),
                    ],
                    spec_refs=["08-implementation-decisions-and-conventions#bootstrap-order"],
                ),
            ],
        },
        # Phase 1 continues in separate write due to size - will be appended
    ],
    "external_dependencies": [],
}


def phase_1_epics() -> list[dict]:
    return [
        epic(
            "phase-1-escalation-engine",
            "Escalation engine",
            "Trigger → notify → escalate → ack/close state machine with river timers per doc 02 and Phase 1 scope.",
            [
                task(
                    "p1-escalation-policy-crud",
                    "Escalation policy and step CRUD API",
                    "GraphQL mutations and queries for escalation policies and ordered steps per service.",
                    [
                        "Admin can create policy with ordered steps and delay_minutes",
                        "Step order gaps rejected with VALIDATION",
                        "Policy changes emit audit_events",
                    ],
                    depends_on=["p0-ci-compose-gate"],
                    estimated_size="M",
                    labels=["phase-1", "escalation", "api"],
                    spec_refs=["02-core-domain-and-features#escalation-engine-semantics"],
                ),
                task(
                    "p1-escalation-trigger-notify",
                    "Alert trigger schedules step-1 notifications",
                    "On alert triggered status, engine enqueues notify job for step 1 targets immediately.",
                    [
                        "Triggered alert creates notification_attempt rows per target/channel",
                        "Integration test with testcontainers verifies step-1 enqueue within 5s",
                    ],
                    depends_on=["p1-escalation-policy-crud", "p0-river-noop-heartbeat-job"],
                    estimated_size="L",
                    labels=["phase-1", "escalation", "engine"],
                    spec_refs=["02-core-domain-and-features#escalation-engine-semantics"],
                ),
                task(
                    "p1-escalation-step-timers",
                    "Per-step delay timers via river",
                    "If unacknowledged after delay_minutes, escalate to next step; store current step and next_escalation_at on alert.",
                    [
                        "Timer fires and advances escalation_state.current_step",
                        "Acknowledged alert cancels pending escalation jobs",
                    ],
                    depends_on=["p1-escalation-trigger-notify"],
                    estimated_size="L",
                    labels=["phase-1", "escalation", "engine", "river"],
                    spec_refs=["02-core-domain-and-features#escalation-engine-semantics", "01-architecture-and-monorepo#job-scheduling"],
                ),
                task(
                    "p1-escalation-repeat-cap",
                    "Repeat-last-step with max repeat count",
                    "Support policy repeat flag with configurable max repeats; never infinite loop.",
                    [
                        "After max repeats, escalation stops and alert flagged escalated_exhausted",
                        "Unit test covers repeat boundary",
                    ],
                    depends_on=["p1-escalation-step-timers"],
                    estimated_size="M",
                    labels=["phase-1", "escalation"],
                    spec_refs=["02-core-domain-and-features#escalation-engine-semantics"],
                ),
                task(
                    "p1-escalation-ack-close",
                    "Acknowledge and close mutations",
                    "GraphQL mutations ack and close alerts; ack stops escalation but does not close.",
                    [
                        "ack sets status=acknowledged and records acked_by/at",
                        "close sets status=closed; closed alerts reject re-ack with VALIDATION",
                    ],
                    depends_on=["p1-escalation-step-timers"],
                    estimated_size="M",
                    labels=["phase-1", "escalation", "api"],
                    spec_refs=["02-core-domain-and-features#escalation-engine-semantics"],
                ),
                task(
                    "p1-escalation-manual-snooze",
                    "Manual snooze and re-escalate from UI/API",
                    "Allow operators to snooze escalation timer or force re-escalate; audit logged.",
                    [
                        "snooze shifts next_escalation_at by requested duration",
                        "reEscalate resets to step 1 and enqueues notifications",
                    ],
                    depends_on=["p1-escalation-ack-close"],
                    estimated_size="M",
                    labels=["phase-1", "escalation"],
                    spec_refs=["02-core-domain-and-features#escalation-engine-semantics"],
                ),
            ],
        ),
        epic(
            "phase-1-scheduling",
            "Schedules, rotations, and overrides",
            "RRULE-based on-call scheduling with overrides and DST tests.",
            [
                task(
                    "p1-schedule-crud",
                    "Schedule and rotation CRUD with RRULE storage",
                    "GraphQL CRUD for schedules and rotations; store RRULE text and participant order.",
                    [
                        "Invalid RRULE rejected with VALIDATION and human-readable message",
                        "Schedules require valid IANA timezone",
                    ],
                    depends_on=["p0-ci-compose-gate"],
                    estimated_size="M",
                    labels=["phase-1", "scheduling"],
                    spec_refs=["02-core-domain-and-features#scheduling"],
                ),
                task(
                    "p1-schedule-on-call-now",
                    "onCallNow query for schedule layers",
                    "Compute current on-call users per rotation layer using rrule-go in schedule timezone.",
                    [
                        "onCallNow returns primary and secondary layers when configured",
                        "Integration test covers DST spring-forward and fall-back boundaries",
                    ],
                    depends_on=["p1-schedule-crud"],
                    estimated_size="L",
                    labels=["phase-1", "scheduling", "rrule"],
                    spec_refs=["02-core-domain-and-features#scheduling", "08-implementation-decisions-and-conventions#pinned-tool-choices"],
                ),
                task(
                    "p1-schedule-overrides",
                    "Override CRUD with audit trail",
                    "Overrides take precedence over rotation; soft-delete retains history; optional approval flag stub for Phase 5.",
                    [
                        "Active override replaces rotation participant for overlapping window",
                        "Override create/delete writes audit_events",
                    ],
                    depends_on=["p1-schedule-crud"],
                    estimated_size="M",
                    labels=["phase-1", "scheduling"],
                    spec_refs=["02-core-domain-and-features#scheduling"],
                ),
                task(
                    "p1-schedule-ical-export",
                    "iCal export for schedule rotations",
                    "HTTP endpoint or GraphQL field exporting RFC 5545 calendar for a schedule.",
                    [
                        "Exported ICS validates against RFC 5545 test fixture",
                        "DTSTART/TZID match schedule timezone",
                    ],
                    depends_on=["p1-schedule-on-call-now"],
                    estimated_size="M",
                    labels=["phase-1", "scheduling", "ical"],
                    spec_refs=["02-core-domain-and-features#scheduling", "08-implementation-decisions-and-conventions#pinned-tool-choices"],
                ),
                task(
                    "p1-escalation-target-rotation",
                    "Escalation targets resolve rotation schedules to users",
                    "Escalation step targets of type RotationSchedule resolve to on-call users at notification time.",
                    [
                        "Step targeting rotation notifies current on-call user(s)",
                        "Empty rotation returns logged skip, not panic",
                    ],
                    depends_on=["p1-schedule-on-call-now", "p1-escalation-trigger-notify"],
                    estimated_size="M",
                    labels=["phase-1", "scheduling", "escalation"],
                    spec_refs=["02-core-domain-and-features#domain-model"],
                ),
            ],
        ),
        epic(
            "phase-1-heartbeat-monitors",
            "Heartbeat monitors (dead man's switch)",
            "GoAlert-equivalent heartbeat monitors per doc 02.",
            [
                task(
                    "p1-heartbeat-crud",
                    "HeartbeatMonitor CRUD per service",
                    "GraphQL CRUD for monitors: name, interval, grace period; status healthy/overdue/triggered.",
                    [
                        "Monitors belong to exactly one service",
                        "interval and grace must be positive durations",
                    ],
                    depends_on=["p0-ci-compose-gate"],
                    estimated_size="M",
                    labels=["phase-1", "heartbeat"],
                    spec_refs=["02-core-domain-and-features#heartbeat-monitors"],
                ),
                task(
                    "p1-heartbeat-ping-endpoint",
                    "Heartbeat ping endpoint with token auth",
                    "GET/POST /heartbeat/{token} idempotent ping; token ≥128-bit entropy; rate limited per token.",
                    [
                        "Invalid token returns 404 without leaking existence",
                        "Valid ping updates last_ping_at and sets status healthy",
                        "Token logged only as prefix per doc 07",
                    ],
                    depends_on=["p1-heartbeat-crud", "p0-secrets-encryption-at-rest"],
                    estimated_size="M",
                    labels=["phase-1", "heartbeat", "security"],
                    spec_refs=["02-core-domain-and-features#heartbeat-monitors", "07-security-and-auth#inbound-webhook-ingestion"],
                ),
                task(
                    "p1-heartbeat-deadline-scan",
                    "River periodic scan for overdue heartbeats",
                    "Replace noop job with scan: monitors past interval+grace flip overdue then triggered.",
                    [
                        "Overdue monitor creates alert with source=heartbeat and dedup_key=monitor id",
                        "Scan interval configurable via ESCALITE_HEARTBEAT_SCAN_INTERVAL",
                    ],
                    depends_on=["p1-heartbeat-ping-endpoint", "p1-escalation-trigger-notify"],
                    estimated_size="M",
                    labels=["phase-1", "heartbeat", "river"],
                    spec_refs=["02-core-domain-and-features#heartbeat-monitors"],
                ),
            ],
        ),
        epic(
            "phase-1-notification-channels",
            "Notification channels (push basic, email, webhook, Slack DM)",
            "Day-1 outbound channels without Twilio or Critical Alerts (Phase 2).",
            [
                task(
                    "p1-channel-registry",
                    "NotificationChannel plugin registry in engine",
                    "Compile-time registry in services/engine/channels/ with Name, Send, ValidateConfig, ConfigSchema.",
                    [
                        "Unknown channel name returns clear error at config save",
                        "ConfigSchema drives web UI form generation",
                    ],
                    depends_on=["p1-escalation-trigger-notify"],
                    estimated_size="M",
                    labels=["phase-1", "notifications", "plugins"],
                    spec_refs=["02-core-domain-and-features#notification-channels"],
                ),
                task(
                    "p1-channel-email",
                    "Email notification channel via SMTP",
                    "Send alert notifications via configured SMTP; HTML+text body; respects user notification rules.",
                    [
                        "SMTP misconfig surfaces actionable startup warning",
                        "Send failure records notification_attempt status failed with error",
                    ],
                    depends_on=["p1-channel-registry"],
                    estimated_size="M",
                    labels=["phase-1", "notifications", "email"],
                    spec_refs=["02-core-domain-and-features#notification-channels"],
                ),
                task(
                    "p1-channel-webhook-outbound",
                    "Generic outbound webhook channel",
                    "POST JSON payload to operator-configured URL with optional HMAC signing.",
                    [
                        "Payload includes alert id, service, status, title, body",
                        "Timeout default 10s; failure retried per engine policy",
                    ],
                    depends_on=["p1-channel-registry"],
                    estimated_size="M",
                    labels=["phase-1", "notifications", "webhook"],
                    spec_refs=["02-core-domain-and-features#notification-channels"],
                ),
                task(
                    "p1-channel-slack-dm",
                    "Slack per-user DM notification channel",
                    "Slack bot token stored encrypted; DM user on alert with basic message (interactive buttons Phase 3).",
                    [
                        "Slack token never returned in full after save (last-4 hint)",
                        "Missing slack_user_id skips with logged reason",
                    ],
                    depends_on=["p1-channel-registry", "p0-secrets-encryption-at-rest"],
                    estimated_size="M",
                    labels=["phase-1", "notifications", "slack"],
                    spec_refs=["02-core-domain-and-features#notification-channels"],
                    notes="Operator must create Slack app and provide bot token — per-install config, not blocking development with test token.",
                ),
                task(
                    "p1-channel-push-basic",
                    "Basic mobile push via Expo Push API (no Critical Alerts)",
                    "Register device tokens; send push with type alert.triggered using Time-Sensitive/default priority.",
                    [
                        "Push payload matches contract in doc 03 minus critical:true default",
                        "Invalid Expo token marks notification_attempt failed",
                    ],
                    depends_on=["p1-channel-registry"],
                    estimated_size="L",
                    labels=["phase-1", "notifications", "push"],
                    spec_refs=["03-mobile-app-spec#push-delivery", "02-core-domain-and-features#notification-channels"],
                ),
                task(
                    "p1-notification-rules",
                    "Per-user notification rules by priority",
                    "CRUD notification rules: channel order and delays per alert priority (low/high).",
                    [
                        "High-priority alert uses user's high rule ordering",
                        "Rules applied when resolving contact methods at send time",
                    ],
                    depends_on=["p1-channel-email", "p1-channel-push-basic"],
                    estimated_size="M",
                    labels=["phase-1", "notifications"],
                    spec_refs=["02-core-domain-and-features#notification-rules"],
                ),
            ],
        ),
        epic(
            "phase-1-inbound-plugins",
            "Inbound plugin system and day-1 plugins",
            "Compile-time InboundPlugin registry and secured webhook router.",
            [
                task(
                    "p1-inbound-registry",
                    "InboundPlugin registry in services/integrations",
                    "Registry with ParseAlert, ValidateConfig, ConfigSchema; AlertCreate includes EventType triggered|resolved.",
                    [
                        "Plugins registered at init via single registry.go import list",
                        "ParseAlert resolved event type documented in interface",
                    ],
                    depends_on=["p0-ci-compose-gate"],
                    estimated_size="M",
                    labels=["phase-1", "integrations", "plugins"],
                    spec_refs=["02-core-domain-and-features#inbound-integration-plugin-system"],
                ),
                task(
                    "p1-inbound-webhook-router",
                    "Secured webhook ingestion router",
                    "REST POST /webhook/{plugin}/{token} with rate limit 120 req/min per IntegrationKey, 256KB max body, JSON content-type enforcement.",
                    [
                        "Returns 429 with {error, code: RATE_LIMITED} when >120 req/min per IntegrationKey",
                        "Returns 413 when body exceeds 256KB",
                        "Invalid token returns 404; token never logged in full",
                    ],
                    depends_on=["p1-inbound-registry", "p0-auth-rbac-guards"],
                    estimated_size="L",
                    labels=["phase-1", "integrations", "security"],
                    spec_refs=["07-security-and-auth#inbound-webhook-ingestion", "02-core-domain-and-features#inbound-integration-plugin-system"],
                ),
                task(
                    "p1-inbound-alertmanager",
                    "prometheus-alertmanager plugin (reference implementation)",
                    "Parse Alertmanager webhook v4; map fingerprint to dedup_key; handle firing and resolved.",
                    [
                        "Firing creates triggered alert with dedup_key=fingerprint",
                        "Resolved auto-closes matching open alert by dedup_key",
                        "Fixture-based unit tests from upstream Alertmanager samples",
                    ],
                    depends_on=["p1-inbound-webhook-router"],
                    estimated_size="L",
                    labels=["phase-1", "integrations", "alertmanager"],
                    spec_refs=["02-core-domain-and-features#inbound-integration-plugin-system"],
                ),
                task(
                    "p1-inbound-generic-webhook",
                    "generic-webhook plugin with JSON field mapping",
                    "Configurable JSON path mapping to title, body, dedup_key, priority, event type.",
                    [
                        "Mapping config validated against JSON Schema",
                        "Missing required mapped fields returns 400 with VALIDATION body",
                    ],
                    depends_on=["p1-inbound-webhook-router"],
                    estimated_size="M",
                    labels=["phase-1", "integrations"],
                    spec_refs=["02-core-domain-and-features#inbound-integration-plugin-system"],
                ),
                task(
                    "p1-inbound-generic-rest",
                    "generic-rest-api authenticated alert create",
                    "Authenticated REST POST /api/v1/alerts with IntegrationKey bearer token creating alerts.",
                    [
                        "Missing Authorization returns 401 {error, code: UNAUTHENTICATED}",
                        "Valid payload creates alert and returns 201 with alert id",
                    ],
                    depends_on=["p1-inbound-webhook-router"],
                    estimated_size="M",
                    labels=["phase-1", "integrations", "api"],
                    spec_refs=["02-core-domain-and-features#inbound-integration-plugin-system"],
                ),
                task(
                    "p1-inbound-email-to-alert",
                    "email-to-alert inbound parsing",
                    "Unique inbound email per IntegrationKey; parse subject/body to alert fields via configurable rules.",
                    [
                        "Inbound email endpoint rejects unsigned/unauthenticated mail per SMTP relay config",
                        "Parsed email creates alert with source=email",
                    ],
                    depends_on=["p1-inbound-registry"],
                    estimated_size="L",
                    labels=["phase-1", "integrations", "email"],
                    spec_refs=["02-core-domain-and-features#inbound-integration-plugin-system"],
                ),
                task(
                    "p1-inbound-grafana",
                    "Grafana inbound plugin",
                    "Parse Grafana unified alerting webhook payload; map labels to dedup_key and severity.",
                    [
                        "Grafana firing and resolved states map to EventType correctly",
                        "Integration test uses captured Grafana fixture JSON",
                    ],
                    depends_on=["p1-inbound-alertmanager"],
                    estimated_size="M",
                    labels=["phase-1", "integrations", "grafana"],
                    spec_refs=["02-core-domain-and-features#inbound-integration-plugin-system"],
                ),
                task(
                    "p1-inbound-datadog",
                    "Datadog inbound plugin",
                    "Parse Datadog monitor alert webhook; stable dedup_key from monitor id + tags.",
                    [
                        "Datadog RECOVERY event resolves matching alert",
                        "Invalid signature when configured returns 401",
                    ],
                    depends_on=["p1-inbound-alertmanager"],
                    estimated_size="M",
                    labels=["phase-1", "integrations", "datadog"],
                    spec_refs=["02-core-domain-and-features#inbound-integration-plugin-system"],
                ),
                task(
                    "p1-inbound-uptime-kuma",
                    "uptime-kuma dedicated plugin",
                    "Parse nested Uptime Kuma payload; dedup_key=monitor.id; heartbeat.status drives auto-resolve.",
                    [
                        "monitor.id used as dedup_key scoped to service",
                        "Up event resolves prior down alert with same dedup_key",
                    ],
                    depends_on=["p1-inbound-generic-webhook"],
                    estimated_size="M",
                    labels=["phase-1", "integrations", "uptime-kuma"],
                    spec_refs=["02-core-domain-and-features#inbound-integration-plugin-system", "06-roadmap-mvp-phasing#phase-1"],
                ),
            ],
        ),
        epic(
            "phase-1-beszel-preset",
            "Beszel integration preset",
            "UI preset atop generic-webhook for Shoutrrr generic:// JSON payloads.",
            [
                task(
                    "p1-beszel-preset-ui",
                    "Beszel preset in integration picker + setup docs",
                    "Add Beszel option pre-filling generic-webhook mapping (title→title, message→body) and copy-paste Shoutrrr URL snippet in UI/docs.",
                    [
                        "Selecting Beszel creates IntegrationKey with correct plugin and mapping",
                        "Docs page shows exact generic:// URL format for Beszel notification settings",
                        "No new backend plugin package added",
                    ],
                    depends_on=["p1-inbound-generic-webhook"],
                    estimated_size="S",
                    labels=["phase-1", "integrations", "beszel", "docs"],
                    spec_refs=["02-core-domain-and-features#integration-presets", "06-roadmap-mvp-phasing#phase-1"],
                ),
            ],
        ),
        epic(
            "phase-1-dedup-basic",
            "Basic dedup-key matching",
            "Collapse repeat alerts on same service+dedup_key; enable auto-resolve from integrations.",
            [
                task(
                    "p1-dedup-key-collapse",
                    "Dedup-key collapse into open alert",
                    "New triggered alert with same service_id+dedup_key as open alert increments counter instead of new row.",
                    [
                        "Second firing increments alert.event_count",
                        "Collapsed alert does not re-notify already-acked targets in Phase 1 basic mode",
                    ],
                    depends_on=["p1-inbound-alertmanager"],
                    estimated_size="M",
                    labels=["phase-1", "dedup"],
                    spec_refs=["02-core-domain-and-features#alert-priority-noise-reduction", "06-roadmap-mvp-phasing#phase-1"],
                ),
                task(
                    "p1-dedup-auto-resolve",
                    "Auto-resolve path for resolved integration events",
                    "Resolved EventType closes matching open alert by dedup_key; idempotent if already closed.",
                    [
                        "Resolve for unknown dedup_key returns 200 no-op logged at debug",
                        "Closed alert records resolved_at and source integration name",
                    ],
                    depends_on=["p1-dedup-key-collapse"],
                    estimated_size="M",
                    labels=["phase-1", "dedup"],
                    spec_refs=["02-core-domain-and-features#inbound-integration-plugin-system"],
                ),
            ],
        ),
        epic(
            "phase-1-realtime",
            "Realtime GraphQL subscriptions",
            "LISTEN/NOTIFY to graphql-ws for live alert and on-call updates.",
            [
                task(
                    "p1-listen-notify-bridge",
                    "Postgres LISTEN/NOTIFY bridge in API",
                    "Emit NOTIFY on alert and schedule changes; API subscribes and fans out to GraphQL subscription handlers.",
                    [
                        "Alert status change triggers NOTIFY within same transaction commit",
                        "Reconnect logic on listener disconnect",
                    ],
                    depends_on=["p1-escalation-ack-close"],
                    estimated_size="M",
                    labels=["phase-1", "realtime", "graphql"],
                    spec_refs=["01-architecture-and-monorepo#realtime-updates"],
                ),
                task(
                    "p1-graphql-subscriptions",
                    "graphql-ws subscriptions for alerts and onCallNow",
                    "Web client subscribes to alertUpdated and onCallUpdated; UI updates without manual refresh.",
                    [
                        "urql subscription receives event within 2s of DB change in integration test",
                        "Unauthenticated subscription rejected",
                    ],
                    depends_on=["p1-listen-notify-bridge", "p0-web-app-shell"],
                    estimated_size="M",
                    labels=["phase-1", "realtime", "web"],
                    spec_refs=["05-ui-design-system#design-direction", "08-implementation-decisions-and-conventions#pinned-tool-choices"],
                ),
            ],
        ),
        epic(
            "phase-1-web-screens",
            "Web app core screens",
            "Alert list/detail, service config, schedule calendar, escalation editor per Phase 1.",
            [
                task(
                    "p1-ui-integration-keys",
                    "Integration key management UI",
                    "Admin UI to create/revoke/rotate integration keys; show URL and prefix only.",
                    [
                        "New key shown once on create; thereafter only prefix visible",
                        "Revoked keys return 404 on webhook within 60s",
                    ],
                    depends_on=["p1-inbound-webhook-router"],
                    estimated_size="M",
                    labels=["phase-1", "web", "integrations"],
                    spec_refs=["07-security-and-auth#inbound-webhook-ingestion"],
                ),
                task(
                    "p1-ui-service-config",
                    "Service configuration screens",
                    "CRUD services, assign escalation policy, link schedules, configure integration keys.",
                    [
                        "Service list searchable by name",
                        "Deleting service soft-deletes with audit event",
                    ],
                    depends_on=["p1-escalation-policy-crud", "p1-ui-integration-keys"],
                    estimated_size="M",
                    labels=["phase-1", "web"],
                    spec_refs=["06-roadmap-mvp-phasing#phase-1"],
                ),
                task(
                    "p1-ui-alert-list-detail",
                    "Alert list and detail views with ack/close",
                    "Realtime alert list with filters; detail shows timeline, escalation state, ack/close actions.",
                    [
                        "Ack and close buttons call mutations and update via subscription",
                        "Alert list shows event_count for deduped alerts",
                    ],
                    depends_on=["p1-escalation-ack-close", "p1-graphql-subscriptions"],
                    estimated_size="L",
                    labels=["phase-1", "web", "alerts"],
                    spec_refs=["06-roadmap-mvp-phasing#phase-1", "05-ui-design-system"],
                ),
                task(
                    "p1-ui-escalation-editor",
                    "Escalation policy editor component",
                    "Visual editor for ordered steps, delays, targets (user/rotation/webhook).",
                    [
                        "Drag reorder updates step order persisted to API",
                        "Invalid step without targets blocked in UI and API",
                    ],
                    depends_on=["p1-escalation-policy-crud"],
                    estimated_size="L",
                    labels=["phase-1", "web", "ui"],
                    spec_refs=["05-ui-design-system#component-library-structure"],
                ),
                task(
                    "p1-ui-schedule-calendar",
                    "Schedule calendar and override UI",
                    "ScheduleCalendar domain component showing layers, on-call now, override creation.",
                    [
                        "Calendar displays viewer-local times with timezone label",
                        "Override creation requires admin or member with team access",
                    ],
                    depends_on=["p1-schedule-overrides", "p1-schedule-on-call-now"],
                    estimated_size="L",
                    labels=["phase-1", "web", "scheduling"],
                    spec_refs=["05-ui-design-system#component-library-structure", "06-roadmap-mvp-phasing#phase-1"],
                ),
                task(
                    "p1-e2e-alert-flow",
                    "E2E: create service → fire test alert → ack",
                    "Extend Playwright smoke to cover core alerting happy path per doc 08 testing note.",
                    [
                        "Test creates service via UI, triggers alert via generic-rest API, acks in UI",
                        "CI runs on main alongside compose gate",
                    ],
                    depends_on=["p1-ui-alert-list-detail", "p1-inbound-generic-rest"],
                    estimated_size="M",
                    labels=["phase-1", "e2e"],
                    spec_refs=["08-implementation-decisions-and-conventions#pinned-tool-choices"],
                ),
            ],
        ),
    ]


def phase_2_epics() -> list[dict]:
    return [
        epic(
            "phase-2-mobile",
            "Mobile app v1 (Expo)",
            "Minimal native app: push, ack, escalate per doc 03.",
            [
                task(
                    "p2-mobile-expo-scaffold",
                    "Expo app scaffold in apps/mobile",
                    "Create Expo managed app with Expo Router, urql client from packages/ts-types, shared tokens theme.",
                    [
                        "apps/mobile builds with eas.json stub for future builds",
                        "Deep link scheme escalite:// configured",
                    ],
                    depends_on=["p0-graphql-codegen-ts"],
                    estimated_size="M",
                    labels=["phase-2", "mobile", "expo"],
                    spec_refs=["03-mobile-app-spec#tech-choice"],
                ),
                task(
                    "p2-mobile-auth-deep-link",
                    "Mobile auth via web login + deep link token exchange",
                    "Web login completes; redirect to app with one-time code exchanged for device refresh token.",
                    [
                        "Refresh token stored in Keychain/Keystore only",
                        "Revoked refresh token returns UNAUTHENTICATED on refresh",
                    ],
                    depends_on=["p2-mobile-expo-scaffold", "p0-auth-login-sessions"],
                    estimated_size="M",
                    labels=["phase-2", "mobile", "auth"],
                    spec_refs=["03-mobile-app-spec#auth", "07-security-and-auth#authn-mvp"],
                ),
                task(
                    "p2-mobile-device-token-register",
                    "Device push token registration mutation",
                    "GraphQL mutation stores Expo push token per device; revocable from user settings web UI.",
                    [
                        "Duplicate token upserts same device row",
                        "User settings lists devices with revoke action",
                    ],
                    depends_on=["p2-mobile-auth-deep-link"],
                    estimated_size="M",
                    labels=["phase-2", "mobile", "push"],
                    spec_refs=["03-mobile-app-spec#push-delivery"],
                ),
                task(
                    "p2-mobile-push-receive",
                    "Receive and display alert push notifications",
                    "Handle alert.triggered payload; tap opens alert action screen; foreground banner shown.",
                    [
                        "Notification displays title/body from payload",
                        "Tap navigates to alert detail route",
                    ],
                    depends_on=["p2-mobile-device-token-register", "p1-channel-push-basic"],
                    estimated_size="M",
                    labels=["phase-2", "mobile", "push"],
                    spec_refs=["03-mobile-app-spec#notification-payload-contract"],
                ),
                task(
                    "p2-mobile-ack-escalate",
                    "Ack and escalate from notification actions and in-app",
                    "Actionable notification buttons ack/escalate; mutations via GraphQL.",
                    [
                        "Ack from lock screen succeeds without opening app (platform permitting)",
                        "Escalate prompts for target selection minimal list",
                    ],
                    depends_on=["p2-mobile-push-receive", "p1-escalation-manual-snooze"],
                    estimated_size="L",
                    labels=["phase-2", "mobile"],
                    spec_refs=["03-mobile-app-spec#scope"],
                ),
            ],
        ),
        epic(
            "phase-2-ios-critical-alerts",
            "iOS Critical Alerts entitlement",
            "Apple entitlement application and implementation; start early parallel to Phase 1 per doc 06.",
            [
                task(
                    "p2-ios-critical-alerts-entitlement",
                    "Submit Apple Critical Alerts entitlement request",
                    "Human submits com.apple.developer.usernotifications.critical-alerts entitlement request via Apple Developer account for the Escalite iOS app.",
                    [
                        "Entitlement request submitted in Apple Developer portal",
                        "App ID configured with Critical Alerts capability pending approval",
                        "Fallback documented: Time-Sensitive notifications ship regardless of approval status",
                    ],
                    depends_on=["p0-scaffold-monorepo"],
                    estimated_size="S",
                    labels=["phase-2", "mobile", "ios", "external"],
                    spec_refs=["03-mobile-app-spec#why-native", "06-roadmap-mvp-phasing#sequencing-notes"],
                    external_dependency={
                        "requires_human_action": True,
                        "reason": "Apple Developer account + submit Critical Alerts entitlement request; approval is discretionary and timing is external",
                    },
                    notes="Doc 06: start in parallel with Phase 1 (not blocked on Phase 2). Bundle ID documented in repo once chosen; mobile scaffold not required to submit entitlement request.",
                ),
                task(
                    "p2-ios-critical-alerts-impl",
                    "Implement Critical Alerts push when entitlement approved",
                    "expo-notifications interruptionLevel critical when org+device configured; Android full-screen intent equivalent.",
                    [
                        "When critical:true in payload and entitlement present, iOS delivers as Critical Alert",
                        "Without entitlement, falls back to Time-Sensitive per doc 03",
                    ],
                    depends_on=["p2-ios-critical-alerts-entitlement", "p2-mobile-push-receive"],
                    estimated_size="M",
                    labels=["phase-2", "mobile", "ios", "push"],
                    spec_refs=["03-mobile-app-spec#notification-payload-contract"],
                    external_dependency={
                        "blocked_by": "p2-ios-critical-alerts-entitlement",
                        "requires_human_action": True,
                        "reason": "Cannot enable Critical Alerts in production until Apple approves entitlement",
                    },
                ),
            ],
        ),
        epic(
            "phase-2-twilio-channels",
            "SMS and voice via Twilio",
            "Pluggable provider with Twilio adapter; operator-supplied credentials.",
            [
                task(
                    "p2-twilio-provider-interface",
                    "Pluggable SMS/voice provider interface",
                    "Provider interface in engine; Twilio as first implementation; config encrypted at rest.",
                    [
                        "Provider selection via ESCALITE_SMS_PROVIDER=twilio",
                        "Missing Twilio creds disable SMS/voice channels with startup warning",
                    ],
                    depends_on=["p1-notification-rules", "p0-secrets-encryption-at-rest"],
                    estimated_size="M",
                    labels=["phase-2", "notifications", "twilio"],
                    spec_refs=["02-core-domain-and-features#notification-channels"],
                ),
                task(
                    "p2-twilio-integration",
                    "Twilio SMS and voice channel implementation",
                    "Send SMS and voice call notifications via twilio-go; record delivery status on notification_attempt.",
                    [
                        "SMS send succeeds against Twilio test credentials in integration test",
                        "Voice call TwiML speaks alert title and service name",
                    ],
                    depends_on=["p2-twilio-provider-interface"],
                    estimated_size="L",
                    labels=["phase-2", "notifications", "twilio", "external"],
                    spec_refs=["02-core-domain-and-features#notification-channels"],
                    external_dependency={
                        "requires_human_action": True,
                        "reason": "Twilio account + credentials for SMS/voice testing and production send",
                    },
                ),
            ],
        ),
        epic(
            "phase-2-dedup-advanced",
            "Advanced deduplication and maintenance windows",
            "Time-window dedup refinements and per-service mute windows.",
            [
                task(
                    "p2-dedup-time-window",
                    "Time-window dedup collapse with counter bumps",
                    "Collapse identical service+dedup_key within configurable window; bump counter; suppress duplicate notifications.",
                    [
                        "Alerts outside window create new alert row",
                        "Window default 5 minutes configurable per service",
                    ],
                    depends_on=["p1-dedup-key-collapse"],
                    estimated_size="M",
                    labels=["phase-2", "dedup"],
                    spec_refs=["06-roadmap-mvp-phasing#phase-2", "02-core-domain-and-features#alert-priority-noise-reduction"],
                ),
                task(
                    "p2-dedup-no-renotify-acked",
                    "Suppress re-notify of already-acked targets",
                    "Within dedup window, do not re-notify targets who already acknowledged.",
                    [
                        "Integration test: second firing does not create notification_attempt for acked user",
                        "Unacked targets still receive repeat per notification rules",
                    ],
                    depends_on=["p2-dedup-time-window"],
                    estimated_size="M",
                    labels=["phase-2", "dedup"],
                    spec_refs=["06-roadmap-mvp-phasing#phase-2"],
                ),
                task(
                    "p2-maintenance-windows",
                    "Maintenance windows / per-service mute",
                    "CRUD maintenance windows suppressing notifications and optionally ingestion per service.",
                    [
                        "Alert during active maintenance does not enqueue notifications",
                        "Admin UI shows active maintenance banner on service",
                    ],
                    depends_on=["p1-ui-service-config"],
                    estimated_size="M",
                    labels=["phase-2", "maintenance"],
                    spec_refs=["02-core-domain-and-features#alert-priority-noise-reduction", "06-roadmap-mvp-phasing#phase-2"],
                ),
            ],
        ),
    ]


def phase_3_epics() -> list[dict]:
    return [
        epic(
            "phase-3-incident-aggregate",
            "Incident aggregate and workflow",
            "Promote alerts to incidents with timeline and roles per doc 02.",
            [
                task(
                    "p3-incident-schema",
                    "Incident, timeline_events, roles schema and API",
                    "Migrations and GraphQL for Incident aggregate linking alerts, timeline, configurable roles.",
                    [
                        "incidents.status enum: investigating, identified, monitoring, resolved",
                        "timeline_events append-only with actor and body",
                    ],
                    depends_on=["p2-dedup-no-renotify-acked"],
                    estimated_size="M",
                    labels=["phase-3", "incidents"],
                    spec_refs=["02-core-domain-and-features#domain-model"],
                ),
                task(
                    "p3-incident-promote-manual",
                    "Manual promote alerts to incident",
                    "One-click promote from alert detail; multiple alerts can attach to one incident.",
                    [
                        "Promote moves alerts under incident_id foreign key",
                        "Promote writes timeline_event type=declared",
                    ],
                    depends_on=["p3-incident-schema"],
                    estimated_size="M",
                    labels=["phase-3", "incidents"],
                    spec_refs=["02-core-domain-and-features#key-departure-from-goalert"],
                ),
                task(
                    "p3-incident-promote-auto",
                    "Auto-promote rule (N alerts same service in window)",
                    "Configurable rule e.g. 3+ alerts in 5 minutes auto-declares incident.",
                    [
                        "Rule default disabled; enabling creates incidents without duplicate if one open",
                        "Auto-promote suppresses per-alert escalation per severity config",
                    ],
                    depends_on=["p3-incident-promote-manual"],
                    estimated_size="M",
                    labels=["phase-3", "incidents"],
                    spec_refs=["02-core-domain-and-features#key-departure-from-goalert"],
                ),
                task(
                    "p3-incident-timeline-ui",
                    "Incident timeline UI with notes and state changes",
                    "Web UI for incident detail, timeline feed, role assignment, status updates.",
                    [
                        "Adding note creates timeline_event visible to subscribers in <2s",
                        "Role assignment shows IC and Comms Lead labels",
                    ],
                    depends_on=["p3-incident-promote-manual", "p1-graphql-subscriptions"],
                    estimated_size="L",
                    labels=["phase-3", "incidents", "web"],
                    spec_refs=["02-core-domain-and-features#domain-model", "05-ui-design-system"],
                ),
                task(
                    "p3-incident-suppress-escalation",
                    "Suppress per-alert escalation when incident active",
                    "Grouped alerts under active incident skip further escalation per config.",
                    [
                        "New alert auto-attached to open incident skips step notifications when configured",
                        "Closing incident resumes normal escalation only for still-triggered alerts if configured",
                    ],
                    depends_on=["p3-incident-promote-auto", "p1-escalation-step-timers"],
                    estimated_size="M",
                    labels=["phase-3", "incidents", "escalation"],
                    spec_refs=["02-core-domain-and-features#key-departure-from-goalert"],
                ),
            ],
        ),
        epic(
            "phase-3-slack-incidents",
            "Slack channel-per-incident integration",
            "Rich Slack integration beyond Phase 1 DMs.",
            [
                task(
                    "p3-slack-app-install",
                    "Slack app OAuth install flow for workspace",
                    "OAuth flow stores workspace bot token encrypted; admin settings UI.",
                    [
                        "OAuth callback stores token with key-id encryption",
                        "Reinstall updates token without duplicate workspace rows",
                    ],
                    depends_on=["p3-incident-schema", "p0-secrets-encryption-at-rest"],
                    estimated_size="M",
                    labels=["phase-3", "slack", "integrations"],
                    spec_refs=["02-core-domain-and-features#outbound-integrations"],
                ),
                task(
                    "p3-slack-channel-per-incident",
                    "Create Slack channel per declared incident",
                    "On incident declare, create #incident-{id} channel and invite configured user group.",
                    [
                        "Channel naming template configurable with default incident-{short_id}",
                        "Failure to create channel logs error but does not roll back incident",
                    ],
                    depends_on=["p3-slack-app-install", "p3-incident-promote-manual"],
                    estimated_size="L",
                    labels=["phase-3", "slack", "incidents"],
                    spec_refs=["06-roadmap-mvp-phasing#phase-3"],
                ),
                task(
                    "p3-slack-interactive-actions",
                    "Slack interactive ack/escalate buttons",
                    "Slack message buttons call Escalite API with signed request verification.",
                    [
                        "Ack button acknowledges alert linked to incident",
                        "Invalid Slack signature returns 401",
                    ],
                    depends_on=["p3-slack-channel-per-incident", "p1-escalation-ack-close"],
                    estimated_size="M",
                    labels=["phase-3", "slack"],
                    spec_refs=["02-core-domain-and-features#outbound-integrations"],
                ),
                task(
                    "p3-slack-timeline-mirror",
                    "Mirror incident timeline to Slack thread",
                    "Timeline events post as threaded replies in incident channel.",
                    [
                        "Note added in web appears in Slack thread within 5s",
                        "Resolve status posts summary message to channel",
                    ],
                    depends_on=["p3-slack-channel-per-incident", "p3-incident-timeline-ui"],
                    estimated_size="M",
                    labels=["phase-3", "slack", "incidents"],
                    spec_refs=["06-roadmap-mvp-phasing#phase-3"],
                ),
            ],
        ),
        epic(
            "phase-3-postmortem",
            "Postmortem Markdown export",
            "Generate postmortem doc from incident timeline.",
            [
                task(
                    "p3-postmortem-export",
                    "Markdown postmortem export from timeline",
                    "Export button generates Markdown with timeline, roles, linked alerts, duration metrics.",
                    [
                        "Exported MD includes ISO8601 timestamps in UTC",
                        "Download returns text/markdown attachment",
                    ],
                    depends_on=["p3-incident-timeline-ui"],
                    estimated_size="M",
                    labels=["phase-3", "postmortem"],
                    spec_refs=["02-core-domain-and-features#reporting-postmortems", "06-roadmap-mvp-phasing#phase-3"],
                ),
            ],
        ),
    ]


def phase_4_epics() -> list[dict]:
    return [
        epic(
            "phase-4-migration-importers",
            "Migration importers",
            "GoAlert and PagerDuty export importers per doc 06 Phase 4.",
            [
                task(
                    "p4-goalert-importer-cli",
                    "GoAlert Postgres importer CLI",
                    "CLI tool mapping GoAlert public schema to Escalite: users, schedules, services, policies, alerts (historical optional).",
                    [
                        "Dry-run mode prints mapping summary without writes",
                        "Import creates id mapping file for audit",
                        "Documented in docs/migration/goalert.md",
                    ],
                    depends_on=["p3-postmortem-export"],
                    estimated_size="L",
                    labels=["phase-4", "migration", "goalert"],
                    spec_refs=["01-architecture-and-monorepo#build-vs-fork", "06-roadmap-mvp-phasing#phase-4"],
                ),
                task(
                    "p4-pagerduty-importer",
                    "PagerDuty export API importer",
                    "Import users, schedules, services, escalation policies from PagerDuty export/API.",
                    [
                        "API token stored only via env at import time, not persisted",
                        "Import report lists skipped unsupported objects",
                    ],
                    depends_on=["p4-goalert-importer-cli"],
                    estimated_size="L",
                    labels=["phase-4", "migration", "pagerduty"],
                    spec_refs=["00-vision-and-scope#success-criteria", "06-roadmap-mvp-phasing#phase-4"],
                    external_dependency={
                        "requires_human_action": True,
                        "reason": "PagerDuty API token or export file from operator account for test/import runs",
                    },
                ),
            ],
        ),
        epic(
            "phase-4-deployment-hardening",
            "Hardened Docker Compose + Coolify validation",
            "Production compose path and PaaS validation; explicitly no Helm.",
            [
                task(
                    "p4-compose-hardening",
                    "Hardened docker-compose.yml and production docs",
                    "Single-node production compose: resource limits, healthchecks, backup notes, upgrade procedure.",
                    [
                        "README deployment section starts with docker compose up",
                        "Services define healthcheck and restart policy",
                        "SBOM generated on release per doc 07",
                    ],
                    depends_on=["p4-goalert-importer-cli"],
                    estimated_size="M",
                    labels=["phase-4", "compose", "docs"],
                    spec_refs=["01-architecture-and-monorepo#deployment-targets", "06-roadmap-mvp-phasing#phase-4"],
                ),
                task(
                    "p4-coolify-validation",
                    "Validate deployment on Coolify",
                    "Document and test deploy on Coolify: env vars, TLS via platform, single-org instance.",
                    [
                        "docs/deploy/coolify.md lists required env vars",
                        "Manual test checklist completed and recorded in doc",
                    ],
                    depends_on=["p4-compose-hardening"],
                    estimated_size="M",
                    labels=["phase-4", "compose", "coolify", "docs"],
                    spec_refs=["06-roadmap-mvp-phasing#phase-4", "04-licensing-and-editions#tenant-isolation"],
                    external_dependency={
                        "requires_human_action": True,
                        "reason": "Coolify instance or VPS for manual validation deploy; DNS/TLS domain decisions by operator",
                    },
                ),
            ],
        ),
    ]


def phase_5_epics() -> list[dict]:
    return [
        epic(
            "phase-5-sso-scim",
            "SSO / SAML / SCIM",
            "Full enterprise auth beyond Phase 0 generic OIDC.",
            [
                task(
                    "p5-saml-sso",
                    "SAML SSO login and config UI",
                    "SAML IdP metadata upload, SP-initiated login, JIT user provisioning without SCIM.",
                    [
                        "SAML response validated against IdP cert",
                        "Disabled SAML falls back to password/OIDC only",
                    ],
                    depends_on=["p4-compose-hardening"],
                    estimated_size="L",
                    labels=["phase-5", "auth", "saml"],
                    spec_refs=["06-roadmap-mvp-phasing#phase-5", "07-security-and-auth#explicitly-out-of-mvp"],
                ),
                task(
                    "p5-scim-provisioning",
                    "SCIM user and group provisioning",
                    "SCIM 2.0 endpoints for user lifecycle and group→team mapping.",
                    [
                        "SCIM bearer token rotatable from admin UI",
                        "Deprovisioned user cannot authenticate within 60s",
                    ],
                    depends_on=["p5-saml-sso"],
                    estimated_size="L",
                    labels=["phase-5", "auth", "scim"],
                    spec_refs=["06-roadmap-mvp-phasing#phase-5"],
                ),
            ],
        ),
        epic(
            "phase-5-status-pages",
            "Status pages app",
            "Public status page renderer in apps/status-page.",
            [
                task(
                    "p5-status-page-schema",
                    "Status page components and incident visibility model",
                    "Schema for components, subscriptions, linking public incidents; CSP exceptions for status page.",
                    [
                        "Public page does not expose internal alert payloads",
                        "Component status enum: operational, degraded, partial_outage, major_outage",
                    ],
                    depends_on=["p4-compose-hardening"],
                    estimated_size="M",
                    labels=["phase-5", "status-page"],
                    spec_refs=["00-vision-and-scope#product-pillars", "04-licensing-and-editions"],
                ),
                task(
                    "p5-status-page-app",
                    "apps/status-page public renderer",
                    "Deployable status page app subscribed to public incident updates; custom domain docs.",
                    [
                        "Status page renders component grid and active incidents",
                        "frame-ancestors CSP distinct from main app per doc 07",
                    ],
                    depends_on=["p5-status-page-schema", "p3-incident-timeline-ui"],
                    estimated_size="L",
                    labels=["phase-5", "status-page", "web"],
                    spec_refs=["01-architecture-and-monorepo#repo-layout", "06-roadmap-mvp-phasing#phase-5"],
                    external_dependency={
                        "requires_human_action": True,
                        "reason": "Custom domain DNS and TLS certificate provisioning for public status page (operator responsibility)",
                    },
                ),
            ],
        ),
        epic(
            "phase-5-analytics",
            "MTTA/MTTR analytics dashboard",
            "Tremor-based analytics in web app.",
            [
                task(
                    "p5-analytics-metrics",
                    "MTTA/MTTR metric computation and API",
                    "Batch or incremental computation of mean time to ack/resolve per team/service over windows.",
                    [
                        "Metrics API returns 7d and 30d rollups",
                        "Computation excludes maintenance-window alerts when configured",
                    ],
                    depends_on=["p3-incident-timeline-ui"],
                    estimated_size="L",
                    labels=["phase-5", "analytics"],
                    spec_refs=["06-roadmap-mvp-phasing#phase-5", "05-ui-design-system#dashboard-data-viz"],
                ),
                task(
                    "p5-analytics-dashboard-ui",
                    "Analytics dashboard UI with Tremor charts",
                    "Dashboard in web app showing MTTA/MTTR trends and alert volume.",
                    [
                        "Charts use packages/ui/charts wrappers",
                        "Dashboard loads under 3s with 10k alert seed dataset",
                    ],
                    depends_on=["p5-analytics-metrics"],
                    estimated_size="M",
                    labels=["phase-5", "analytics", "web"],
                    spec_refs=["05-ui-design-system#dashboard-data-viz"],
                ),
            ],
        ),
        epic(
            "phase-5-multi-org",
            "Multi-org support (MSP use case)",
            "Multiple organizations in one self-hosted deployment.",
            [
                task(
                    "p5-multi-org-model",
                    "Multi-org data model and org switcher API",
                    "Users may belong to multiple orgs; org_id scoping on all queries; org switcher mutation.",
                    [
                        "Member in org A cannot query org B resources (FORBIDDEN)",
                        "Admin role is per-organization not global",
                    ],
                    depends_on=["p5-saml-sso"],
                    estimated_size="L",
                    labels=["phase-5", "multi-org"],
                    spec_refs=["06-roadmap-mvp-phasing#phase-5", "00-vision-and-scope#target-users"],
                ),
                task(
                    "p5-multi-org-ui",
                    "Org switcher UI and per-org settings",
                    "Header org switcher; settings pages scoped to active org.",
                    [
                        "Switcher lists only orgs user belongs to",
                        "Switching org refetches urql cache",
                    ],
                    depends_on=["p5-multi-org-model"],
                    estimated_size="M",
                    labels=["phase-5", "multi-org", "web"],
                    spec_refs=["06-roadmap-mvp-phasing#phase-5"],
                ),
            ],
        ),
        epic(
            "phase-5-audit-enterprise",
            "Audit log UI and ticketing integrations",
            "Admin audit UI and optional Jira/ServiceNow outbound.",
            [
                task(
                    "p5-audit-log-ui",
                    "Admin audit log list and export",
                    "Simple paginated audit log UI for admins; CSV export.",
                    [
                        "Audit list filterable by action type and date range",
                        "Export produces CSV with RFC3339 timestamps",
                    ],
                    depends_on=["p0-audit-events-auth"],
                    estimated_size="M",
                    labels=["phase-5", "audit"],
                    spec_refs=["07-security-and-auth#audit-log"],
                ),
                task(
                    "p5-jira-servicenow-outbound",
                    "Jira/ServiceNow ticket on incident declare (optional)",
                    "Outbound plugin creates ticket when incident declared if integration configured.",
                    [
                        "Ticket URL stored on incident record",
                        "Missing integration config skips silently with info log",
                    ],
                    depends_on=["p3-incident-promote-manual"],
                    estimated_size="L",
                    labels=["phase-5", "integrations"],
                    spec_refs=["02-core-domain-and-features#outbound-integrations"],
                    notes="Build cost permitting per doc 02; may defer if schedule pressure.",
                ),
            ],
        ),
    ]


ROADMAP["phases"].extend(
    [
        {
            "id": "phase-1",
            "name": "Core alerting (GoAlert parity)",
            "source_doc": "06-roadmap-mvp-phasing",
            "epics": phase_1_epics(),
        },
        {
            "id": "phase-2",
            "name": "Mobile + reliable delivery",
            "source_doc": "06-roadmap-mvp-phasing",
            "epics": phase_2_epics(),
        },
        {
            "id": "phase-3",
            "name": "Incident response layer",
            "source_doc": "06-roadmap-mvp-phasing",
            "epics": phase_3_epics(),
        },
        {
            "id": "phase-4",
            "name": "Migration & adoption tooling",
            "source_doc": "06-roadmap-mvp-phasing",
            "epics": phase_4_epics(),
        },
        {
            "id": "phase-5",
            "name": "Remaining CE features",
            "source_doc": "06-roadmap-mvp-phasing",
            "epics": phase_5_epics(),
        },
    ]
)

def collect_task_ids() -> dict[str, dict]:
    ids: dict[str, dict] = {}
    for phase in ROADMAP["phases"]:
        for epic in phase["epics"]:
            for t in epic["tasks"]:
                if t["id"] in ids:
                    raise ValueError(f"Duplicate task id: {t['id']}")
                ids[t["id"]] = t
    return ids


def validate_external_dependencies(task_ids: dict[str, dict]) -> None:
    for tid, t in task_ids.items():
        ext = t.get("external_dependency")
        if not ext:
            continue
        if ext.get("requires_human_action") is not True:
            raise ValueError(f"Task {tid}: external_dependency.requires_human_action must be true")
        if not ext.get("reason"):
            raise ValueError(f"Task {tid}: external_dependency.reason is required")
        blocked_by = ext.get("blocked_by")
        if blocked_by is not None:
            if blocked_by not in task_ids:
                raise ValueError(f"Task {tid}: external_dependency.blocked_by references unknown task {blocked_by}")
            if blocked_by not in t.get("depends_on", []):
                raise ValueError(
                    f"Task {tid}: external_dependency.blocked_by ({blocked_by}) must also appear in depends_on"
                )
        extra_keys = set(ext) - {"requires_human_action", "reason", "blocked_by"}
        if extra_keys:
            raise ValueError(f"Task {tid}: unexpected external_dependency keys: {extra_keys}")


def generate_external_dependencies(task_ids: dict[str, dict]) -> list[dict]:
    """Build root external_dependencies from per-task external_dependency fields."""
    by_reason: dict[str, list[str]] = {}
    for tid, t in sorted(task_ids.items()):
        ext = t.get("external_dependency")
        if not ext:
            continue
        by_reason.setdefault(ext["reason"], []).append(tid)

    return [
        {"blocks": blocks, "requires_human_action": reason}
        for reason, blocks in sorted(by_reason.items(), key=lambda item: item[1])
    ]


def validate_dag(task_ids: dict[str, dict]) -> None:
    for tid, t in task_ids.items():
        for dep in t["depends_on"]:
            if dep not in task_ids:
                raise ValueError(f"Task {tid} depends on unknown task {dep}")


def render_markdown() -> str:
    task_ids = collect_task_ids()
    lines = [
        "# Escalite MVP Roadmap (Phases 0–5)",
        "",
        "> **Source of truth:** [`roadmap.yaml`](./roadmap.yaml). This file is generated — do not edit by hand. Regenerate with `python docs/roadmap/generate_roadmap.py`.",
        "",
        "## Summary",
        "",
    ]

    phase_count = len(ROADMAP["phases"])
    epic_count = sum(len(p["epics"]) for p in ROADMAP["phases"])
    task_count = len(task_ids)
    lines.extend(
        [
            f"- **Phases:** {phase_count} (0–5)",
            f"- **Epics:** {epic_count}",
            f"- **Tasks:** {task_count}",
            "",
            "## External dependencies (human action required)",
            "",
        ]
    )
    for ext in ROADMAP["external_dependencies"]:
        blocks = ", ".join(f"`{b}`" for b in ext["blocks"])
        lines.append(f"- **Blocks:** {blocks}")
        lines.append(f"  - **Action:** {ext['requires_human_action']}")
        lines.append("")

    lines.extend(["## Phases", ""])

    for phase in ROADMAP["phases"]:
        lines.append(f"### {phase['id']}: {phase['name']}")
        lines.append("")
        for epic in phase["epics"]:
            lines.append(f"#### Epic: {epic['title']} (`{epic['id']}`)")
            lines.append("")
            lines.append(textwrap.fill(epic["description"], width=100))
            lines.append("")
            for t in epic["tasks"]:
                deps = ", ".join(f"`{d}`" for d in t["depends_on"]) if t["depends_on"] else "_none_"
                ext = ""
                if t.get("external_dependency"):
                    ext = " ⚠️ **EXTERNAL DEPENDENCY**"
                lines.append(f"- **`{t['id']}`** — {t['title']}{ext}")
                lines.append(f"  - Size: {t['estimated_size']} | Depends on: {deps}")
                if t.get("notes"):
                    lines.append(f"  - Note: {t['notes']}")
                lines.append("  - Acceptance criteria:")
                for ac in t["acceptance_criteria"]:
                    lines.append(f"    - {ac}")
                lines.append("")
        lines.append("---")
        lines.append("")

    return "\n".join(lines)


def main() -> None:
    task_ids = collect_task_ids()
    validate_dag(task_ids)
    validate_external_dependencies(task_ids)
    ROADMAP["external_dependencies"] = generate_external_dependencies(task_ids)

    YAML_PATH.write_text(
        yaml.dump(ROADMAP, sort_keys=False, allow_unicode=True, default_flow_style=False, width=120),
        encoding="utf-8",
    )
    MD_PATH.write_text(render_markdown(), encoding="utf-8")
    print(f"Wrote {YAML_PATH} ({len(task_ids)} tasks)")
    print(f"Wrote {MD_PATH}")


if __name__ == "__main__":
    main()
