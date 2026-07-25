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
