package escalation

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/mdg-labs/escalite/services/engine/internal/db"
)

// AdvanceEscalationStep moves a triggered alert to the next escalation step.
func AdvanceEscalationStep(
	ctx context.Context,
	q db.Querier,
	inserter JobInserter,
	alertID, organizationID uuid.UUID,
) error {
	alert, err := q.GetAlertByID(ctx, db.GetAlertByIDParams{
		ID:             alertID,
		OrganizationID: organizationID,
	})
	if err != nil {
		return fmt.Errorf("load alert: %w", err)
	}
	if alert.Status != statusTriggered {
		return nil
	}

	state, err := parseState(alert.EscalationState)
	if err != nil {
		return err
	}
	if state.CurrentStep < 1 {
		return fmt.Errorf("alert %s has no current escalation step", alertID)
	}

	policies, err := q.ListEscalationPoliciesByServiceID(ctx, db.ListEscalationPoliciesByServiceIDParams{
		ServiceID:      alert.ServiceID,
		OrganizationID: organizationID,
	})
	if err != nil {
		return fmt.Errorf("list escalation policies: %w", err)
	}
	if len(policies) == 0 {
		return fmt.Errorf("no escalation policy for service %s", alert.ServiceID)
	}
	policyID := policies[0].ID

	nextStepOrder := state.CurrentStep + 1
	if err := scheduleStepNotifications(ctx, q, inserter, alert, nextStepOrder); err != nil {
		return err
	}

	currentStep, err := q.GetEscalationStepByPolicyAndOrder(ctx, db.GetEscalationStepByPolicyAndOrderParams{
		EscalationPolicyID: policyID,
		OrganizationID:     organizationID,
		StepOrder:          int32(nextStepOrder),
	})
	if err != nil {
		return fmt.Errorf("load current step: %w", err)
	}

	_, err = q.GetEscalationStepByPolicyAndOrder(ctx, db.GetEscalationStepByPolicyAndOrderParams{
		EscalationPolicyID: policyID,
		OrganizationID:     organizationID,
		StepOrder:          int32(nextStepOrder + 1),
	})
	if err == nil {
		timerState := State{CurrentStep: nextStepOrder}
		return scheduleEscalationTimer(ctx, q, inserter, alertID, organizationID, timerState, currentStep.DelayMinutes)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("load next step: %w", err)
	}

	return nil
}
