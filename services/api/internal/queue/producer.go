package queue

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/riverqueue/river/rivertype"
)

const riverAdvisoryLockKey int64 = 738573852

// Producer inserts and cancels River jobs without running workers.
type Producer struct {
	pool  *pgxpool.Pool
	river *river.Client[pgx.Tx]
}

// NewProducer prepares a River client for job enqueue from the API service.
func NewProducer(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) (*Producer, error) {
	if logger == nil {
		logger = slog.Default()
	}

	if err := migrateUp(ctx, pool, logger); err != nil {
		return nil, fmt.Errorf("river migrations: %w", err)
	}

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{})
	if err != nil {
		return nil, fmt.Errorf("create river client: %w", err)
	}

	return &Producer{
		pool:  pool,
		river: riverClient,
	}, nil
}

// Insert enqueues a River job.
func (p *Producer) Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	return p.river.Insert(ctx, args, opts)
}

// CancelJob cancels a scheduled River job by ID.
func (p *Producer) CancelJob(ctx context.Context, jobID int64) error {
	_, err := p.river.JobCancel(ctx, jobID)
	if err != nil {
		return fmt.Errorf("cancel river job: %w", err)
	}
	return nil
}

func migrateUp(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire database connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", riverAdvisoryLockKey); err != nil {
		return fmt.Errorf("acquire river advisory lock: %w", err)
	}
	defer func() {
		if _, unlockErr := conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", riverAdvisoryLockKey); unlockErr != nil {
			logger.Warn("release river advisory lock failed", "error", unlockErr)
		}
	}()

	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return fmt.Errorf("create river migrator: %w", err)
	}

	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		return fmt.Errorf("apply river migrations: %w", err)
	}

	return nil
}
