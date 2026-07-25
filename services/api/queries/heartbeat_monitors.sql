-- name: CreateHeartbeatMonitor :one
INSERT INTO heartbeat_monitors (
    id,
    organization_id,
    service_id,
    name,
    interval_seconds,
    grace_seconds,
    token_hash,
    prefix,
    status
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9
)
RETURNING *;

-- name: GetHeartbeatMonitorByID :one
SELECT *
FROM heartbeat_monitors
WHERE id = $1
  AND organization_id = $2
LIMIT 1;

-- name: ListHeartbeatMonitorsByServiceID :many
SELECT *
FROM heartbeat_monitors
WHERE service_id = $1
  AND organization_id = $2
ORDER BY created_at;

-- name: UpdateHeartbeatMonitor :one
UPDATE heartbeat_monitors
SET name = $3,
    interval_seconds = $4,
    grace_seconds = $5,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
RETURNING *;

-- name: DeleteHeartbeatMonitor :exec
DELETE FROM heartbeat_monitors
WHERE id = $1
  AND organization_id = $2;
