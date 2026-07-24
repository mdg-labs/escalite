-- name: CreateEscalationStep :one
INSERT INTO escalation_steps (
    id,
    escalation_policy_id,
    organization_id,
    step_order,
    delay_minutes,
    repeat_last_step,
    max_repeats
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

-- name: ListEscalationStepsByPolicyID :many
SELECT *
FROM escalation_steps
WHERE escalation_policy_id = $1
  AND organization_id = $2
ORDER BY step_order;

-- name: GetEscalationStepByPolicyAndOrder :one
SELECT *
FROM escalation_steps
WHERE escalation_policy_id = $1
  AND organization_id = $2
  AND step_order = $3
LIMIT 1;

-- name: DeleteEscalationStepsByPolicyID :exec
DELETE FROM escalation_steps
WHERE escalation_policy_id = $1
  AND organization_id = $2;
