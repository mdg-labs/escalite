package queue

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/mdg-labs/escalite/services/engine/internal/db"
	"github.com/mdg-labs/escalite/services/engine/internal/escalation"
	"github.com/mdg-labs/escalite/services/engine/internal/jobs"
)

// EscalationStepWorker advances escalation when a step delay elapses.
type EscalationStepWorker struct {
	river.WorkerDefaults[jobs.EscalationStepArgs]
	logger   *slog.Logger
	pool     *pgxpool.Pool
	inserter escalation.JobInserter
}

func NewEscalationStepWorker(logger *slog.Logger, pool *pgxpool.Pool, inserter escalation.JobInserter) *EscalationStepWorker {
	if logger == nil {
		logger = slog.Default()
	}
	return &EscalationStepWorker{logger: logger, pool: pool, inserter: inserter}
}

func (w *EscalationStepWorker) Work(ctx context.Context, job *river.Job[jobs.EscalationStepArgs]) error {
	queries := db.New(w.pool)
	if err := escalation.AdvanceEscalationStep(
		ctx,
		queries,
		w.inserter,
		job.Args.AlertID,
		job.Args.OrganizationID,
	); err != nil {
		return fmt.Errorf("advance escalation step: %w", err)
	}
	w.logger.Info(
		"escalation step advanced",
		"alert_id", job.Args.AlertID,
		"organization_id", job.Args.OrganizationID,
	)
	return nil
}
