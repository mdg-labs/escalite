package queue

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	emailchannel "github.com/mdg-labs/escalite/services/engine/channels/email"
	"github.com/mdg-labs/escalite/services/engine/internal/db"
	"github.com/mdg-labs/escalite/services/engine/internal/jobs"
	"github.com/mdg-labs/escalite/services/engine/internal/statuspage"
)

// StatusPageIncidentNotifyWorker emails status page subscribers about incident updates.
type StatusPageIncidentNotifyWorker struct {
	river.WorkerDefaults[jobs.StatusPageIncidentNotifyArgs]
	logger *slog.Logger
	pool   *pgxpool.Pool
	cfg    statuspage.NotifyConfig
}

func NewStatusPageIncidentNotifyWorker(
	logger *slog.Logger,
	pool *pgxpool.Pool,
	cfg statuspage.NotifyConfig,
) *StatusPageIncidentNotifyWorker {
	if logger == nil {
		logger = slog.Default()
	}
	return &StatusPageIncidentNotifyWorker{logger: logger, pool: pool, cfg: cfg}
}

func (w *StatusPageIncidentNotifyWorker) Work(ctx context.Context, job *river.Job[jobs.StatusPageIncidentNotifyArgs]) error {
	sender := emailchannel.Sender()
	if sender == nil {
		w.logger.Info(
			"skipping status page subscriber email; smtp not configured",
			"status_page_incident_id", job.Args.StatusPageIncidentID,
			"organization_id", job.Args.OrganizationID,
		)
		return nil
	}

	queries := db.New(w.pool)
	if err := statuspage.NotifySubscribers(ctx, queries, sender, w.cfg, statuspage.NotifyParams{
		OrganizationID:       job.Args.OrganizationID,
		StatusPageIncidentID: job.Args.StatusPageIncidentID,
		UpdateID:             job.Args.UpdateID,
	}); err != nil {
		return fmt.Errorf("notify status page subscribers: %w", err)
	}

	w.logger.Info(
		"status page subscriber emails sent",
		"status_page_incident_id", job.Args.StatusPageIncidentID,
		"update_id", job.Args.UpdateID,
		"organization_id", job.Args.OrganizationID,
	)
	return nil
}
