package testutil

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/mdg-labs/escalite/services/dbmigrate"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
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

// MigrateUp applies API migrations from the sibling services/api module.
func MigrateUp(ctx context.Context, databaseURL string, logger *slog.Logger) error {
	migrationsDir, err := apiMigrationsDir()
	if err != nil {
		return err
	}

	return dbmigrate.Up(ctx, databaseURL, logger, os.DirFS(migrationsDir), ".")
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
