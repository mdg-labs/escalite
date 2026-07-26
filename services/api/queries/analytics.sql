-- name: GetOrganizationAnalyticsSettings :one
SELECT *
FROM organization_analytics_settings
WHERE organization_id = $1;

-- name: UpsertOrganizationAnalyticsSettings :one
INSERT INTO organization_analytics_settings (
    organization_id,
    exclude_maintenance_window_alerts
) VALUES (
    $1,
    $2
)
ON CONFLICT (organization_id) DO UPDATE SET
    exclude_maintenance_window_alerts = EXCLUDED.exclude_maintenance_window_alerts,
    updated_at = now()
RETURNING *;

-- name: ComputeAlertAnalyticsRollup :one
SELECT
    AVG(
        CASE
            WHEN a.acknowledged_at IS NOT NULL
            THEN EXTRACT(EPOCH FROM (a.acknowledged_at - a.created_at))
        END
    ) AS mtta_seconds,
    SUM(
        CASE WHEN a.acknowledged_at IS NOT NULL THEN 1 ELSE 0 END
    )::integer AS acknowledged_count,
    AVG(
        CASE
            WHEN COALESCE(a.resolved_at, a.closed_at) IS NOT NULL
            THEN EXTRACT(EPOCH FROM (COALESCE(a.resolved_at, a.closed_at) - a.created_at))
        END
    ) AS mttr_seconds,
    SUM(
        CASE WHEN COALESCE(a.resolved_at, a.closed_at) IS NOT NULL THEN 1 ELSE 0 END
    )::integer AS resolved_count
FROM alerts a
INNER JOIN services s
    ON s.id = a.service_id
   AND s.organization_id = a.organization_id
WHERE a.organization_id = sqlc.arg(organization_id)
  AND a.created_at >= sqlc.arg(window_start)
  AND (
    sqlc.narg(team_id)::uuid IS NULL
    OR s.team_id = sqlc.narg(team_id)
  )
  AND (
    sqlc.narg(service_id)::uuid IS NULL
    OR a.service_id = sqlc.narg(service_id)
  )
  AND (
    sqlc.arg(exclude_maintenance_window_alerts)::boolean IS NOT TRUE
    OR NOT EXISTS (
      SELECT 1
      FROM maintenance_windows mw
      WHERE mw.service_id = a.service_id
        AND mw.organization_id = a.organization_id
        AND mw.starts_at <= a.created_at
        AND mw.ends_at > a.created_at
    )
  );
