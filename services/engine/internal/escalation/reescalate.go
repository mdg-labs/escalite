package escalation

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/mdg-labs/escalite/services/engine/internal/db"
)

// ReEscalateAlert resets escalation to step 1 and enqueues step-1 notifications.
func ReEscalateAlert(
	ctx context.Context,
	q db.Querier,
	canceller JobCanceller,
	inserter JobInserter,
	alertID, organizationID uuid.UUID,
) (db.Alert, error) {
	alert, err := q.GetAlertByID(ctx, db.GetAlertByIDParams{
		ID:             alertID,
		OrganizationID: organizationID,
	})
	if err != nil {
		return db.Alert{}, fmt.Errorf("load alert: %w", err)
	}
	if alert.Status == "closed" {
		return db.Alert{}, fmt.Errorf("cannot re-escalate a closed alert")
	}

	if err := CancelPendingEscalation(ctx, q, canceller, alertID, organizationID); err != nil {
		return db.Alert{}, err
	}

	resetState := State{}
	raw, err := marshalState(resetState)
	if err != nil {
		return db.Alert{}, err
	}

	alert, err = q.ReEscalateAlert(ctx, db.ReEscalateAlertParams{
		ID:              alertID,
		OrganizationID:  organizationID,
		EscalationState: raw,
	})
	if err != nil {
		return db.Alert{}, fmt.Errorf("reset alert for re-escalation: %w", err)
	}

	if err := ScheduleStep1Notifications(ctx, q, inserter, alertID, organizationID); err != nil {
		return db.Alert{}, fmt.Errorf("schedule step-1 notifications: %w", err)
	}

	alert, err = q.GetAlertByID(ctx, db.GetAlertByIDParams{
		ID:             alertID,
		OrganizationID: organizationID,
	})
	if err != nil {
		return db.Alert{}, fmt.Errorf("reload alert: %w", err)
	}

	return alert, nil
}
