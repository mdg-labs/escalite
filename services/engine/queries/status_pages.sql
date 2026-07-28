-- name: GetStatusPageByID :one
SELECT *
FROM status_pages
WHERE id = $1
  AND organization_id = $2
LIMIT 1;

-- name: GetStatusPageIncidentByID :one
SELECT *
FROM status_page_incidents
WHERE id = $1
  AND organization_id = $2
LIMIT 1;

-- name: GetStatusPageIncidentUpdateByID :one
SELECT *
FROM status_page_incident_updates
WHERE id = $1
  AND organization_id = $2
LIMIT 1;

-- name: ListActiveStatusPageSubscriptions :many
SELECT *
FROM status_page_subscriptions
WHERE status_page_id = $1
  AND organization_id = $2
  AND unsubscribed_at IS NULL
ORDER BY created_at ASC;
