-- name: GetActiveIntegrationKeyByTokenHash :one
SELECT *
FROM integration_keys
WHERE token = $1
  AND revoked_at IS NULL
LIMIT 1;
