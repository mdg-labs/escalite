package dbmigrate

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// AdvisoryLockKey serializes concurrent migration runners on a single Postgres instance.
const AdvisoryLockKey int64 = 738573851

// Up applies goose migrations from migrations rooted at root within migrations.
// When migrations is non-nil, files are read from the fs.FS (e.g. embed.FS).
// When migrations is nil, root is a filesystem path to the migrations directory.
func Up(ctx context.Context, databaseURL string, logger *slog.Logger, migrations fs.FS, root string) error {
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
	if err := withAdvisoryLock(ctx, db, AdvisoryLockKey, func() error {
		logger.Info("running database migrations")
		return applyGooseMigrations(ctx, db, migrations, root)
	}); err != nil {
		return err
	}

	logger.Info("database migrations complete")
	return nil
}

func applyGooseMigrations(ctx context.Context, db *sql.DB, migrations fs.FS, root string) error {
	goose.SetBaseFS(migrations)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	if err := goose.UpContext(ctx, db, root); err != nil {
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
