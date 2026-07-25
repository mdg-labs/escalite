-- name: ListRotationsByScheduleID :many
SELECT *
FROM rotations
WHERE schedule_id = $1
  AND organization_id = $2
ORDER BY layer;

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
