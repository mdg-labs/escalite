-- name: GetServiceByID :one
SELECT *
FROM services
WHERE id = $1
  AND organization_id = $2
  AND deleted_at IS NULL
LIMIT 1;

-- name: ListServicesByOrganizationID :many
SELECT *
FROM services
WHERE organization_id = $1
  AND deleted_at IS NULL
ORDER BY name ASC;

-- name: CreateService :one
INSERT INTO services (
    id,
    organization_id,
    team_id,
    name
) VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING *;

-- name: UpdateService :one
UPDATE services
SET name = COALESCE(sqlc.narg('name')::text, name),
    auto_promote_enabled = COALESCE(sqlc.narg('auto_promote_enabled')::boolean, auto_promote_enabled),
    auto_promote_alert_threshold = COALESCE(sqlc.narg('auto_promote_alert_threshold')::integer, auto_promote_alert_threshold),
    auto_promote_window_seconds = COALESCE(sqlc.narg('auto_promote_window_seconds')::integer, auto_promote_window_seconds),
    auto_promote_suppress_escalation_priorities = COALESCE(sqlc.narg('auto_promote_suppress_escalation_priorities')::text[], auto_promote_suppress_escalation_priorities),
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
  AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteService :one
UPDATE services
SET deleted_at = now(),
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
  AND deleted_at IS NULL
RETURNING *;
