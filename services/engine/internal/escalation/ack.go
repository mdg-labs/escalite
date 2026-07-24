package escalation

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mdg-labs/escalite/services/engine/internal/db"
)

// JobCanceller cancels a scheduled River job by ID.
type JobCanceller interface {
	CancelJob(ctx context.Context, jobID int64) error
}

// CancelPendingEscalation cancels the pending escalation timer for an alert, if any.
func CancelPendingEscalation(ctx context.Context, q db.Querier, canceller JobCanceller, alertID, organizationID uuid.UUID) error {
	alert, err := q.GetAlertByID(ctx, db.GetAlertByIDParams{
		ID:             alertID,
		OrganizationID: organizationID,
	})
	if err != nil {
		return fmt.Errorf("load alert: %w", err)
	}

	state, err := parseState(alert.EscalationState)
	if err != nil {
		return err
	}
	if state.PendingEscalationJobID == nil {
		return nil
	}

	if err := canceller.CancelJob(ctx, *state.PendingEscalationJobID); err != nil {
		return fmt.Errorf("cancel escalation job: %w", err)
	}

	cleared := clearedTimerState(state)
	raw, err := marshalState(cleared)
	if err != nil {
		return err
	}
	if _, err := q.UpdateAlertEscalationState(ctx, db.UpdateAlertEscalationStateParams{
		ID:              alertID,
		OrganizationID:  organizationID,
		EscalationState: raw,
	}); err != nil {
		return fmt.Errorf("clear pending escalation job: %w", err)
	}

	return nil
}

// AcknowledgeAlert marks an alert acknowledged and cancels any pending escalation timer.
func AcknowledgeAlert(
	ctx context.Context,
	q db.Querier,
	canceller JobCanceller,
	alertID, organizationID, acknowledgedBy uuid.UUID,
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

	acknowledged, err := q.AcknowledgeAlert(ctx, db.AcknowledgeAlertParams{
		ID:                   alertID,
		OrganizationID:       organizationID,
		EscalationState:      raw,
		AcknowledgedByUserID: pgtype.UUID{Bytes: acknowledgedBy, Valid: true},
	})
	if err != nil {
		return db.Alert{}, fmt.Errorf("acknowledge alert: %w", err)
	}

	return acknowledged, nil
}
