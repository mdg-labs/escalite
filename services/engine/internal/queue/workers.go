package queue

import (
	"context"

	"github.com/riverqueue/river"
)

// NoopArgs is a placeholder job kind so the River client can start before
// domain jobs (for example heartbeat scan in p0-river-noop-heartbeat-job) register.
type NoopArgs struct{}

func (NoopArgs) Kind() string { return "noop" }

// NoopWorker completes immediately. It is not enqueued in normal operation.
type NoopWorker struct {
	river.WorkerDefaults[NoopArgs]
}

func (w *NoopWorker) Work(_ context.Context, _ *river.Job[NoopArgs]) error {
	return nil
}

// NewWorkers registers engine workers on a River workers bundle.
func NewWorkers() *river.Workers {
	workers := river.NewWorkers()
	river.AddWorker(workers, &NoopWorker{})
	return workers
}
