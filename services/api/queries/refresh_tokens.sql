-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (
    id,
    user_id,
    organization_id,
    token_hash,
    user_agent,
    expires_at
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING *;

-- name: GetRefreshTokenByHash :one
SELECT *
FROM refresh_tokens
WHERE token_hash = $1
  AND revoked_at IS NULL
  AND expires_at > now()
LIMIT 1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked_at = now(),
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
  AND revoked_at IS NULL;

-- name: RevokeAllUserRefreshTokens :exec
UPDATE refresh_tokens
SET revoked_at = now(),
    updated_at = now()
WHERE user_id = $1
  AND organization_id = $2
  AND revoked_at IS NULL;
