package queue

import (
	"context"
	"log/slog"

	"github.com/riverqueue/river"
)

const heartbeatScanKind = "heartbeat_scan"

// HeartbeatScanArgs is a periodic noop job that validates end-to-end River wiring.
type HeartbeatScanArgs struct{}

func (HeartbeatScanArgs) Kind() string { return heartbeatScanKind }

// HeartbeatScanWorker logs and exits. Real overdue-heartbeat scanning lands in Phase 1.
type HeartbeatScanWorker struct {
	river.WorkerDefaults[HeartbeatScanArgs]
	logger *slog.Logger
}

func NewHeartbeatScanWorker(logger *slog.Logger) *HeartbeatScanWorker {
	if logger == nil {
		logger = slog.Default()
	}
	return &HeartbeatScanWorker{logger: logger}
}

func (w *HeartbeatScanWorker) Work(_ context.Context, _ *river.Job[HeartbeatScanArgs]) error {
	w.logger.Info("heartbeat scan completed")
	return nil
}
