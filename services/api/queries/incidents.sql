-- name: CreateIncident :one
INSERT INTO incidents (
    id,
    organization_id,
    team_id,
    title,
    status,
    created_by_user_id
) VALUES (
    $1,
    $2,
    $3,
    $4,
    'investigating',
    $5
)
RETURNING *;

-- name: GetIncidentByID :one
SELECT *
FROM incidents
WHERE id = $1
  AND organization_id = $2
LIMIT 1;

-- name: UpdateIncidentSlackChannelID :one
UPDATE incidents
SET slack_channel_id = $3,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
RETURNING *;

-- name: UpdateIncidentSlackThreadTS :one
UPDATE incidents
SET slack_thread_ts = $3,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
RETURNING *;

-- name: UpdateIncidentTicketURL :one
UPDATE incidents
SET ticket_url = $3,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
RETURNING *;

-- name: ListIncidentsForOrgAdmin :many
SELECT *
FROM incidents
WHERE organization_id = $1
  AND (
    sqlc.narg('status_filter')::text IS NULL
    OR status = sqlc.narg('status_filter')
  )
  AND (
    sqlc.narg('team_id_filter')::uuid IS NULL
    OR team_id = sqlc.narg('team_id_filter')
  )
ORDER BY created_at DESC
LIMIT $2;

-- name: ListIncidentsForTeamMember :many
SELECT i.*
FROM incidents i
INNER JOIN team_memberships tm
  ON tm.team_id = i.team_id
 AND tm.user_id = $2
 AND tm.organization_id = i.organization_id
WHERE i.organization_id = $1
  AND (
    sqlc.narg('status_filter')::text IS NULL
    OR i.status = sqlc.narg('status_filter')
  )
  AND (
    sqlc.narg('team_id_filter')::uuid IS NULL
    OR i.team_id = sqlc.narg('team_id_filter')
  )
ORDER BY i.created_at DESC
LIMIT $3;

-- name: UpdateIncidentStatus :one
UPDATE incidents
SET status = $3,
    resolved_at = CASE
        WHEN $3 = 'resolved' THEN now()
        ELSE NULL
    END,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
RETURNING *;

-- name: CreateTimelineEvent :one
INSERT INTO timeline_events (
    id,
    incident_id,
    organization_id,
    actor_id,
    event_type,
    body,
    metadata
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7
)
RETURNING *;

-- name: ListTimelineEventsByIncidentID :many
SELECT *
FROM timeline_events
WHERE incident_id = $1
  AND organization_id = $2
ORDER BY created_at ASC;

-- name: GetTimelineEventByID :one
SELECT *
FROM timeline_events
WHERE id = $1
  AND organization_id = $2
LIMIT 1;

-- name: ListAlertsByIncidentID :many
SELECT *
FROM alerts
WHERE incident_id = $1
  AND organization_id = $2
ORDER BY created_at ASC;

-- name: GetOpenIncidentForTeamWithServiceAlerts :one
SELECT DISTINCT i.*
FROM incidents i
INNER JOIN alerts a
  ON a.incident_id = i.id
 AND a.organization_id = i.organization_id
WHERE i.organization_id = $1
  AND i.team_id = $2
  AND i.status <> 'resolved'
  AND a.service_id = $3
ORDER BY i.created_at DESC
LIMIT 1;

-- name: GetOpenIncidentForTeam :one
SELECT *
FROM incidents
WHERE organization_id = $1
  AND team_id = $2
  AND status <> 'resolved'
ORDER BY created_at DESC
LIMIT 1;

-- name: CreateIncidentRoleDefinition :one
INSERT INTO incident_role_definitions (
    id,
    organization_id,
    name,
    sort_order
) VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING *;

-- name: GetIncidentRoleDefinitionByID :one
SELECT *
FROM incident_role_definitions
WHERE id = $1
  AND organization_id = $2
LIMIT 1;

-- name: ListIncidentRoleDefinitions :many
SELECT *
FROM incident_role_definitions
WHERE organization_id = $1
ORDER BY sort_order ASC, name ASC;

-- name: UpdateIncidentRoleDefinition :one
UPDATE incident_role_definitions
SET name = $3,
    sort_order = $4,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
RETURNING *;

-- name: DeleteIncidentRoleDefinition :exec
DELETE FROM incident_role_definitions
WHERE id = $1
  AND organization_id = $2;

-- name: CreateIncidentRoleAssignment :one
INSERT INTO incident_role_assignments (
    id,
    incident_id,
    organization_id,
    role_definition_id,
    user_id,
    assigned_by_user_id
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING *;

-- name: GetIncidentRoleAssignmentByID :one
SELECT *
FROM incident_role_assignments
WHERE id = $1
  AND organization_id = $2
LIMIT 1;

-- name: ListIncidentRoleAssignmentsByIncidentID :many
SELECT *
FROM incident_role_assignments
WHERE incident_id = $1
  AND organization_id = $2
ORDER BY created_at ASC;

-- name: DeleteIncidentRoleAssignment :exec
DELETE FROM incident_role_assignments
WHERE id = $1
  AND organization_id = $2;
