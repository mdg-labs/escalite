-- name: ListActiveOverridesByScheduleAt :many
SELECT *
FROM overrides
WHERE schedule_id = $1
  AND organization_id = $2
  AND deleted_at IS NULL
  AND starts_at <= $3
  AND ends_at > $3
ORDER BY rotation_id, created_at DESC;
