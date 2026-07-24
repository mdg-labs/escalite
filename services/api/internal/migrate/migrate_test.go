package migrate

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

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
	databaseURL := os.Getenv("ESCALITE_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("ESCALITE_DATABASE_URL not set")
	}

	ctx := context.Background()
	logger := slog.Default()

	require.NoError(t, Up(ctx, databaseURL, logger))
	require.NoError(t, Up(ctx, databaseURL, logger))
}

func TestConcurrentUpUsesAdvisoryLock(t *testing.T) {
	databaseURL := os.Getenv("ESCALITE_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("ESCALITE_DATABASE_URL not set")
	}

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
