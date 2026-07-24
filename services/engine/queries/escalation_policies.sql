-- name: ListEscalationPoliciesByServiceID :many
SELECT *
FROM escalation_policies
WHERE service_id = $1
  AND organization_id = $2
ORDER BY created_at;
