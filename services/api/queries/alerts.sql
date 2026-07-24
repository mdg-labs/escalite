-- name: CreateTriggeredAlert :one
INSERT INTO alerts (
    id,
    organization_id,
    service_id,
    integration_key_id,
    status,
    dedup_key,
    summary,
    description,
    priority,
    escalation_state
) VALUES (
    $1,
    $2,
    $3,
    $4,
    'triggered',
    $5,
    $6,
    $7,
    $8,
    $9
)
RETURNING *;

-- name: GetAlertByID :one
SELECT *
FROM alerts
WHERE id = $1
  AND organization_id = $2
LIMIT 1;

-- name: UpdateAlertEscalationState :one
UPDATE alerts
SET escalation_state = $3,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
RETURNING *;

-- name: AcknowledgeAlert :one
UPDATE alerts
SET status = 'acknowledged',
    acknowledged_at = now(),
    acknowledged_by_user_id = $4,
    escalation_state = $3,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
  AND status = 'triggered'
RETURNING *;

-- name: CloseAlert :one
UPDATE alerts
SET status = 'closed',
    closed_at = now(),
    escalation_state = $3,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
  AND status IN ('triggered', 'acknowledged')
RETURNING *;
