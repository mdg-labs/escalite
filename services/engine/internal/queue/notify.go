package queue

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/mdg-labs/escalite/services/engine/internal/db"
	"github.com/mdg-labs/escalite/services/engine/internal/jobs"
)

// NotifyWorker processes outbound notification jobs. Delivery channels land in later tasks.
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

	w.logger.Info(
		"notify job received",
		"notification_attempt_id", attempt.ID,
		"channel", attempt.Channel,
		"alert_id", attempt.AlertID,
	)
	return nil
}
