-- name: GetUserByID :one
SELECT *
FROM users
WHERE id = $1
  AND organization_id = $2
LIMIT 1;
