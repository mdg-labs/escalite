-- name: CreateTriggeredAlert :one
INSERT INTO alerts (
    id,
    organization_id,
    service_id,
    integration_key_id,
    status,
    dedup_key,
    summary,
    description,
    priority,
    escalation_state
) VALUES (
    $1,
    $2,
    $3,
    $4,
    'triggered',
    $5,
    $6,
    $7,
    $8,
    $9
)
RETURNING *;

-- name: GetAlertByID :one
SELECT *
FROM alerts
WHERE id = $1
  AND organization_id = $2
LIMIT 1;

-- name: ListAlertsForOrgAdmin :many
SELECT *
FROM alerts
WHERE organization_id = $1
  AND (
    sqlc.narg('status_filter')::text IS NULL
    OR status = sqlc.narg('status_filter')
  )
ORDER BY created_at DESC
LIMIT $2;

-- name: ListAlertsForTeamMember :many
SELECT a.*
FROM alerts a
INNER JOIN services s
  ON s.id = a.service_id
 AND s.organization_id = a.organization_id
INNER JOIN team_memberships tm
  ON tm.team_id = s.team_id
 AND tm.user_id = $2
 AND tm.organization_id = a.organization_id
WHERE a.organization_id = $1
  AND (
    sqlc.narg('status_filter')::text IS NULL
    OR a.status = sqlc.narg('status_filter')
  )
ORDER BY a.created_at DESC
LIMIT $3;

-- name: GetOpenAlertByServiceDedupKey :one
SELECT *
FROM alerts
WHERE service_id = sqlc.arg(service_id)
  AND dedup_key = sqlc.arg(dedup_key)
  AND status IN ('triggered', 'acknowledged')
  AND updated_at >= now() - (sqlc.arg(dedup_window_seconds)::integer * interval '1 second')
LIMIT 1;

-- name: GetOpenAlertByServiceDedupKeyForResolve :one
SELECT *
FROM alerts
WHERE service_id = sqlc.arg(service_id)
  AND dedup_key = sqlc.arg(dedup_key)
  AND status IN ('triggered', 'acknowledged')
LIMIT 1;

-- name: IncrementOpenAlertEventCount :one
UPDATE alerts
SET event_count = event_count + 1,
    updated_at = now()
WHERE service_id = sqlc.arg(service_id)
  AND dedup_key = sqlc.arg(dedup_key)
  AND status IN ('triggered', 'acknowledged')
  AND updated_at >= now() - (sqlc.arg(dedup_window_seconds)::integer * interval '1 second')
RETURNING *;

-- name: UpdateAlertEscalationState :one
UPDATE alerts
SET escalation_state = $3,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
RETURNING *;

-- name: AcknowledgeAlert :one
UPDATE alerts
SET status = 'acknowledged',
    acknowledged_at = now(),
    acknowledged_by_user_id = $4,
    escalation_state = $3,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
  AND status = 'triggered'
RETURNING *;

-- name: CloseAlert :one
UPDATE alerts
SET status = 'closed',
    closed_at = now(),
    escalation_state = $3,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
  AND status IN ('triggered', 'acknowledged')
RETURNING *;

-- name: ResolveOpenAlert :one
UPDATE alerts
SET status = 'closed',
    closed_at = now(),
    resolved_at = now(),
    resolved_integration = $4,
    escalation_state = $3,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
  AND status IN ('triggered', 'acknowledged')
RETURNING *;

-- name: ReEscalateAlert :one
UPDATE alerts
SET status = 'triggered',
    acknowledged_at = NULL,
    acknowledged_by_user_id = NULL,
    escalation_state = $3,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
  AND status IN ('triggered', 'acknowledged')
RETURNING *;

-- name: AssignAlertToIncident :one
UPDATE alerts
SET incident_id = $3,
    updated_at = now()
WHERE id = $1
  AND organization_id = $2
  AND incident_id IS NULL
  AND status IN ('triggered', 'acknowledged')
RETURNING *;

-- name: CountRecentOpenAlertsByService :one
SELECT count(*)::integer AS count
FROM alerts
WHERE service_id = sqlc.arg(service_id)
  AND organization_id = sqlc.arg(organization_id)
  AND status IN ('triggered', 'acknowledged')
  AND created_at >= now() - (sqlc.arg(window_seconds)::integer * interval '1 second');

-- name: ListRecentUnassignedAlertsByService :many
SELECT *
FROM alerts
WHERE service_id = sqlc.arg(service_id)
  AND organization_id = sqlc.arg(organization_id)
  AND status IN ('triggered', 'acknowledged')
  AND incident_id IS NULL
  AND created_at >= now() - (sqlc.arg(window_seconds)::integer * interval '1 second')
ORDER BY created_at ASC;
