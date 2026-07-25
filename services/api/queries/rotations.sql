-- name: CreateRotation :one
INSERT INTO rotations (
    id,
    schedule_id,
    organization_id,
    name,
    layer,
    rrule,
    participants
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7
)
RETURNING *;

-- name: GetRotationByID :one
SELECT *
FROM rotations
WHERE id = $1
  AND organization_id = $2
LIMIT 1;

-- name: ListRotationsByScheduleID :many
SELECT *
FROM rotations
WHERE schedule_id = $1
  AND organization_id = $2
ORDER BY layer;

-- name: UpdateRotation :one
UPDATE rotations
SET name = $3,
    layer = $4,
    rrule = $5,
    participants = $6,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
RETURNING *;

-- name: DeleteRotation :exec
DELETE FROM rotations
WHERE id = $1
  AND organization_id = $2;
