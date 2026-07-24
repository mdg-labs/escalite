package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"ariga.io/atlas/atlasexec"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// StartPostgres boots a disposable Postgres instance for integration tests.
func StartPostgres(ctx context.Context) (string, func(), error) {
	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("escalite"),
		postgres.WithUsername("escalite"),
		postgres.WithPassword("escalite"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return "", nil, err
	}

	databaseURL, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		return "", nil, err
	}

	cleanup := func() {
		_ = testcontainers.TerminateContainer(container)
	}

	return databaseURL, cleanup, nil
}

const advisoryLockKey int64 = 738573851

// MigrateUp applies API migrations from the sibling services/api module.
func MigrateUp(ctx context.Context, databaseURL string, logger *slog.Logger) error {
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

	if _, err := db.ExecContext(ctx, "SELECT pg_advisory_lock($1)", advisoryLockKey); err != nil {
		return fmt.Errorf("acquire advisory lock: %w", err)
	}
	defer func() {
		_, _ = db.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", advisoryLockKey)
	}()

	migrationsDir, err := apiMigrationsDir()
	if err != nil {
		return err
	}

	workdir, err := atlasexec.NewWorkingDir(
		atlasexec.WithMigrations(os.DirFS(migrationsDir)),
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

	if _, err := client.MigrateApply(ctx, &atlasexec.MigrateApplyParams{URL: databaseURL}); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	logger.Info("database migrations complete")
	return nil
}

func apiMigrationsDir() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("resolve testutil path")
	}
	dir := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "api", "migrations"))
	if _, err := os.Stat(dir); err != nil {
		return "", fmt.Errorf("locate api migrations: %w", err)
	}
	return dir, nil
}

func resolveAtlasBin() (string, error) {
	if path := os.Getenv("ESCALITE_ATLAS_BIN"); path != "" {
		return path, nil
	}
	if path, err := exec.LookPath("atlas"); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("atlas CLI not found: install atlas or set ESCALITE_ATLAS_BIN")
}
