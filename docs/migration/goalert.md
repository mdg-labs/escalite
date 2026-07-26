# GoAlert → Escalite migration

CLI tool that reads a GoAlert Postgres database (public schema) and imports core on-call configuration into Escalite.

## Prerequisites

- Running GoAlert Postgres instance with the standard public schema (`users`, `schedules`, `rotations`, `services`, `escalation_policies`, `alerts`, …).
- Escalite Postgres with migrations applied (`task migrate` or API startup).
- An existing Escalite organization (from `/api/v1/setup` or bootstrap).
- A target team in that organization, or use `--create-team` to create one during import.

## Build

```bash
cd tools/goalert-importer
go build -o goalert-importer .
```

## Environment variables

| Variable | Purpose |
| -------- | ------- |
| `GOALERT_DATABASE_URL` | Source GoAlert Postgres URL (overridden by `--source-url`) |
| `ESCALITE_DATABASE_URL` | Target Escalite Postgres URL (overridden by `--target-url`) |

## Usage

### Dry-run (no writes)

Prints a mapping summary: source row counts, target tables, and sample entity names.

```bash
goalert-importer \
  --source-url "$GOALERT_DATABASE_URL" \
  --target-url "$ESCALITE_DATABASE_URL" \
  --organization-id "<escalite-org-uuid>" \
  --team-id "<escalite-team-uuid>" \
  --dry-run
```

### Import

Creates Escalite rows and writes an audit ID mapping file (default: `goalert-id-mapping.json`).

```bash
goalert-importer \
  --source-url "$GOALERT_DATABASE_URL" \
  --target-url "$ESCALITE_DATABASE_URL" \
  --organization-id "<escalite-org-uuid>" \
  --create-team \
  --team-name "Production On-Call" \
  --mapping-file ./goalert-id-mapping.json
```

### Historical alerts (optional)

```bash
goalert-importer ... --include-alerts
```

## Entity mapping

| GoAlert (source) | Escalite (target) | Notes |
| ---------------- | ----------------- | ----- |
| `users` | `users`, `team_memberships` | Passwords are not copied; imported users need SSO or password reset |
| `schedules` | `schedules` | Timezone preserved |
| `rotations` + `rotation_participants` | `rotations` | RRULE derived from GoAlert rotation type |
| `services` | `services` | One Escalite service per GoAlert service |
| `escalation_policies` + steps + actions | `escalation_policies`, `escalation_steps`, `escalation_step_targets` | One policy per service (Escalite model) |
| `alerts` | `alerts` | Optional; status `active` → `triggered`, `closed` → `closed` |

GoAlert escalation actions targeting users or schedules map to Escalite step targets. Rotation-target actions resolve via the parent schedule. Webhook channel destinations map to webhook targets. Unsupported actions are recorded in the mapping file under `skipped`.

## ID mapping file

After import, `goalert-id-mapping.json` (or `--mapping-file`) contains:

- `imported_at`, `organization_id`, `team_id`
- Maps from GoAlert source IDs (strings) to Escalite UUIDs per entity type
- `skipped` entries for rows that could not be imported

Retain this file for audit and troubleshooting cross-references.

## Flags

| Flag | Default | Description |
| ---- | ------- | ----------- |
| `--source-url` | `$GOALERT_DATABASE_URL` | GoAlert Postgres URL |
| `--target-url` | `$ESCALITE_DATABASE_URL` | Escalite Postgres URL |
| `--organization-id` | (required) | Target organization UUID |
| `--team-id` | | Target team UUID |
| `--create-team` | `false` | Create a new team instead of using `--team-id` |
| `--team-name` | `GoAlert Import` | Name for `--create-team` |
| `--dry-run` | `false` | Summary only; no writes |
| `--include-alerts` | `false` | Import historical alerts |
| `--mapping-file` | `goalert-id-mapping.json` | Audit mapping output path |

## Limitations

- Single organization / team per run; multi-team GoAlert deployments may need multiple imports or manual team assignment.
- Notification channels, integration keys, and heartbeat monitors are not imported in this version.
- Imported users cannot sign in with their GoAlert password.

## Tests

```bash
cd tools/goalert-importer && go test ./...
```
