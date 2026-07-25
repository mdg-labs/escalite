package escalation

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mdg-labs/escalite/services/engine/internal/db"
)

// canRepeatLastStep reports whether the last step may fire another repeat cycle.
func canRepeatLastStep(repeatLastStep bool, maxRepeats pgtype.Int4, repeatCount int) bool {
	if !repeatLastStep || !maxRepeats.Valid {
		return false
	}
	return repeatCount < int(maxRepeats.Int32)
}

func markEscalationExhausted(
	ctx context.Context,
	q db.Querier,
	alertID, organizationID uuid.UUID,
	state State,
) error {
	state = clearedTimerState(state)
	state.EscalatedExhausted = true

	raw, err := marshalState(state)
	if err != nil {
		return err
	}
	if _, err := q.UpdateAlertEscalationState(ctx, db.UpdateAlertEscalationStateParams{
		ID:              alertID,
		OrganizationID:  organizationID,
		EscalationState: raw,
	}); err != nil {
		return fmt.Errorf("mark escalation exhausted: %w", err)
	}
	return nil
}

func scheduleAfterStep(
	ctx context.Context,
	q db.Querier,
	inserter JobInserter,
	alertID, organizationID uuid.UUID,
	state State,
	step db.EscalationStep,
	hasNextStep bool,
) error {
	if hasNextStep {
		return scheduleEscalationTimer(ctx, q, inserter, alertID, organizationID, state, step.DelayMinutes)
	}
	if !canRepeatLastStep(step.RepeatLastStep, step.MaxRepeats, state.RepeatCount) {
		return nil
	}
	return scheduleEscalationTimer(ctx, q, inserter, alertID, organizationID, state, step.DelayMinutes)
}

func handleLastStepRepeat(
	ctx context.Context,
	q db.Querier,
	inserter JobInserter,
	alert db.Alert,
	state State,
	lastStep db.EscalationStep,
) error {
	if !canRepeatLastStep(lastStep.RepeatLastStep, lastStep.MaxRepeats, state.RepeatCount) {
		return markEscalationExhausted(ctx, q, alert.ID, alert.OrganizationID, state)
	}

	state.RepeatCount++
	if err := scheduleStepNotifications(ctx, q, inserter, alert, state, int(lastStep.StepOrder), nil); err != nil {
		return err
	}

	if canRepeatLastStep(lastStep.RepeatLastStep, lastStep.MaxRepeats, state.RepeatCount) {
		timerState := State{
			CurrentStep:       int(lastStep.StepOrder),
			RepeatCount:       state.RepeatCount,
			EscalatedExhausted: state.EscalatedExhausted,
		}
		return scheduleEscalationTimer(ctx, q, inserter, alert.ID, alert.OrganizationID, timerState, lastStep.DelayMinutes)
	}

	return markEscalationExhausted(ctx, q, alert.ID, alert.OrganizationID, State{
		CurrentStep:        int(lastStep.StepOrder),
		RepeatCount:        state.RepeatCount,
		EscalatedExhausted: state.EscalatedExhausted,
	})
}
