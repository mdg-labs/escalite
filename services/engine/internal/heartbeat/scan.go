package heartbeat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/engine/internal/db"
	"github.com/mdg-labs/escalite/services/engine/internal/escalation"
)

const alertSource = "heartbeat"

// ScanOverdueMonitors marks monitors past their interval as overdue and triggers alerts
// for monitors past interval+grace.
func ScanOverdueMonitors(
	ctx context.Context,
	pool *pgxpool.Pool,
	inserter escalation.JobInserter,
	logger *slog.Logger,
) error {
	if logger == nil {
		logger = slog.Default()
	}

	queries := db.New(pool)

	overdueCount, err := queries.MarkHeartbeatMonitorsOverdue(ctx)
	if err != nil {
		return fmt.Errorf("mark heartbeat monitors overdue: %w", err)
	}
	if overdueCount > 0 {
		logger.Info("heartbeat monitors marked overdue", "count", overdueCount)
	}

	monitors, err := queries.ListHeartbeatMonitorsReadyToTrigger(ctx)
	if err != nil {
		return fmt.Errorf("list heartbeat monitors ready to trigger: %w", err)
	}

	for _, monitor := range monitors {
		if err := triggerMonitor(ctx, pool, queries, inserter, monitor, logger); err != nil {
			return err
		}
	}

	if len(monitors) > 0 {
		logger.Info("heartbeat scan completed", "triggered", len(monitors))
	}

	return nil
}

func triggerMonitor(
	ctx context.Context,
	pool *pgxpool.Pool,
	queries *db.Queries,
	inserter escalation.JobInserter,
	monitor db.HeartbeatMonitor,
	logger *slog.Logger,
) error {
	dedupKey := monitor.ID.String()

	existing, err := queries.GetAlertByServiceDedupKey(ctx, db.GetAlertByServiceDedupKeyParams{
		ServiceID:      monitor.ServiceID,
		OrganizationID: monitor.OrganizationID,
		DedupKey:       dedupKey,
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("lookup open heartbeat alert for monitor %s: %w", monitor.ID, err)
	}
	if err == nil {
		logger.Info(
			"heartbeat monitor already has open alert",
			"monitor_id", monitor.ID,
			"alert_id", existing.ID,
		)
		_, err = queries.MarkHeartbeatMonitorTriggered(ctx, db.MarkHeartbeatMonitorTriggeredParams{
			ID:             monitor.ID,
			OrganizationID: monitor.OrganizationID,
		})
		if err != nil {
			return fmt.Errorf("mark heartbeat monitor %s triggered: %w", monitor.ID, err)
		}
		return nil
	}

	alertID, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("generate alert id: %w", err)
	}

	summary := fmt.Sprintf("Heartbeat monitor %q missed check-in", monitor.Name)
	description := pgtype.Text{
		String: fmt.Sprintf(
			"Monitor %s on service %s did not receive a ping within %d seconds plus %d seconds grace.",
			monitor.Name,
			monitor.ServiceID,
			monitor.IntervalSeconds,
			monitor.GraceSeconds,
		),
		Valid: true,
	}

	alert, err := escalation.CreateTriggeredAlert(ctx, pool, inserter, escalation.CreateTriggeredAlertParams{
		AlertID:        alertID,
		OrganizationID: monitor.OrganizationID,
		ServiceID:      monitor.ServiceID,
		DedupKey:       dedupKey,
		Summary:        summary,
		Description:    description,
		Priority:       "high",
		Source:         alertSource,
	})
	if err != nil {
		return fmt.Errorf("create heartbeat alert for monitor %s: %w", monitor.ID, err)
	}

	_, err = queries.MarkHeartbeatMonitorTriggered(ctx, db.MarkHeartbeatMonitorTriggeredParams{
		ID:             monitor.ID,
		OrganizationID: monitor.OrganizationID,
	})
	if err != nil {
		return fmt.Errorf("mark heartbeat monitor %s triggered: %w", monitor.ID, err)
	}

	logger.Info(
		"heartbeat monitor triggered",
		"monitor_id", monitor.ID,
		"alert_id", alert.ID,
		"dedup_key", dedupKey,
	)

	return nil
}

// AlertSourceFromState returns the alert source stored in escalation_state, if present.
func AlertSourceFromState(state []byte) (string, error) {
	if len(state) == 0 {
		return "", nil
	}

	var payload map[string]any
	if err := json.Unmarshal(state, &payload); err != nil {
		return "", err
	}

	source, _ := payload["source"].(string)
	return source, nil
}
