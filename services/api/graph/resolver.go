package graph

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/audit"
	"github.com/mdg-labs/escalite/services/api/internal/queue"
)

// Resolver is the root GraphQL resolver with shared dependencies.
type Resolver struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
	audit  *audit.Recorder
	jobs   *queue.Producer
}

// NewResolver returns a resolver wired with database and logging dependencies.
func NewResolver(pool *pgxpool.Pool, logger *slog.Logger, jobs *queue.Producer) *Resolver {
	return &Resolver{
		pool:   pool,
		logger: logger,
		audit:  audit.NewRecorder(logger),
		jobs:   jobs,
	}
}
