-- name: GetOrganizationTicketingSettings :one
SELECT *
FROM organization_ticketing_settings
WHERE organization_id = $1
LIMIT 1;
