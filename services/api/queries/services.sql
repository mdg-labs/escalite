-- name: GetServiceByID :one
SELECT *
FROM services
WHERE id = $1
  AND organization_id = $2
LIMIT 1;

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
