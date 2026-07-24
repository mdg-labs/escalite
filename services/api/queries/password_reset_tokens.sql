-- name: CreatePasswordResetToken :one
INSERT INTO password_reset_tokens (
    id,
    user_id,
    organization_id,
    token_hash,
    expires_at
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
RETURNING *;

-- name: GetPasswordResetTokenByHash :one
SELECT *
FROM password_reset_tokens
WHERE token_hash = $1
LIMIT 1;

-- name: MarkPasswordResetTokenUsed :exec
UPDATE password_reset_tokens
SET used_at = now(),
    updated_at = now()
WHERE id = $1;

-- name: InvalidateUnusedPasswordResetTokensForUser :exec
UPDATE password_reset_tokens
SET used_at = now(),
    updated_at = now()
WHERE user_id = $1
  AND used_at IS NULL;
