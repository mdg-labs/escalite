package escalation

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/mdg-labs/escalite/services/engine/internal/db"
	"github.com/mdg-labs/escalite/services/engine/internal/jobs"
)

const maxSnoozeDurationMinutes = 7 * 24 * 60

// SnoozeEscalation delays the next escalation step by the requested duration.
func SnoozeEscalation(
	ctx context.Context,
	q db.Querier,
	canceller JobCanceller,
	inserter JobInserter,
	alertID, organizationID uuid.UUID,
	durationMinutes int32,
) (db.Alert, error) {
	if durationMinutes <= 0 {
		return db.Alert{}, fmt.Errorf("duration must be positive")
	}
	if durationMinutes > maxSnoozeDurationMinutes {
		return db.Alert{}, fmt.Errorf("duration exceeds maximum of %d minutes", maxSnoozeDurationMinutes)
	}

	alert, err := q.GetAlertByID(ctx, db.GetAlertByIDParams{
		ID:             alertID,
		OrganizationID: organizationID,
	})
	if err != nil {
		return db.Alert{}, fmt.Errorf("load alert: %w", err)
	}
	if alert.Status != statusTriggered {
		return db.Alert{}, fmt.Errorf("only triggered alerts can be snoozed")
	}

	state, err := parseState(alert.EscalationState)
	if err != nil {
		return db.Alert{}, err
	}
	if state.EscalatedExhausted {
		return db.Alert{}, fmt.Errorf("escalation is exhausted")
	}
	if state.CurrentStep < 1 {
		return db.Alert{}, fmt.Errorf("alert has no active escalation step")
	}

	if state.PendingEscalationJobID != nil {
		if err := canceller.CancelJob(ctx, *state.PendingEscalationJobID); err != nil {
			return db.Alert{}, fmt.Errorf("cancel pending escalation job: %w", err)
		}
		state.PendingEscalationJobID = nil
	}

	base := time.Now()
	if state.NextEscalationAt != nil {
		base = *state.NextEscalationAt
	}
	nextAt := base.Add(time.Duration(durationMinutes) * time.Minute)

	result, err := inserter.Insert(ctx, jobs.EscalationStepArgs{
		AlertID:        alertID,
		OrganizationID: organizationID,
		FromStep:       state.CurrentStep,
	}, &river.InsertOpts{
		ScheduledAt: nextAt,
	})
	if err != nil {
		return db.Alert{}, fmt.Errorf("enqueue snoozed escalation job: %w", err)
	}

	jobID := result.Job.ID
	state.NextEscalationAt = &nextAt
	state.PendingEscalationJobID = &jobID

	raw, err := marshalState(state)
	if err != nil {
		return db.Alert{}, err
	}

	updated, err := q.UpdateAlertEscalationState(ctx, db.UpdateAlertEscalationStateParams{
		ID:              alertID,
		OrganizationID:  organizationID,
		EscalationState: raw,
	})
	if err != nil {
		return db.Alert{}, fmt.Errorf("update escalation state: %w", err)
	}

	return updated, nil
}
