-- name: CreateSession :one
INSERT INTO sessions (
    id,
    user_id,
    organization_id,
    expires_at,
    user_agent
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
RETURNING *;

-- name: GetSessionByID :one
SELECT *
FROM sessions
WHERE id = $1
  AND organization_id = $2
  AND revoked_at IS NULL
  AND expires_at > now()
LIMIT 1;

-- name: GetActiveSessionByID :one
SELECT *
FROM sessions
WHERE id = $1
  AND revoked_at IS NULL
  AND expires_at > now()
LIMIT 1;

-- name: RevokeSession :exec
UPDATE sessions
SET revoked_at = now(),
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
  AND revoked_at IS NULL;

-- name: RevokeAllUserSessions :exec
UPDATE sessions
SET revoked_at = now(),
    updated_at = now()
WHERE user_id = $1
  AND organization_id = $2
  AND revoked_at IS NULL;

-- name: UpdateSessionOrganization :one
UPDATE sessions
SET user_id = $2,
    organization_id = $3,
    updated_at = now()
WHERE id = $1
  AND revoked_at IS NULL
  AND expires_at > now()
RETURNING *;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE id = $1
  AND organization_id = $2;
