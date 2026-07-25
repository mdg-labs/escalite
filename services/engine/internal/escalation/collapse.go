package escalation

import (
	"context"

	"github.com/google/uuid"

	"github.com/mdg-labs/escalite/services/engine/internal/db"
)

// RenotifyCollapsedAlert schedules repeat notifications for the alert's current
// escalation step when a dedup-collapsed alert fires again. Users who already
// acknowledged the alert are skipped.
func RenotifyCollapsedAlert(
	ctx context.Context,
	q db.Querier,
	inserter JobInserter,
	alert db.Alert,
) error {
	if alert.Status == "closed" {
		return nil
	}

	policies, err := q.ListEscalationPoliciesByServiceID(ctx, db.ListEscalationPoliciesByServiceIDParams{
		ServiceID:      alert.ServiceID,
		OrganizationID: alert.OrganizationID,
	})
	if err != nil {
		return err
	}
	if len(policies) == 0 {
		return nil
	}

	state, err := parseState(alert.EscalationState)
	if err != nil {
		return err
	}

	stepOrder := state.CurrentStep
	if stepOrder <= 0 {
		stepOrder = 1
	}

	skipUserIDs := make(map[uuid.UUID]struct{})
	if alert.Status == "acknowledged" && alert.AcknowledgedByUserID.Valid {
		skipUserIDs[uuid.UUID(alert.AcknowledgedByUserID.Bytes)] = struct{}{}
	}

	return scheduleStepNotifications(ctx, q, inserter, alert, state, stepOrder, skipUserIDs)
}
