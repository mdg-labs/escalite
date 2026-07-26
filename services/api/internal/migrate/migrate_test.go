package migrate

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/mdg-labs/escalite/services/api/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpRejectsUnreachableDatabase(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := Up(ctx, "postgres://invalid:invalid@127.0.0.1:1/nope?sslmode=disable", slog.Default())
	require.Error(t, err)
}

func TestUpAppliesMigrationsIdempotently(t *testing.T) {
	databaseURL, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()
	logger := slog.Default()

	require.NoError(t, Up(ctx, databaseURL, logger))
	require.NoError(t, Up(ctx, databaseURL, logger))
}

func TestConcurrentUpUsesAdvisoryLock(t *testing.T) {
	databaseURL, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()
	logger := slog.Default()

	var wg sync.WaitGroup
	errs := make(chan error, 2)

	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- Up(ctx, databaseURL, logger)
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		assert.NoError(t, err)
	}
}
