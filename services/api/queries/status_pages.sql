-- name: GetStatusPageByOrganizationID :one
SELECT *
FROM status_pages
WHERE organization_id = $1
LIMIT 1;

-- name: GetStatusPageBySlug :one
SELECT *
FROM status_pages
WHERE slug = $1
  AND enabled = true
LIMIT 1;

-- name: CreateStatusPage :one
INSERT INTO status_pages (
    id,
    organization_id,
    slug,
    title,
    enabled,
    frame_ancestors_csp
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING *;

-- name: UpdateStatusPage :one
UPDATE status_pages
SET slug = $3,
    title = $4,
    enabled = $5,
    frame_ancestors_csp = $6,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
RETURNING *;

-- name: ListStatusPageComponents :many
SELECT *
FROM status_page_components
WHERE status_page_id = $1
  AND organization_id = $2
ORDER BY position ASC, created_at ASC;

-- name: ListPublicStatusPageComponents :many
SELECT id, status_page_id, organization_id, name, description, status, position, created_at, updated_at
FROM status_page_components
WHERE status_page_id = $1
  AND organization_id = $2
ORDER BY position ASC, created_at ASC;

-- name: GetStatusPageComponentByID :one
SELECT *
FROM status_page_components
WHERE id = $1
  AND organization_id = $2
LIMIT 1;

-- name: CreateStatusPageComponent :one
INSERT INTO status_page_components (
    id,
    status_page_id,
    organization_id,
    name,
    description,
    status,
    position,
    service_id
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
RETURNING *;

-- name: UpdateStatusPageComponent :one
UPDATE status_page_components
SET name = $3,
    description = $4,
    status = $5,
    position = $6,
    service_id = $7,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
RETURNING *;

-- name: DeleteStatusPageComponent :exec
DELETE FROM status_page_components
WHERE id = $1
  AND organization_id = $2;

-- name: CreateStatusPageSubscription :one
INSERT INTO status_page_subscriptions (
    id,
    status_page_id,
    organization_id,
    email
) VALUES (
    $1,
    $2,
    $3,
    $4
)
ON CONFLICT (status_page_id, email) DO UPDATE
SET unsubscribed_at = NULL
RETURNING *;

-- name: ListStatusPageSubscriptions :many
SELECT *
FROM status_page_subscriptions
WHERE status_page_id = $1
  AND organization_id = $2
  AND unsubscribed_at IS NULL
ORDER BY created_at ASC;

-- name: GetStatusPageIncidentByID :one
SELECT *
FROM status_page_incidents
WHERE id = $1
  AND organization_id = $2
LIMIT 1;

-- name: ListActiveStatusPageIncidents :many
SELECT *
FROM status_page_incidents
WHERE status_page_id = $1
  AND organization_id = $2
  AND status <> 'resolved'
ORDER BY created_at DESC;

-- name: ListPublicStatusPageIncidents :many
SELECT id, status_page_id, organization_id, title, status, resolved_at, created_at, updated_at
FROM status_page_incidents
WHERE status_page_id = $1
  AND organization_id = $2
  AND status <> 'resolved'
ORDER BY created_at DESC;

-- name: ListPublicStatusPageResolvedIncidents :many
SELECT id, status_page_id, organization_id, title, status, resolved_at, created_at, updated_at
FROM status_page_incidents
WHERE status_page_id = $1
  AND organization_id = $2
  AND status = 'resolved'
  AND resolved_at >= now() - (sqlc.arg(resolved_window_days)::integer * interval '1 day')
ORDER BY resolved_at DESC;

-- name: CreateStatusPageIncident :one
INSERT INTO status_page_incidents (
    id,
    status_page_id,
    organization_id,
    incident_id,
    title,
    status
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING *;

-- name: UpdateStatusPageIncidentStatus :one
UPDATE status_page_incidents
SET status = $3,
    resolved_at = CASE
        WHEN $3 = 'resolved' THEN now()
        ELSE NULL
    END,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
RETURNING *;

-- name: ListStatusPageIncidentUpdates :many
SELECT *
FROM status_page_incident_updates
WHERE status_page_incident_id = $1
  AND organization_id = $2
ORDER BY created_at ASC;

-- name: ListPublicStatusPageIncidentUpdates :many
SELECT id, status_page_incident_id, organization_id, body, status, created_at
FROM status_page_incident_updates
WHERE status_page_incident_id = $1
  AND organization_id = $2
ORDER BY created_at ASC;

-- name: CreateStatusPageIncidentUpdate :one
INSERT INTO status_page_incident_updates (
    id,
    status_page_incident_id,
    organization_id,
    body,
    status
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
RETURNING *;

-- name: ListStatusPageIncidentComponentIDs :many
SELECT status_page_component_id
FROM status_page_incident_components
WHERE status_page_incident_id = $1
  AND organization_id = $2
ORDER BY created_at ASC;

-- name: ReplaceStatusPageIncidentComponents :exec
DELETE FROM status_page_incident_components
WHERE status_page_incident_id = $1
  AND organization_id = $2;

-- name: InsertStatusPageIncidentComponent :exec
INSERT INTO status_page_incident_components (
    status_page_incident_id,
    status_page_component_id,
    organization_id
) VALUES (
    $1,
    $2,
    $3
);
