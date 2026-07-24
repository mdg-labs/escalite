-- name: CreateEscalationPolicy :one
INSERT INTO escalation_policies (
    id,
    organization_id,
    service_id,
    name
) VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING *;

-- name: GetEscalationPolicyByID :one
SELECT *
FROM escalation_policies
WHERE id = $1
  AND organization_id = $2
LIMIT 1;

-- name: ListEscalationPoliciesByServiceID :many
SELECT *
FROM escalation_policies
WHERE service_id = $1
  AND organization_id = $2
ORDER BY created_at;

-- name: UpdateEscalationPolicy :one
UPDATE escalation_policies
SET name = $3,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
RETURNING *;

-- name: DeleteEscalationPolicy :exec
DELETE FROM escalation_policies
WHERE id = $1
  AND organization_id = $2;
