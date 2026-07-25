-- name: GetAlertByID :one
SELECT *
FROM alerts
WHERE id = $1
  AND organization_id = $2
LIMIT 1;

-- name: AssignAlertToIncident :one
UPDATE alerts
SET incident_id = $3,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
  AND incident_id IS NULL
  AND status IN ('triggered', 'acknowledged')
RETURNING *;

-- name: CountRecentOpenAlertsByService :one
SELECT count(*)::integer AS count
FROM alerts
WHERE service_id = sqlc.arg(service_id)
  AND organization_id = sqlc.arg(organization_id)
  AND status IN ('triggered', 'acknowledged')
  AND created_at >= now() - (sqlc.arg(window_seconds)::integer * interval '1 second');

-- name: ListRecentUnassignedAlertsByService :many
SELECT *
FROM alerts
WHERE service_id = sqlc.arg(service_id)
  AND organization_id = sqlc.arg(organization_id)
  AND status IN ('triggered', 'acknowledged')
  AND incident_id IS NULL
  AND created_at >= now() - (sqlc.arg(window_seconds)::integer * interval '1 second')
ORDER BY created_at ASC;

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

-- name: ListAlertsByIncidentID :many
SELECT *
FROM alerts
WHERE incident_id = $1
  AND organization_id = $2
ORDER BY created_at ASC;

-- name: ReEscalateAlert :one
UPDATE alerts
SET status = 'triggered',
    acknowledged_at = NULL,
    acknowledged_by_user_id = NULL,
    escalation_state = $3,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
  AND status IN ('triggered', 'acknowledged')
RETURNING *;
