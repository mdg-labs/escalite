-- name: CountOrganizations :one
SELECT count(*)::bigint AS count
FROM organizations;

-- name: GetOrganizationByID :one
SELECT *
FROM organizations
WHERE id = $1
LIMIT 1;

-- name: CreateOrganization :one
INSERT INTO organizations (
    id,
    name
) VALUES (
    $1,
    $2
)
RETURNING *;

-- name: BootstrapOrganizationWithAdmin :one
WITH new_org AS (
    INSERT INTO organizations (id, name)
    VALUES (sqlc.arg(org_id), sqlc.arg(org_name))
    RETURNING *
),
new_user AS (
    INSERT INTO users (id, organization_id, email, password_hash, role)
    VALUES (sqlc.arg(user_id), (SELECT id FROM new_org), sqlc.arg(email), sqlc.arg(password_hash), 'admin')
    RETURNING *
)
SELECT
    new_org.id AS organization_id,
    new_org.name AS organization_name,
    new_user.id AS user_id,
    new_user.email AS user_email,
    new_user.role AS user_role
FROM new_org,
     new_user;
