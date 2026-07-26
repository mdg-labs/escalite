package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// StartPostgres boots a disposable Postgres 16 container for integration tests.
func StartPostgres(t *testing.T) (databaseURL string, cleanup func()) {
	t.Helper()

	ctx := context.Background()

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
	require.NoError(t, err)

	databaseURL, err = container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	cleanup = func() {
		require.NoError(t, testcontainers.TerminateContainer(container))
	}

	return databaseURL, cleanup
}
