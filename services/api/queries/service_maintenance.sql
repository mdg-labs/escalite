-- name: CreateMaintenanceWindow :one
INSERT INTO maintenance_windows (
    id,
    organization_id,
    service_id,
    description,
    starts_at,
    ends_at,
    suppress_notifications,
    suppress_ingestion
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8
)
RETURNING *;

-- name: GetMaintenanceWindowByID :one
SELECT *
FROM maintenance_windows
WHERE id = $1
  AND organization_id = $2
LIMIT 1;

-- name: ListMaintenanceWindowsByServiceID :many
SELECT *
FROM maintenance_windows
WHERE service_id = $1
  AND organization_id = $2
ORDER BY starts_at DESC;

-- name: ListActiveMaintenanceWindowsByServiceID :many
SELECT *
FROM maintenance_windows
WHERE service_id = $1
  AND organization_id = $2
  AND starts_at <= $3
  AND ends_at > $3
ORDER BY starts_at;

-- name: ServiceHasActiveNotificationSuppression :one
SELECT EXISTS (
    SELECT 1
    FROM maintenance_windows
    WHERE service_id = $1
      AND organization_id = $2
      AND suppress_notifications = true
      AND starts_at <= $3
      AND ends_at > $3
) AS active;

-- name: ServiceHasActiveIngestionSuppression :one
SELECT EXISTS (
    SELECT 1
    FROM maintenance_windows
    WHERE service_id = $1
      AND organization_id = $2
      AND suppress_ingestion = true
      AND starts_at <= $3
      AND ends_at > $3
) AS active;

-- name: UpdateMaintenanceWindow :one
UPDATE maintenance_windows
SET description = $3,
    starts_at = $4,
    ends_at = $5,
    suppress_notifications = $6,
    suppress_ingestion = $7,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
RETURNING *;

-- name: DeleteMaintenanceWindow :exec
DELETE FROM maintenance_windows
WHERE id = $1
  AND organization_id = $2;
