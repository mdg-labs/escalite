-- name: CreateAuditEvent :one
INSERT INTO audit_events (
    id,
    organization_id,
    actor_id,
    action,
    target_type,
    target_id,
    metadata
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7
)
RETURNING *;

-- name: ListAuditEventsByOrganization :many
SELECT *
FROM audit_events
WHERE organization_id = $1
ORDER BY created_at ASC;
