-- name: ServiceHasActiveNotificationSuppression :one
SELECT EXISTS (
    SELECT 1
    FROM maintenance_windows
    WHERE service_id = $1
      AND organization_id = $2
      AND suppress_notifications = true
      AND starts_at <= $3
      AND ends_at > $3
) AS active;

-- name: ServiceHasActiveIngestionSuppression :one
SELECT EXISTS (
    SELECT 1
    FROM maintenance_windows
    WHERE service_id = $1
      AND organization_id = $2
      AND suppress_ingestion = true
      AND starts_at <= $3
      AND ends_at > $3
) AS active;
