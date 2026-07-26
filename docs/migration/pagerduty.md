# PagerDuty → Escalite migration

CLI tool that reads PagerDuty configuration via the REST API or an offline export file and imports core on-call configuration into Escalite.

## Prerequisites

- PagerDuty API token with read access to users, schedules, services, and escalation policies — **or** a bundled export JSON file.
- Escalite Postgres with migrations applied (`task migrate` or API startup).
- An existing Escalite organization (from `/api/v1/setup` or bootstrap).
- A target team in that organization, or use `--create-team` to create one during import.

## Build

```bash
cd tools/pagerduty-importer
go build -o pagerduty-importer .
```

## Environment variables

| Variable | Purpose |
| -------- | ------- |
| `PAGERDUTY_API_TOKEN` | PagerDuty REST API token (read at import time only; **never persisted**) |
| `PAGERDUTY_API_BASE` | Optional API base URL (default: `https://api.pagerduty.com`) |
| `ESCALITE_DATABASE_URL` | Target Escalite Postgres URL (overridden by `--target-url`) |

The API token is used only in memory for HTTP requests. It is not written to the mapping file, Escalite database, or any other persisted output.

## Usage

### Dry-run (no writes)

Prints a mapping summary: source object counts, target tables, sample entity names, and unsupported objects that will be skipped.

```bash
export PAGERDUTY_API_TOKEN="your-token"

pagerduty-importer \
  --target-url "$ESCALITE_DATABASE_URL" \
  --organization-id "<escalite-org-uuid>" \
  --team-id "<escalite-team-uuid>" \
  --dry-run
```

### Import via API

```bash
export PAGERDUTY_API_TOKEN="your-token"

pagerduty-importer \
  --target-url "$ESCALITE_DATABASE_URL" \
  --organization-id "<escalite-org-uuid>" \
  --create-team \
  --team-name "Production On-Call" \
  --mapping-file ./pagerduty-id-mapping.json
```

### Import from export file (offline)

Use when you have a JSON export snapshot instead of live API access:

```bash
pagerduty-importer \
  --export-file ./pagerduty-export.json \
  --target-url "$ESCALITE_DATABASE_URL" \
  --organization-id "<escalite-org-uuid>" \
  --team-id "<escalite-team-uuid>"
```

Export file format:

```json
{
  "users": [],
  "schedules": [],
  "services": [],
  "escalation_policies": [],
  "unsupported": []
}
```

## Entity mapping

| PagerDuty (source) | Escalite (target) | Notes |
| ------------------ | ----------------- | ----- |
| `users` | `users`, `team_memberships` | Passwords are not copied; imported users need SSO or password reset |
| `schedules` | `schedules` | Timezone preserved |
| `schedule_layers` | `rotations` | RRULE derived from `rotation_turn_length_seconds` |
| `services` | `services` | One Escalite service per PagerDuty service |
| `escalation_policies` + rules | `escalation_policies`, `escalation_steps`, `escalation_step_targets` | One policy per service (Escalite model) |

PagerDuty escalation targets of type `user_reference` or `schedule_reference` map to Escalite step targets. Nested policy references, team targets, and other unsupported types are listed in the import report under **Skipped unsupported objects**.

## Import report

After import, stdout includes counts per entity type and a **Skipped unsupported objects** section listing every object that could not be imported (with reason). The mapping file also records skipped entries under `skipped`.

## ID mapping file

After import, `pagerduty-id-mapping.json` (or `--mapping-file`) contains:

- `imported_at`, `organization_id`, `team_id`
- Maps from PagerDuty source IDs to Escalite UUIDs per entity type
- `skipped` entries for rows that could not be imported

Retain this file for audit and troubleshooting cross-references.

## Flags

| Flag | Default | Description |
| ---- | ------- | ----------- |
| `--api-token` | `$PAGERDUTY_API_TOKEN` | PagerDuty API token (not persisted) |
| `--export-file` | | Offline export JSON (alternative to API) |
| `--target-url` | `$ESCALITE_DATABASE_URL` | Escalite Postgres URL |
| `--organization-id` | (required) | Target organization UUID |
| `--team-id` | | Target team UUID |
| `--create-team` | `false` | Create a new team instead of using `--team-id` |
| `--team-name` | `PagerDuty Import` | Name for `--create-team` |
| `--dry-run` | `false` | Summary only; no writes |
| `--mapping-file` | `pagerduty-id-mapping.json` | Audit mapping output path |

## Limitations

- Single organization / team per run.
- Integrations, event rules, maintenance windows, incident workflows, and on-call overrides are not imported.
- Nested escalation policy references and team escalation targets are skipped and reported.
- Shared PagerDuty escalation policies are duplicated per service (Escalite one-policy-per-service model).
- Imported users cannot sign in with a PagerDuty password.

## Tests

```bash
cd tools/pagerduty-importer && go test ./...
```
