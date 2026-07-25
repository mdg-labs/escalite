-- name: CreateMobileAuthCode :one
INSERT INTO mobile_auth_codes (
    id,
    user_id,
    organization_id,
    code_hash,
    expires_at
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
RETURNING *;

-- name: GetMobileAuthCodeByHash :one
SELECT *
FROM mobile_auth_codes
WHERE code_hash = $1
  AND used_at IS NULL
  AND expires_at > now()
LIMIT 1;

-- name: MarkMobileAuthCodeUsed :exec
UPDATE mobile_auth_codes
SET used_at = now(),
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
  AND used_at IS NULL;
