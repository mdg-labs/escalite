package queue

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/mdg-labs/escalite/services/engine/internal/escalation"
	"github.com/mdg-labs/escalite/services/engine/internal/heartbeat"
)

const heartbeatScanKind = "heartbeat_scan"

// HeartbeatScanArgs is the periodic job that scans for overdue heartbeat monitors.
type HeartbeatScanArgs struct{}

func (HeartbeatScanArgs) Kind() string { return heartbeatScanKind }

// HeartbeatScanWorker scans heartbeat monitors and triggers alerts when deadlines pass.
type HeartbeatScanWorker struct {
	river.WorkerDefaults[HeartbeatScanArgs]
	logger   *slog.Logger
	pool     *pgxpool.Pool
	inserter escalation.JobInserter
}

func NewHeartbeatScanWorker(
	logger *slog.Logger,
	pool *pgxpool.Pool,
	inserter escalation.JobInserter,
) *HeartbeatScanWorker {
	if logger == nil {
		logger = slog.Default()
	}
	return &HeartbeatScanWorker{logger: logger, pool: pool, inserter: inserter}
}

func (w *HeartbeatScanWorker) Work(ctx context.Context, _ *river.Job[HeartbeatScanArgs]) error {
	return heartbeat.ScanOverdueMonitors(ctx, w.pool, w.inserter, w.logger)
}
