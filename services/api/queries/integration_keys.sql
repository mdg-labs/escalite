-- name: GetActiveIntegrationKeyByTokenHash :one
SELECT *
FROM integration_keys
WHERE token = $1
  AND revoked_at IS NULL
LIMIT 1;

-- name: GetIntegrationKeyByID :one
SELECT *
FROM integration_keys
WHERE id = $1
  AND organization_id = $2
LIMIT 1;

-- name: ListIntegrationKeysByServiceID :many
SELECT *
FROM integration_keys
WHERE service_id = $1
  AND organization_id = $2
ORDER BY created_at DESC;

-- name: RevokeIntegrationKey :one
UPDATE integration_keys
SET revoked_at = now(),
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
  AND revoked_at IS NULL
RETURNING *;

-- name: CreateIntegrationKey :one
INSERT INTO integration_keys (
    id,
    service_id,
    organization_id,
    token,
    prefix,
    plugin_name,
    config
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
