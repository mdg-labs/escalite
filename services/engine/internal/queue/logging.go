package queue

import (
	"context"
	"log/slog"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// NewWorkerLoggingMiddleware logs structured job lifecycle events for every worker.
func NewWorkerLoggingMiddleware(logger *slog.Logger) river.WorkerMiddlewareFunc {
	if logger == nil {
		logger = slog.Default()
	}

	return func(ctx context.Context, job *rivertype.JobRow, doInner func(ctx context.Context) error) error {
		jobLogger := logger.With(
			"job_id", job.ID,
			"job_kind", job.Kind,
			"queue", job.Queue,
			"attempt", job.Attempt,
		)

		jobLogger.Info("job started")
		startedAt := time.Now()

		err := doInner(ctx)
		durationMS := time.Since(startedAt).Milliseconds()
		if err != nil {
			jobLogger.Error("job finished", "duration_ms", durationMS, "error", err.Error())
			return err
		}

		jobLogger.Info("job finished", "duration_ms", durationMS)
		return nil
	}
}
