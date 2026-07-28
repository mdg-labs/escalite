package statuspage_test

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/engine/internal/testutil"
)

func startPostgres(ctx context.Context) (string, func(), error) {
	return testutil.StartPostgres(ctx)
}

func openPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	return pgxpool.New(ctx, databaseURL)
}

func migrateUp(ctx context.Context, databaseURL string) error {
	return testutil.MigrateUp(ctx, databaseURL, slog.Default())
}
