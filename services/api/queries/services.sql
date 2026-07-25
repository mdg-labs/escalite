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
SET name = $3,
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
