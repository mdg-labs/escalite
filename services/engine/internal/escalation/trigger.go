package escalation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/mdg-labs/escalite/services/engine/internal/db"
	"github.com/mdg-labs/escalite/services/engine/internal/jobs"
	"github.com/mdg-labs/escalite/services/engine/internal/notificationrules"
	"github.com/mdg-labs/escalite/services/engine/oncall"
)

const (
	statusTriggered    = "triggered"
	statusPending      = "pending"
	targetTypeUser     = "user"
	targetTypeRotation = "rotation"
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

	step, err := q.GetEscalationStepByPolicyAndOrder(ctx, db.GetEscalationStepByPolicyAndOrderParams{
		EscalationPolicyID: policies[0].ID,
		OrganizationID:     organizationID,
		StepOrder:          1,
	})
	if err != nil {
		return fmt.Errorf("load step 1: %w", err)
	}

	if err := scheduleStepNotifications(ctx, q, inserter, alert, State{}, 1); err != nil {
		return err
	}

	_, step2Err := q.GetEscalationStepByPolicyAndOrder(ctx, db.GetEscalationStepByPolicyAndOrderParams{
		EscalationPolicyID: policies[0].ID,
		OrganizationID:     organizationID,
		StepOrder:          2,
	})
	if step2Err == nil {
		timerState := State{CurrentStep: 1}
		return scheduleEscalationTimer(ctx, q, inserter, alertID, organizationID, timerState, step.DelayMinutes)
	}
	if !errors.Is(step2Err, pgx.ErrNoRows) {
		return fmt.Errorf("load step 2: %w", step2Err)
	}

	timerState := State{CurrentStep: 1}
	return scheduleAfterStep(ctx, q, inserter, alertID, organizationID, timerState, step, false)
}

func scheduleStepNotifications(
	ctx context.Context,
	q db.Querier,
	inserter JobInserter,
	alert db.Alert,
	priorState State,
	stepOrder int,
) error {
	policies, err := q.ListEscalationPoliciesByServiceID(ctx, db.ListEscalationPoliciesByServiceIDParams{
		ServiceID:      alert.ServiceID,
		OrganizationID: alert.OrganizationID,
	})
	if err != nil {
		return fmt.Errorf("list escalation policies: %w", err)
	}
	if len(policies) == 0 {
		return fmt.Errorf("no escalation policy for service %s", alert.ServiceID)
	}

	step, err := q.GetEscalationStepByPolicyAndOrder(ctx, db.GetEscalationStepByPolicyAndOrderParams{
		EscalationPolicyID: policies[0].ID,
		OrganizationID:     alert.OrganizationID,
		StepOrder:          int32(stepOrder),
	})
	if err != nil {
		return fmt.Errorf("load step %d: %w", stepOrder, err)
	}

	targets, err := q.ListEscalationStepTargetsByStepID(ctx, db.ListEscalationStepTargetsByStepIDParams{
		EscalationStepID: step.ID,
		OrganizationID:   alert.OrganizationID,
	})
	if err != nil {
		return fmt.Errorf("list step targets: %w", err)
	}
	if len(targets) == 0 {
		return fmt.Errorf("step %d has no targets", stepOrder)
	}

	state := State{
		CurrentStep:        stepOrder,
		RepeatCount:        priorState.RepeatCount,
		EscalatedExhausted: priorState.EscalatedExhausted,
	}
	raw, err := marshalState(state)
	if err != nil {
		return err
	}
	if _, err := q.UpdateAlertEscalationState(ctx, db.UpdateAlertEscalationStateParams{
		ID:              alert.ID,
		OrganizationID:  alert.OrganizationID,
		EscalationState: raw,
	}); err != nil {
		return fmt.Errorf("update escalation state: %w", err)
	}

	stepID := pgtype.UUID{Bytes: step.ID, Valid: true}

	for _, target := range targets {
		resolved, err := resolveTarget(ctx, q, alert.OrganizationID, target, time.Now().UTC())
		if err != nil {
			return err
		}
		if len(resolved) == 0 {
			continue
		}

		for _, item := range resolved {
			userID, err := uuid.Parse(strings.TrimSpace(item.recipient.UserID))
			if err != nil {
				return fmt.Errorf("invalid user id for notification rules: %w", err)
			}

			channelSchedules, err := notificationrules.ResolveChannelSchedule(
				ctx,
				q,
				alert.OrganizationID,
				userID,
				alert.Priority,
				item.channels,
			)
			if err != nil {
				return err
			}

			for _, schedule := range channelSchedules {
				channel := schedule.Channel
				recipient := item.recipient
				if channel == "slack-dm" {
					config, skip, err := slackDMRecipientConfig(ctx, q, alert.OrganizationID, recipient.UserID, alert.ID)
					if err != nil {
						return err
					}
					if skip {
						continue
					}
					recipient.Config = config
				}
				if channel == "push" {
					config, err := pushRecipientConfig(ctx, q, alert.OrganizationID, recipient.UserID)
					if err != nil {
						return err
					}
					recipient.Config = config
				}

				attemptID := uuid.Must(uuid.NewV7())
				recipientJSON, err := json.Marshal(recipient)
				if err != nil {
					return fmt.Errorf("marshal recipient: %w", err)
				}

				if _, err := q.CreateNotificationAttempt(ctx, db.CreateNotificationAttemptParams{
					ID:               attemptID,
					OrganizationID:   alert.OrganizationID,
					AlertID:          alert.ID,
					EscalationStepID: stepID,
					Channel:          channel,
					Status:           statusPending,
					Recipient:        recipientJSON,
				}); err != nil {
					return fmt.Errorf("create notification attempt: %w", err)
				}

				scheduledAt := time.Now().UTC().Add(time.Duration(schedule.DelayMinutes) * time.Minute)
				if _, err := inserter.Insert(ctx, jobs.NotifyArgs{
					NotificationAttemptID: attemptID,
					OrganizationID:      alert.OrganizationID,
				}, &river.InsertOpts{
					MaxAttempts: jobs.NotifyMaxAttempts,
					ScheduledAt: scheduledAt,
				}); err != nil {
					return fmt.Errorf("enqueue notify job: %w", err)
				}
			}
		}
	}

	return nil
}

func scheduleEscalationTimer(
	ctx context.Context,
	q db.Querier,
	inserter JobInserter,
	alertID, organizationID uuid.UUID,
	state State,
	delayMinutes int32,
) error {
	nextAt := time.Now().Add(time.Duration(delayMinutes) * time.Minute)
	result, err := inserter.Insert(ctx, jobs.EscalationStepArgs{
		AlertID:        alertID,
		OrganizationID: organizationID,
		FromStep:       state.CurrentStep,
	}, &river.InsertOpts{
		ScheduledAt: nextAt,
	})
	if err != nil {
		return fmt.Errorf("enqueue escalation step job: %w", err)
	}

	jobID := result.Job.ID
	state.NextEscalationAt = &nextAt
	state.PendingEscalationJobID = &jobID

	raw, err := marshalState(state)
	if err != nil {
		return err
	}
	if _, err := q.UpdateAlertEscalationState(ctx, db.UpdateAlertEscalationStateParams{
		ID:              alertID,
		OrganizationID:  organizationID,
		EscalationState: raw,
	}); err != nil {
		return fmt.Errorf("update escalation state: %w", err)
	}

	return nil
}

type recipientPayload struct {
	Type   string          `json:"type"`
	UserID string          `json:"user_id,omitempty"`
	Email  string          `json:"email,omitempty"`
	URL    string          `json:"url,omitempty"`
	Config json.RawMessage `json:"config,omitempty"`
}

type resolvedNotification struct {
	channels  []string
	recipient recipientPayload
}

func resolveTarget(
	ctx context.Context,
	q db.Querier,
	organizationID uuid.UUID,
	target db.EscalationStepTarget,
	at time.Time,
) ([]resolvedNotification, error) {
	channels, err := parseChannels(target.Channels)
	if err != nil {
		return nil, fmt.Errorf("parse channels: %w", err)
	}

	switch target.TargetType {
	case targetTypeUser:
		if !target.UserID.Valid {
			return nil, fmt.Errorf("user target missing user_id")
		}
		user, err := q.GetUserByID(ctx, db.GetUserByIDParams{
			ID:             uuid.UUID(target.UserID.Bytes),
			OrganizationID: organizationID,
		})
		if err != nil {
			return nil, fmt.Errorf("load user target: %w", err)
		}
		return []resolvedNotification{{
			channels: channels,
			recipient: recipientPayload{
				Type:   targetTypeUser,
				UserID: user.ID.String(),
				Email:  user.Email,
			},
		}}, nil
	case targetTypeRotation:
		if !target.ScheduleID.Valid {
			return nil, fmt.Errorf("rotation target missing schedule_id")
		}
		scheduleID := uuid.UUID(target.ScheduleID.Bytes)
		userIDs, err := oncall.UsersAt(ctx, q, organizationID, scheduleID, at)
		if err != nil {
			return nil, fmt.Errorf("resolve rotation schedule %s: %w", scheduleID, err)
		}
		if len(userIDs) == 0 {
			slog.Default().Info(
				"skipping rotation target with no on-call users",
				"schedule_id", scheduleID,
				"organization_id", organizationID,
			)
			return nil, nil
		}

		resolved := make([]resolvedNotification, 0, len(userIDs))
		for _, userID := range userIDs {
			user, err := q.GetUserByID(ctx, db.GetUserByIDParams{
				ID:             userID,
				OrganizationID: organizationID,
			})
			if err != nil {
				return nil, fmt.Errorf("load on-call user %s: %w", userID, err)
			}
			resolved = append(resolved, resolvedNotification{
				channels: channels,
				recipient: recipientPayload{
					Type:   targetTypeUser,
					UserID: user.ID.String(),
					Email:  user.Email,
				},
			})
		}
		return resolved, nil
	default:
		return nil, fmt.Errorf("unsupported target type %q", target.TargetType)
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

func slackDMRecipientConfig(
	ctx context.Context,
	q db.Querier,
	organizationID uuid.UUID,
	userID string,
	alertID uuid.UUID,
) (json.RawMessage, bool, error) {
	parsedUserID, err := uuid.Parse(strings.TrimSpace(userID))
	if err != nil {
		return nil, false, fmt.Errorf("invalid user id for slack-dm: %w", err)
	}

	contact, err := q.GetUserContactMethodByChannel(ctx, db.GetUserContactMethodByChannelParams{
		OrganizationID: organizationID,
		UserID:         parsedUserID,
		Channel:        "slack-dm",
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Default().Info(
				"skipping slack-dm notification: missing slack_user_id",
				"user_id", parsedUserID,
				"organization_id", organizationID,
				"alert_id", alertID,
			)
			return nil, true, nil
		}
		return nil, false, fmt.Errorf("load slack-dm contact method: %w", err)
	}

	slackUserID, err := slackUserIDFromConfig(contact.Config)
	if err != nil {
		return nil, false, err
	}
	if slackUserID == "" {
		slog.Default().Info(
			"skipping slack-dm notification: missing slack_user_id",
			"user_id", parsedUserID,
			"organization_id", organizationID,
			"alert_id", alertID,
		)
		return nil, true, nil
	}

	raw, err := json.Marshal(map[string]string{"slack_user_id": slackUserID})
	if err != nil {
		return nil, false, fmt.Errorf("marshal slack-dm config: %w", err)
	}
	return raw, false, nil
}

func slackUserIDFromConfig(raw []byte) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}
	var cfg struct {
		SlackUserID string `json:"slack_user_id"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return "", fmt.Errorf("parse slack-dm contact config: %w", err)
	}
	return strings.TrimSpace(cfg.SlackUserID), nil
}

func pushRecipientConfig(
	ctx context.Context,
	q db.Querier,
	organizationID uuid.UUID,
	userID string,
) (json.RawMessage, error) {
	parsedUserID, err := uuid.Parse(strings.TrimSpace(userID))
	if err != nil {
		return nil, fmt.Errorf("invalid user id for push: %w", err)
	}

	contact, err := q.GetUserContactMethodByChannel(ctx, db.GetUserContactMethodByChannelParams{
		OrganizationID: organizationID,
		UserID:         parsedUserID,
		Channel:        "push",
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("load push contact method: %w", err)
	}

	token, err := expoPushTokenFromConfig(contact.Config)
	if err != nil {
		return nil, err
	}
	if token == "" {
		return nil, nil
	}

	raw, err := json.Marshal(map[string]string{"expo_push_token": token})
	if err != nil {
		return nil, fmt.Errorf("marshal push config: %w", err)
	}
	return raw, nil
}

func expoPushTokenFromConfig(raw []byte) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}
	var cfg struct {
		ExpoPushToken string `json:"expo_push_token"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return "", fmt.Errorf("parse push contact config: %w", err)
	}
	return strings.TrimSpace(cfg.ExpoPushToken), nil
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
	Source         string
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

	initialState, err := initialEscalationState(params.Source)
	if err != nil {
		return db.Alert{}, fmt.Errorf("marshal escalation state: %w", err)
	}
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

func initialEscalationState(source string) ([]byte, error) {
	if source == "" {
		return []byte(`{}`), nil
	}
	return json.Marshal(map[string]string{"source": source})
}
