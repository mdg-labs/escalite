-- name: GetUserNotificationRuleByPriority :one
SELECT *
FROM user_notification_rules
WHERE organization_id = $1
  AND user_id = $2
  AND priority = $3
LIMIT 1;

-- name: UpsertUserNotificationRule :one
INSERT INTO user_notification_rules (
    id,
    organization_id,
    user_id,
    priority,
    steps
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
ON CONFLICT (organization_id, user_id, priority) DO UPDATE
SET steps = EXCLUDED.steps,
    updated_at = now()
RETURNING *;
