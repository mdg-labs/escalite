-- name: CreateOverride :one
INSERT INTO overrides (
    id,
    schedule_id,
    rotation_id,
    organization_id,
    user_id,
    replaced_user_id,
    starts_at,
    ends_at,
    created_by_user_id
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9
)
RETURNING *;

-- name: GetOverrideByID :one
SELECT *
FROM overrides
WHERE id = $1
  AND organization_id = $2
  AND deleted_at IS NULL
LIMIT 1;

-- name: ListActiveOverridesByScheduleID :many
SELECT *
FROM overrides
WHERE schedule_id = $1
  AND organization_id = $2
  AND deleted_at IS NULL
ORDER BY starts_at, created_at;

-- name: ListActiveOverridesByScheduleAt :many
SELECT *
FROM overrides
WHERE schedule_id = $1
  AND organization_id = $2
  AND deleted_at IS NULL
  AND starts_at <= $3
  AND ends_at > $3
ORDER BY rotation_id, created_at DESC;

-- name: SoftDeleteOverride :one
UPDATE overrides
SET deleted_at = now(),
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
  AND deleted_at IS NULL
RETURNING *;
