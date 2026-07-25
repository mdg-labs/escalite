-- name: CreateSchedule :one
INSERT INTO schedules (
    id,
    organization_id,
    team_id,
    name,
    timezone
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
RETURNING *;

-- name: GetScheduleByID :one
SELECT *
FROM schedules
WHERE id = $1
  AND organization_id = $2
LIMIT 1;

-- name: ListSchedulesByTeamID :many
SELECT *
FROM schedules
WHERE team_id = $1
  AND organization_id = $2
ORDER BY created_at;

-- name: UpdateSchedule :one
UPDATE schedules
SET name = $3,
    timezone = $4,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
RETURNING *;

-- name: DeleteSchedule :exec
DELETE FROM schedules
WHERE id = $1
  AND organization_id = $2;
