-- name: GetTeamByID :one
SELECT *
FROM teams
WHERE id = $1
  AND organization_id = $2
LIMIT 1;

-- name: ListTeamsByOrganizationID :many
SELECT *
FROM teams
WHERE organization_id = $1
ORDER BY name ASC;

-- name: CreateTeam :one
INSERT INTO teams (
    id,
    organization_id,
    name
) VALUES (
    $1,
    $2,
    $3
)
RETURNING *;

-- name: GetTeamByName :one
SELECT *
FROM teams
WHERE organization_id = $1
  AND name = $2
LIMIT 1;

-- name: UpdateTeam :one
UPDATE teams
SET name = $3,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
RETURNING *;

-- name: DeleteTeam :one
DELETE FROM teams
WHERE id = $1
  AND organization_id = $2
RETURNING *;

-- name: CountServicesByTeamID :one
SELECT COUNT(*)::bigint AS count
FROM services
WHERE team_id = $1
  AND organization_id = $2
  AND deleted_at IS NULL;
