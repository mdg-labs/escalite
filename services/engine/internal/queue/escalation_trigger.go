package queue

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/mdg-labs/escalite/services/engine/internal/db"
	"github.com/mdg-labs/escalite/services/engine/internal/escalation"
	"github.com/mdg-labs/escalite/services/engine/internal/jobs"
)

// EscalationTriggerWorker handles alert trigger escalation scheduling.
type EscalationTriggerWorker struct {
	river.WorkerDefaults[jobs.EscalationTriggerArgs]
	logger   *slog.Logger
	pool     *pgxpool.Pool
	inserter escalation.JobInserter
}

func NewEscalationTriggerWorker(logger *slog.Logger, pool *pgxpool.Pool, inserter escalation.JobInserter) *EscalationTriggerWorker {
	if logger == nil {
		logger = slog.Default()
	}
	return &EscalationTriggerWorker{logger: logger, pool: pool, inserter: inserter}
}

func (w *EscalationTriggerWorker) Work(ctx context.Context, job *river.Job[jobs.EscalationTriggerArgs]) error {
	queries := db.New(w.pool)
	if err := escalation.ScheduleStep1Notifications(
		ctx,
		queries,
		w.inserter,
		job.Args.AlertID,
		job.Args.OrganizationID,
	); err != nil {
		return fmt.Errorf("schedule step-1 notifications: %w", err)
	}
	w.logger.Info(
		"step-1 notifications scheduled",
		"alert_id", job.Args.AlertID,
		"organization_id", job.Args.OrganizationID,
	)
	return nil
}

// Insert implements escalation.JobInserter using the River client.
func (c *Client) Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	return c.river.Insert(ctx, args, opts)
}

// CancelJob implements escalation.JobCanceller.
func (c *Client) CancelJob(ctx context.Context, jobID int64) error {
	_, err := c.river.JobCancel(ctx, jobID)
	if err != nil {
		return fmt.Errorf("cancel river job: %w", err)
	}
	return nil
}
