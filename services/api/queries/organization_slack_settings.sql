-- name: GetOrganizationSlackSettings :one
SELECT *
FROM organization_slack_settings
WHERE organization_id = $1
LIMIT 1;

-- name: UpsertOrganizationSlackSettings :one
INSERT INTO organization_slack_settings (
    organization_id,
    bot_token_ciphertext,
    encryption_key_id,
    token_hint
) VALUES (
    $1,
    $2,
    $3,
    $4
)
ON CONFLICT (organization_id) DO UPDATE
SET bot_token_ciphertext = EXCLUDED.bot_token_ciphertext,
    encryption_key_id = EXCLUDED.encryption_key_id,
    token_hint = EXCLUDED.token_hint,
    updated_at = now()
RETURNING *;
