-- name: GetOrganizationScimSettings :one
SELECT *
FROM organization_scim_settings
WHERE organization_id = $1;

-- name: GetOrganizationScimSettingsByTokenHash :one
SELECT *
FROM organization_scim_settings
WHERE token_hash = $1
LIMIT 1;

-- name: UpsertOrganizationScimSettings :one
INSERT INTO organization_scim_settings (
    organization_id,
    token_hash,
    token_prefix
) VALUES (
    $1, $2, $3
)
ON CONFLICT (organization_id) DO UPDATE SET
    token_hash = EXCLUDED.token_hash,
    token_prefix = EXCLUDED.token_prefix,
    updated_at = now()
RETURNING *;
