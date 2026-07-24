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
