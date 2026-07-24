package queue

import (
	"log/slog"

	"github.com/riverqueue/river"
)

// NewWorkers registers engine workers on a River workers bundle.
func NewWorkers(logger *slog.Logger) *river.Workers {
	workers := river.NewWorkers()
	river.AddWorker(workers, NewHeartbeatScanWorker(logger))
	return workers
}
