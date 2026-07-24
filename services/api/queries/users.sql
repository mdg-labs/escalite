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
