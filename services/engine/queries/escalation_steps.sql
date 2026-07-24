-- name: GetEscalationStepByPolicyAndOrder :one
SELECT *
FROM escalation_steps
WHERE escalation_policy_id = $1
  AND organization_id = $2
  AND step_order = $3
LIMIT 1;
