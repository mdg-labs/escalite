-- name: UpsertMobileDevice :one
INSERT INTO mobile_devices (
    id,
    organization_id,
    user_id,
    refresh_token_id,
    expo_push_token,
    push_token_prefix,
    platform,
    device_label,
    last_registered_at
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    now()
)
ON CONFLICT (expo_push_token) DO UPDATE
SET user_id = EXCLUDED.user_id,
    organization_id = EXCLUDED.organization_id,
    refresh_token_id = EXCLUDED.refresh_token_id,
    push_token_prefix = EXCLUDED.push_token_prefix,
    platform = EXCLUDED.platform,
    device_label = EXCLUDED.device_label,
    revoked_at = NULL,
    last_registered_at = now(),
    updated_at = now()
RETURNING *;

-- name: ListMobileDevicesForUser :many
SELECT *
FROM mobile_devices
WHERE organization_id = $1
  AND user_id = $2
ORDER BY last_registered_at DESC;

-- name: RevokeMobileDevice :one
UPDATE mobile_devices
SET revoked_at = now(),
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
  AND user_id = $3
  AND revoked_at IS NULL
RETURNING *;
