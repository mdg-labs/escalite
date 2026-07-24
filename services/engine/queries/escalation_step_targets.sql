-- name: ListEscalationStepTargetsByStepID :many
SELECT *
FROM escalation_step_targets
WHERE escalation_step_id = $1
  AND organization_id = $2
ORDER BY created_at;
