-- name: MarkHeartbeatMonitorsOverdue :execrows
UPDATE heartbeat_monitors
SET status = 'overdue',
    updated_at = now()
WHERE status = 'healthy'
  AND now() > COALESCE(last_ping_at, created_at) + (interval_seconds * interval '1 second');

-- name: ListHeartbeatMonitorsReadyToTrigger :many
SELECT *
FROM heartbeat_monitors
WHERE status IN ('healthy', 'overdue')
  AND now() > COALESCE(last_ping_at, created_at)
      + ((interval_seconds + grace_seconds) * interval '1 second')
ORDER BY created_at;

-- name: MarkHeartbeatMonitorTriggered :one
UPDATE heartbeat_monitors
SET status = 'triggered',
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
  AND status IN ('healthy', 'overdue')
RETURNING *;

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
    status,
    last_ping_at
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10
)
RETURNING *;

-- name: GetAlertByServiceDedupKey :one
SELECT *
FROM alerts
WHERE service_id = $1
  AND organization_id = $2
  AND dedup_key = $3
  AND status IN ('triggered', 'acknowledged')
ORDER BY created_at DESC
LIMIT 1;
