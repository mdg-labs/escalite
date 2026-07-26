-- name: CountAccounts :one
SELECT count(*)::bigint AS count
FROM accounts;

-- name: GetAccountByID :one
SELECT *
FROM accounts
WHERE id = $1
LIMIT 1;

-- name: GetAccountByEmail :one
SELECT *
FROM accounts
WHERE email = $1
LIMIT 1;

-- name: CreateAccount :one
INSERT INTO accounts (
    id,
    email,
    password_hash
) VALUES (
    $1,
    $2,
    $3
)
RETURNING *;

-- name: UpdateAccountPasswordHash :exec
UPDATE accounts
SET password_hash = $2,
    updated_at = now()
WHERE id = $1;
