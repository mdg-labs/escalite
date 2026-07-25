package queue

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/mdg-labs/escalite/services/engine/internal/crypto"
	"github.com/mdg-labs/escalite/services/engine/internal/escalation"
)

// NewWorkers registers engine workers on a River workers bundle.
func NewWorkers(logger *slog.Logger, pool *pgxpool.Pool, inserter escalation.JobInserter, secrets *crypto.Box) *river.Workers {
	workers := river.NewWorkers()
	river.AddWorker(workers, NewHeartbeatScanWorker(logger, pool, inserter))
	river.AddWorker(workers, NewEscalationTriggerWorker(logger, pool, inserter))
	river.AddWorker(workers, NewEscalationStepWorker(logger, pool, inserter))
	river.AddWorker(workers, NewNotifyWorker(logger, pool, secrets))
	return workers
}
