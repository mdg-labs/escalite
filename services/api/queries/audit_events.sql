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

-- name: ListAuditEventsFiltered :many
SELECT *
FROM audit_events
WHERE organization_id = sqlc.arg('organization_id')
  AND (
    sqlc.narg('action_filter')::text IS NULL
    OR action = sqlc.narg('action_filter')
  )
  AND (
    sqlc.narg('from_time')::timestamptz IS NULL
    OR created_at >= sqlc.narg('from_time')
  )
  AND (
    sqlc.narg('to_time')::timestamptz IS NULL
    OR created_at <= sqlc.narg('to_time')
  )
ORDER BY created_at DESC
LIMIT sqlc.arg('page_limit') OFFSET sqlc.arg('page_offset');

-- name: CountAuditEventsFiltered :one
SELECT COUNT(*)::int AS total_count
FROM audit_events
WHERE organization_id = sqlc.arg('organization_id')
  AND (
    sqlc.narg('action_filter')::text IS NULL
    OR action = sqlc.narg('action_filter')
  )
  AND (
    sqlc.narg('from_time')::timestamptz IS NULL
    OR created_at >= sqlc.narg('from_time')
  )
  AND (
    sqlc.narg('to_time')::timestamptz IS NULL
    OR created_at <= sqlc.narg('to_time')
  );

-- name: ListAuditEventActionsByOrganization :many
SELECT DISTINCT action
FROM audit_events
WHERE organization_id = $1
ORDER BY action ASC;
