-- name: GetOrganizationSlackSettings :one
SELECT *
FROM organization_slack_settings
WHERE organization_id = $1
LIMIT 1;
