-- name: GetScheduleByID :one
SELECT *
FROM schedules
WHERE id = $1
  AND organization_id = $2
LIMIT 1;

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
