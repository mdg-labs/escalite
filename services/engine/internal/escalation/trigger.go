package escalation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/mdg-labs/escalite/services/engine/internal/db"
	"github.com/mdg-labs/escalite/services/engine/internal/jobs"
)

const (
	statusTriggered = "triggered"
	statusPending   = "pending"
	targetTypeUser  = "user"
)

// JobInserter inserts River jobs, typically a river.Client.
type JobInserter interface {
	Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

// ScheduleStep1Notifications creates notification_attempt rows for step-1 targets and enqueues notify jobs.
func ScheduleStep1Notifications(
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
		return fmt.Errorf("alert %s is not triggered", alertID)
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

	policy := policies[0]
	step, err := q.GetEscalationStepByPolicyAndOrder(ctx, db.GetEscalationStepByPolicyAndOrderParams{
		EscalationPolicyID: policy.ID,
		OrganizationID:     organizationID,
		StepOrder:          1,
	})
	if err != nil {
		return fmt.Errorf("load step 1: %w", err)
	}

	targets, err := q.ListEscalationStepTargetsByStepID(ctx, db.ListEscalationStepTargetsByStepIDParams{
		EscalationStepID: step.ID,
		OrganizationID:   organizationID,
	})
	if err != nil {
		return fmt.Errorf("list step targets: %w", err)
	}
	if len(targets) == 0 {
		return fmt.Errorf("step 1 has no targets")
	}

	escalationState, err := json.Marshal(map[string]int{"current_step": 1})
	if err != nil {
		return fmt.Errorf("marshal escalation state: %w", err)
	}
	if _, err := q.UpdateAlertEscalationState(ctx, db.UpdateAlertEscalationStateParams{
		ID:              alertID,
		OrganizationID:  organizationID,
		EscalationState: escalationState,
	}); err != nil {
		return fmt.Errorf("update escalation state: %w", err)
	}

	stepID := pgtype.UUID{Bytes: step.ID, Valid: true}

	for _, target := range targets {
		channels, recipient, err := resolveTarget(ctx, q, organizationID, target)
		if err != nil {
			return err
		}

		for _, channel := range channels {
			attemptID := uuid.Must(uuid.NewV7())
			recipientJSON, err := json.Marshal(recipient)
			if err != nil {
				return fmt.Errorf("marshal recipient: %w", err)
			}

			if _, err := q.CreateNotificationAttempt(ctx, db.CreateNotificationAttemptParams{
				ID:               attemptID,
				OrganizationID:   organizationID,
				AlertID:          alertID,
				EscalationStepID: stepID,
				Channel:          channel,
				Status:           statusPending,
				Recipient:        recipientJSON,
			}); err != nil {
				return fmt.Errorf("create notification attempt: %w", err)
			}

			if _, err := inserter.Insert(ctx, jobs.NotifyArgs{
				NotificationAttemptID: attemptID,
				OrganizationID:        organizationID,
			}, nil); err != nil {
				return fmt.Errorf("enqueue notify job: %w", err)
			}
		}
	}

	return nil
}

type recipientPayload struct {
	Type   string `json:"type"`
	UserID string `json:"user_id,omitempty"`
	Email  string `json:"email,omitempty"`
	URL    string `json:"url,omitempty"`
}

func resolveTarget(
	ctx context.Context,
	q db.Querier,
	organizationID uuid.UUID,
	target db.EscalationStepTarget,
) ([]string, recipientPayload, error) {
	channels, err := parseChannels(target.Channels)
	if err != nil {
		return nil, recipientPayload{}, fmt.Errorf("parse channels: %w", err)
	}

	switch target.TargetType {
	case targetTypeUser:
		if !target.UserID.Valid {
			return nil, recipientPayload{}, fmt.Errorf("user target missing user_id")
		}
		user, err := q.GetUserByID(ctx, db.GetUserByIDParams{
			ID:             uuid.UUID(target.UserID.Bytes),
			OrganizationID: organizationID,
		})
		if err != nil {
			return nil, recipientPayload{}, fmt.Errorf("load user target: %w", err)
		}
		return channels, recipientPayload{
			Type:   targetTypeUser,
			UserID: user.ID.String(),
			Email:  user.Email,
		}, nil
	default:
		return nil, recipientPayload{}, fmt.Errorf("unsupported target type %q", target.TargetType)
	}
}

func parseChannels(raw []byte) ([]string, error) {
	if len(raw) == 0 {
		return []string{"email"}, nil
	}
	var channels []string
	if err := json.Unmarshal(raw, &channels); err != nil {
		return nil, err
	}
	if len(channels) == 0 {
		return []string{"email"}, nil
	}
	return channels, nil
}

// CreateTriggeredAlertParams configures a new triggered alert and schedules step-1 escalation.
type CreateTriggeredAlertParams struct {
	AlertID        uuid.UUID
	OrganizationID uuid.UUID
	ServiceID      uuid.UUID
	DedupKey       string
	Summary        string
	Description    pgtype.Text
	Priority       string
}

// CreateTriggeredAlert inserts a triggered alert and enqueues step-1 escalation in one transaction.
func CreateTriggeredAlert(
	ctx context.Context,
	pool *pgxpool.Pool,
	inserter JobInserter,
	params CreateTriggeredAlertParams,
) (db.Alert, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return db.Alert{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	q := db.New(tx)

	initialState := []byte(`{}`)
	alert, err := q.CreateTriggeredAlert(ctx, db.CreateTriggeredAlertParams{
		ID:               params.AlertID,
		OrganizationID:   params.OrganizationID,
		ServiceID:        params.ServiceID,
		IntegrationKeyID: pgtype.UUID{},
		DedupKey:         params.DedupKey,
		Summary:          params.Summary,
		Description:      params.Description,
		Priority:         params.Priority,
		EscalationState:  initialState,
	})
	if err != nil {
		return db.Alert{}, fmt.Errorf("create alert: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Alert{}, fmt.Errorf("commit transaction: %w", err)
	}

	if _, err := inserter.Insert(ctx, jobs.EscalationTriggerArgs{
		AlertID:        alert.ID,
		OrganizationID: alert.OrganizationID,
	}, nil); err != nil {
		return db.Alert{}, fmt.Errorf("enqueue escalation trigger: %w", err)
	}

	return alert, nil
}
