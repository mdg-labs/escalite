package migrate

import (
	"context"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/allure-framework/allure-go/testify/assert"
	"github.com/allure-framework/allure-go/testify/require"
	"github.com/mdg-labs/escalite/services/api/internal/testutil"
)

func TestUpRejectsUnreachableDatabase(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := Up(ctx, "postgres://invalid:invalid@127.0.0.1:1/nope?sslmode=disable", slog.Default())
		require.Error(a, err)
	})
}

func TestUpAppliesMigrationsIdempotently(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		databaseURL, cleanup := testutil.StartPostgres(t)
		defer cleanup()

		ctx := context.Background()
		logger := slog.Default()

		require.NoError(a, Up(ctx, databaseURL, logger))
		require.NoError(a, Up(ctx, databaseURL, logger))
	})
}

func TestConcurrentUpUsesAdvisoryLock(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
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
			assert.NoError(a, err)
		}
	})
}
