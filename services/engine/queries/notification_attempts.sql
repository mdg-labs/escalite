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

-- name: GetNotificationAttemptByID :one
SELECT *
FROM notification_attempts
WHERE id = $1
  AND organization_id = $2
LIMIT 1;

-- name: FinishNotificationAttempt :one
UPDATE notification_attempts
SET
    status = $3,
    error_message = $4,
    sent_at = CASE WHEN $3 = 'sent' THEN now() ELSE sent_at END
WHERE id = $1
  AND organization_id = $2
RETURNING *;
