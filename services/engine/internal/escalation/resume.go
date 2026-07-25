package escalation

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mdg-labs/escalite/services/engine/internal/db"
	"github.com/mdg-labs/escalite/services/engine/internal/incident"
)

// ResumeEscalationOnIncidentClose schedules step-1 notifications for triggered alerts whose
// escalation was suppressed while grouped under a resolved incident.
func ResumeEscalationOnIncidentClose(
	ctx context.Context,
	q db.Querier,
	inserter JobInserter,
	incidentID, organizationID uuid.UUID,
) error {
	alerts, err := q.ListAlertsByIncidentID(ctx, db.ListAlertsByIncidentIDParams{
		IncidentID:     pgtype.UUID{Bytes: incidentID, Valid: true},
		OrganizationID: organizationID,
	})
	if err != nil {
		return fmt.Errorf("list incident alerts: %w", err)
	}

	for _, alert := range alerts {
		if alert.Status != statusTriggered {
			continue
		}

		service, err := q.GetServiceByID(ctx, db.GetServiceByIDParams{
			ID:             alert.ServiceID,
			OrganizationID: organizationID,
		})
		if err != nil {
			return fmt.Errorf("load service for alert %s: %w", alert.ID, err)
		}
		if !incident.PrioritySuppressed(alert.Priority, service.AutoPromoteSuppressEscalationPriorities) {
			continue
		}

		count, err := q.CountNotificationAttemptsByAlertID(ctx, db.CountNotificationAttemptsByAlertIDParams{
			AlertID:        alert.ID,
			OrganizationID: organizationID,
		})
		if err != nil {
			return fmt.Errorf("count notification attempts for alert %s: %w", alert.ID, err)
		}
		if count > 0 {
			continue
		}

		if err := ScheduleStep1Notifications(ctx, q, inserter, alert.ID, organizationID); err != nil {
			return fmt.Errorf("resume escalation for alert %s: %w", alert.ID, err)
		}
	}

	return nil
}
