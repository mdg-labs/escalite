-- name: GetUserByID :one
SELECT *
FROM users
WHERE id = $1
  AND organization_id = $2
LIMIT 1;

-- name: GetFirstAdminUserByOrganization :one
SELECT *
FROM users
WHERE organization_id = $1
  AND role = 'admin'
ORDER BY created_at ASC
LIMIT 1;
