-- name: GetOrganizationSlackSettings :one
SELECT *
FROM organization_slack_settings
WHERE organization_id = $1
LIMIT 1;

-- name: GetOrganizationSlackSettingsByWorkspaceID :one
SELECT *
FROM organization_slack_settings
WHERE workspace_id = sqlc.arg(workspace_id)
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

-- name: UpsertOrganizationSlackOAuthInstall :one
-- Reinstalling the Escalite Slack app (same organization) upserts this single row by
-- organization_id, so re-authorizing never creates a duplicate workspace row.
INSERT INTO organization_slack_settings (
    organization_id,
    bot_token_ciphertext,
    encryption_key_id,
    token_hint,
    workspace_id,
    workspace_name,
    bot_user_id,
    scope
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8
)
ON CONFLICT (organization_id) DO UPDATE
SET bot_token_ciphertext = EXCLUDED.bot_token_ciphertext,
    encryption_key_id = EXCLUDED.encryption_key_id,
    token_hint = EXCLUDED.token_hint,
    workspace_id = EXCLUDED.workspace_id,
    workspace_name = EXCLUDED.workspace_name,
    bot_user_id = EXCLUDED.bot_user_id,
    scope = EXCLUDED.scope,
    updated_at = now()
RETURNING *;
