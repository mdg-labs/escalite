package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/mdg-labs/escalite/services/engine/channels"
	"github.com/mdg-labs/escalite/services/engine/internal/db"
	"github.com/mdg-labs/escalite/services/engine/internal/jobs"
)

const (
	notificationStatusSent   = "sent"
	notificationStatusFailed = "failed"
)

// NotifyWorker processes outbound notification jobs.
type NotifyWorker struct {
	river.WorkerDefaults[jobs.NotifyArgs]
	logger *slog.Logger
	pool   *pgxpool.Pool
}

func NewNotifyWorker(logger *slog.Logger, pool *pgxpool.Pool) *NotifyWorker {
	if logger == nil {
		logger = slog.Default()
	}
	return &NotifyWorker{logger: logger, pool: pool}
}

func (w *NotifyWorker) Work(ctx context.Context, job *river.Job[jobs.NotifyArgs]) error {
	queries := db.New(w.pool)
	attempt, err := queries.GetNotificationAttemptByID(ctx, db.GetNotificationAttemptByIDParams{
		ID:             job.Args.NotificationAttemptID,
		OrganizationID: job.Args.OrganizationID,
	})
	if err != nil {
		return fmt.Errorf("load notification attempt: %w", err)
	}

	alert, err := queries.GetAlertByID(ctx, db.GetAlertByIDParams{
		ID:             attempt.AlertID,
		OrganizationID: attempt.OrganizationID,
	})
	if err != nil {
		return fmt.Errorf("load alert: %w", err)
	}

	channel, err := channels.Get(attempt.Channel)
	if err != nil {
		return w.finishAttempt(ctx, queries, attempt, notificationStatusFailed, err.Error())
	}

	target, err := parseRecipient(attempt.Recipient)
	if err != nil {
		return w.finishAttempt(ctx, queries, attempt, notificationStatusFailed, err.Error())
	}

	sendErr := channel.Send(ctx, channels.SendParams{
		Target: target,
		Alert: channels.Alert{
			ID:             alert.ID.String(),
			OrganizationID: alert.OrganizationID.String(),
			ServiceID:      alert.ServiceID.String(),
			Summary:        alert.Summary,
			Description:    alert.Description.String,
			Priority:       alert.Priority,
			Status:         alert.Status,
		},
	})
	if sendErr != nil {
		w.logger.Warn(
			"notification delivery failed",
			"notification_attempt_id", attempt.ID,
			"channel", attempt.Channel,
			"alert_id", attempt.AlertID,
			"error", sendErr.Error(),
		)
		return w.finishAttempt(ctx, queries, attempt, notificationStatusFailed, sendErr.Error())
	}

	w.logger.Info(
		"notification delivered",
		"notification_attempt_id", attempt.ID,
		"channel", attempt.Channel,
		"alert_id", attempt.AlertID,
	)
	return w.finishAttempt(ctx, queries, attempt, notificationStatusSent, "")
}

func (w *NotifyWorker) finishAttempt(
	ctx context.Context,
	queries *db.Queries,
	attempt db.NotificationAttempt,
	status string,
	errorMessage string,
) error {
	var errMsg pgtype.Text
	if strings.TrimSpace(errorMessage) != "" {
		errMsg = pgtype.Text{String: errorMessage, Valid: true}
	}

	if _, err := queries.FinishNotificationAttempt(ctx, db.FinishNotificationAttemptParams{
		ID:             attempt.ID,
		OrganizationID: attempt.OrganizationID,
		Status:         status,
		ErrorMessage:   errMsg,
	}); err != nil {
		return fmt.Errorf("update notification attempt: %w", err)
	}
	return nil
}

func parseRecipient(raw []byte) (channels.Target, error) {
	var recipient struct {
		Type   string `json:"type"`
		UserID string `json:"user_id"`
		Email  string `json:"email"`
	}
	if err := json.Unmarshal(raw, &recipient); err != nil {
		return channels.Target{}, fmt.Errorf("parse recipient: %w", err)
	}
	return channels.Target{
		Type:   recipient.Type,
		UserID: recipient.UserID,
		Email:  recipient.Email,
	}, nil
}
