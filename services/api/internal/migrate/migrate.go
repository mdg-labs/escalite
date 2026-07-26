package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/mdg-labs/escalite/services/api/migrations"
	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// advisoryLockKey serializes concurrent migration runners on a single Postgres instance.
const advisoryLockKey int64 = 738573851

// Up applies embedded goose migrations guarded by a Postgres advisory lock.
func Up(ctx context.Context, databaseURL string, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	logger.Info("acquiring migration advisory lock")
	if err := withAdvisoryLock(ctx, db, advisoryLockKey, func() error {
		logger.Info("running database migrations")
		return applyGooseMigrations(ctx, db)
	}); err != nil {
		return err
	}

	logger.Info("database migrations complete")
	return nil
}

func applyGooseMigrations(ctx context.Context, db *sql.DB) error {
	goose.SetBaseFS(migrations.Files)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	if err := goose.UpContext(ctx, db, "."); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

func withAdvisoryLock(ctx context.Context, db *sql.DB, key int64, fn func() error) error {
	if _, err := db.ExecContext(ctx, "SELECT pg_advisory_lock($1)", key); err != nil {
		return fmt.Errorf("acquire advisory lock: %w", err)
	}

	defer func() {
		if _, err := db.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", key); err != nil {
			slog.Default().Warn("release advisory lock failed", "error", err)
		}
	}()

	return fn()
}
