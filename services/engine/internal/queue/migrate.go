package queue

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

// riverAdvisoryLockKey serializes concurrent River migration runners.
const riverAdvisoryLockKey int64 = 738573852

// MigrateUp applies River schema migrations guarded by a Postgres advisory lock.
func MigrateUp(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire database connection: %w", err)
	}
	defer conn.Release()

	logger.Info("acquiring river migration advisory lock")
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

	logger.Info("running river migrations")
	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		return fmt.Errorf("apply river migrations: %w", err)
	}

	logger.Info("river migrations complete")
	return nil
}
