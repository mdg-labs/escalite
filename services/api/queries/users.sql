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
SELECT u.*
FROM users u
JOIN accounts a ON a.id = u.account_id
WHERE a.email = $1
ORDER BY u.created_at ASC
LIMIT 1;

-- name: GetUserByAccountAndOrganization :one
SELECT *
FROM users
WHERE account_id = $1
  AND organization_id = $2
  AND deprovisioned_at IS NULL
LIMIT 1;

-- name: ListOrganizationMembershipsByAccountID :many
SELECT
    sqlc.embed(u),
    sqlc.embed(o)
FROM users u
JOIN organizations o ON o.id = u.organization_id
WHERE u.account_id = $1
  AND u.deprovisioned_at IS NULL
ORDER BY o.name ASC;

-- name: GetLoginMembershipByAccountID :one
SELECT u.*
FROM users u
WHERE u.account_id = $1
  AND u.deprovisioned_at IS NULL
ORDER BY (
    SELECT max(s.created_at)
    FROM sessions s
    WHERE s.user_id = u.id
      AND s.organization_id = u.organization_id
      AND s.revoked_at IS NULL
) DESC NULLS LAST,
u.created_at ASC
LIMIT 1;

-- name: GetFirstAdminUserByOrganization :one
SELECT *
FROM users
WHERE organization_id = $1
  AND role = 'admin'
ORDER BY created_at ASC
LIMIT 1;

-- name: CountOrgAdmins :one
SELECT count(*)::bigint AS count
FROM users
WHERE organization_id = $1
  AND role = 'admin'
  AND deprovisioned_at IS NULL;

-- name: ListOrganizationUsersForOrgAdmin :many
SELECT *
FROM users
WHERE organization_id = $1
  AND deprovisioned_at IS NULL
ORDER BY email ASC
LIMIT $2;

-- name: ListOrganizationUsersForTeamMember :many
SELECT DISTINCT u.*
FROM users u
INNER JOIN team_memberships tm_viewer
  ON tm_viewer.user_id = $2
 AND tm_viewer.organization_id = u.organization_id
INNER JOIN team_memberships tm_target
  ON tm_target.team_id = tm_viewer.team_id
 AND tm_target.user_id = u.id
 AND tm_target.organization_id = u.organization_id
WHERE u.organization_id = $1
  AND u.deprovisioned_at IS NULL
ORDER BY u.email ASC
LIMIT $3;

-- name: CreateUser :one
INSERT INTO users (
    id,
    account_id,
    organization_id,
    email,
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
    account_id,
    organization_id,
    email,
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
