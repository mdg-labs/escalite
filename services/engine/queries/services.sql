-- name: GetServiceByID :one
SELECT *
FROM services
WHERE id = $1
  AND organization_id = $2
LIMIT 1;
