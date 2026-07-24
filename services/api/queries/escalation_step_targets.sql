-- name: CreateEscalationStepTarget :one
INSERT INTO escalation_step_targets (
    id,
    escalation_step_id,
    organization_id,
    target_type,
    user_id,
    schedule_id,
    webhook_url,
    channels
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8
)
RETURNING *;

-- name: ListEscalationStepTargetsByStepID :many
SELECT *
FROM escalation_step_targets
WHERE escalation_step_id = $1
  AND organization_id = $2
ORDER BY created_at;
