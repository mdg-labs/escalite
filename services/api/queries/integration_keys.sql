-- name: GetActiveIntegrationKeyByTokenHash :one
SELECT *
FROM integration_keys
WHERE token = $1
  AND revoked_at IS NULL
LIMIT 1;

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
