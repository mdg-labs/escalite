package incident

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/mdg-labs/escalite/services/engine/internal/db"
)

const incidentStatusResolved = "resolved"

// EscalationSuppressed reports whether per-alert escalation should be skipped for an incident-grouped alert.
func EscalationSuppressed(ctx context.Context, q db.Querier, alert db.Alert) (bool, error) {
	if !alert.IncidentID.Valid {
		return false, nil
	}

	incident, err := q.GetIncidentByID(ctx, db.GetIncidentByIDParams{
		ID:             uuid.UUID(alert.IncidentID.Bytes),
		OrganizationID: alert.OrganizationID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("load incident for escalation suppression: %w", err)
	}
	if incident.Status == incidentStatusResolved {
		return false, nil
	}

	service, err := q.GetServiceByID(ctx, db.GetServiceByIDParams{
		ID:             alert.ServiceID,
		OrganizationID: alert.OrganizationID,
	})
	if err != nil {
		return false, fmt.Errorf("load service for escalation suppression: %w", err)
	}

	return PrioritySuppressed(alert.Priority, service.AutoPromoteSuppressEscalationPriorities), nil
}

// PrioritySuppressed reports whether a priority is configured for incident escalation suppression.
func PrioritySuppressed(priority string, configured []string) bool {
	for _, candidate := range configured {
		if candidate == priority {
			return true
		}
	}
	return false
}
