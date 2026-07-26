-- name: GetOrganizationTicketingSettings :one
SELECT *
FROM organization_ticketing_settings
WHERE organization_id = $1
LIMIT 1;

-- name: UpsertOrganizationTicketingSettings :one
INSERT INTO organization_ticketing_settings (
    organization_id,
    plugin_name,
    config,
    api_token_ciphertext,
    encryption_key_id,
    token_hint
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
ON CONFLICT (organization_id) DO UPDATE
SET plugin_name = EXCLUDED.plugin_name,
    config = EXCLUDED.config,
    api_token_ciphertext = EXCLUDED.api_token_ciphertext,
    encryption_key_id = EXCLUDED.encryption_key_id,
    token_hint = EXCLUDED.token_hint,
    updated_at = now()
RETURNING *;
