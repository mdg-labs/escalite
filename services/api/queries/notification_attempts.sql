-- name: CreateNotificationAttempt :one
INSERT INTO notification_attempts (
    id,
    organization_id,
    alert_id,
    escalation_step_id,
    channel,
    status,
    recipient
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

-- name: ListNotificationAttemptsByAlertID :many
SELECT *
FROM notification_attempts
WHERE alert_id = $1
  AND organization_id = $2
ORDER BY created_at;

-- name: CountNotificationAttemptsByAlertID :one
SELECT COUNT(*)::bigint
FROM notification_attempts
WHERE alert_id = $1
  AND organization_id = $2;
