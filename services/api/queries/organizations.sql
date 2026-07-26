-- name: CountOrganizations :one
SELECT count(*)::bigint AS count
FROM organizations;

-- name: GetOrganizationByID :one
SELECT *
FROM organizations
WHERE id = $1
LIMIT 1;

-- name: GetFirstOrganization :one
SELECT *
FROM organizations
ORDER BY created_at ASC
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
new_account AS (
    INSERT INTO accounts (id, email, password_hash)
    VALUES (sqlc.arg(account_id), sqlc.arg(email), sqlc.arg(password_hash))
    RETURNING *
),
new_user AS (
    INSERT INTO users (id, account_id, organization_id, email, role)
    VALUES (
        sqlc.arg(user_id),
        (SELECT id FROM new_account),
        (SELECT id FROM new_org),
        sqlc.arg(email),
        'admin'
    )
    RETURNING *
)
SELECT
    new_org.id AS organization_id,
    new_org.name AS organization_name,
    new_user.id AS user_id,
    new_user.email AS user_email,
    new_user.role AS user_role,
    new_account.id AS account_id
FROM new_org,
     new_user,
     new_account;
