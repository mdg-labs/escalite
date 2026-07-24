package escalation

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/mdg-labs/escalite/services/engine/internal/db"
)

// CloseAlert marks an alert closed and cancels any pending escalation timer.
func CloseAlert(
	ctx context.Context,
	q db.Querier,
	canceller JobCanceller,
	alertID, organizationID uuid.UUID,
) (db.Alert, error) {
	if err := CancelPendingEscalation(ctx, q, canceller, alertID, organizationID); err != nil {
		return db.Alert{}, err
	}

	alert, err := q.GetAlertByID(ctx, db.GetAlertByIDParams{
		ID:             alertID,
		OrganizationID: organizationID,
	})
	if err != nil {
		return db.Alert{}, fmt.Errorf("reload alert: %w", err)
	}

	state, err := parseState(alert.EscalationState)
	if err != nil {
		return db.Alert{}, err
	}
	state = clearedTimerState(state)

	raw, err := marshalState(state)
	if err != nil {
		return db.Alert{}, err
	}

	closed, err := q.CloseAlert(ctx, db.CloseAlertParams{
		ID:              alertID,
		OrganizationID:  organizationID,
		EscalationState: raw,
	})
	if err != nil {
		return db.Alert{}, fmt.Errorf("close alert: %w", err)
	}

	return closed, nil
}
