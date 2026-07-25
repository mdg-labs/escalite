-- name: GetUserContactMethodByChannel :one
SELECT *
FROM user_contact_methods
WHERE organization_id = $1
  AND user_id = $2
  AND channel = $3
LIMIT 1;

-- name: UpsertUserContactMethod :one
INSERT INTO user_contact_methods (
    id,
    organization_id,
    user_id,
    channel,
    config
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
ON CONFLICT (organization_id, user_id, channel) DO UPDATE
SET config = EXCLUDED.config,
    updated_at = now()
RETURNING *;
