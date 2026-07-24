package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"ariga.io/atlas/atlasexec"
	"github.com/mdg-labs/escalite/services/api/migrations"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// advisoryLockKey serializes concurrent migration runners on a single Postgres instance.
const advisoryLockKey int64 = 738573851

// Up applies embedded Atlas migrations guarded by a Postgres advisory lock.
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
		return applyAtlasMigrations(ctx, databaseURL)
	}); err != nil {
		return err
	}

	logger.Info("database migrations complete")
	return nil
}

func applyAtlasMigrations(ctx context.Context, databaseURL string) error {
	workdir, err := atlasexec.NewWorkingDir(
		atlasexec.WithMigrations(migrations.Files),
	)
	if err != nil {
		return fmt.Errorf("prepare migration working dir: %w", err)
	}
	defer workdir.Close()

	atlasBin, err := resolveAtlasBin()
	if err != nil {
		return err
	}

	client, err := atlasexec.NewClient(workdir.Path(), atlasBin)
	if err != nil {
		return fmt.Errorf("create atlas client: %w", err)
	}

	_, err = client.MigrateApply(ctx, &atlasexec.MigrateApplyParams{
		URL: databaseURL,
	})
	if err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}

func resolveAtlasBin() (string, error) {
	if path := os.Getenv("ESCALITE_ATLAS_BIN"); path != "" {
		return path, nil
	}
	if path, err := exec.LookPath("atlas"); err == nil {
		return path, nil
	}
	if _, err := os.Stat("/app/atlas"); err == nil {
		return "/app/atlas", nil
	}
	return "", fmt.Errorf("atlas CLI not found: install atlas or set ESCALITE_ATLAS_BIN")
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
