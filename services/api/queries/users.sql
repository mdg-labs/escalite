-- name: CountUsers :one
SELECT count(*)::bigint AS count
FROM users;

-- name: GetUserByID :one
SELECT *
FROM users
WHERE id = $1
  AND organization_id = $2
LIMIT 1;

-- name: GetUserByEmail :one
SELECT *
FROM users
WHERE organization_id = $1
  AND email = $2
LIMIT 1;

-- name: GetUserByEmailForAuth :one
SELECT *
FROM users
WHERE email = $1
LIMIT 1;

-- name: GetFirstAdminUserByOrganization :one
SELECT *
FROM users
WHERE organization_id = $1
  AND role = 'admin'
ORDER BY created_at ASC
LIMIT 1;

-- name: UpdateUserPasswordHash :exec
UPDATE users
SET password_hash = $2,
    updated_at = now()
WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (
    id,
    organization_id,
    email,
    password_hash,
    role
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
RETURNING *;

-- name: CreateScimUser :one
INSERT INTO users (
    id,
    organization_id,
    email,
    password_hash,
    role,
    scim_external_id
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING *;

-- name: GetUserByScimExternalID :one
SELECT *
FROM users
WHERE organization_id = $1
  AND scim_external_id = $2
LIMIT 1;

-- name: GetUserByIDIncludingDeprovisioned :one
SELECT *
FROM users
WHERE id = $1
  AND organization_id = $2
LIMIT 1;

-- name: ListScimUsersByOrganizationID :many
SELECT *
FROM users
WHERE organization_id = $1
  AND scim_external_id IS NOT NULL
ORDER BY email ASC;

-- name: UpdateScimUser :one
UPDATE users
SET email = $3,
    scim_external_id = $4,
    deprovisioned_at = $5,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
RETURNING *;

-- name: DeprovisionUser :one
UPDATE users
SET deprovisioned_at = now(),
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
  AND deprovisioned_at IS NULL
RETURNING *;

-- name: ReprovisionScimUser :one
UPDATE users
SET email = $3,
    scim_external_id = $4,
    deprovisioned_at = NULL,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
RETURNING *;
