-- name: GetScimGroupByID :one
SELECT *
FROM scim_groups
WHERE id = $1
  AND organization_id = $2
LIMIT 1;

-- name: GetScimGroupByExternalID :one
SELECT *
FROM scim_groups
WHERE organization_id = $1
  AND external_id = $2
LIMIT 1;

-- name: ListScimGroupsByOrganizationID :many
SELECT *
FROM scim_groups
WHERE organization_id = $1
ORDER BY display_name ASC;

-- name: CreateScimGroup :one
INSERT INTO scim_groups (
    id,
    organization_id,
    external_id,
    display_name,
    team_id
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: UpdateScimGroupDisplayName :one
UPDATE scim_groups
SET display_name = $3,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
RETURNING *;

-- name: DeleteScimGroup :exec
DELETE FROM scim_groups
WHERE id = $1
  AND organization_id = $2;
