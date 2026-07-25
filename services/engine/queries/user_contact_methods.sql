-- name: GetUserContactMethodByChannel :one
SELECT *
FROM user_contact_methods
WHERE organization_id = $1
  AND user_id = $2
  AND channel = $3
LIMIT 1;
