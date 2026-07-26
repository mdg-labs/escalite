package migrate

import (
	"context"
	"log/slog"

	"github.com/mdg-labs/escalite/services/api/migrations"
	"github.com/mdg-labs/escalite/services/dbmigrate"
)

// Up applies embedded goose migrations guarded by a Postgres advisory lock.
func Up(ctx context.Context, databaseURL string, logger *slog.Logger) error {
	return dbmigrate.Up(ctx, databaseURL, logger, migrations.Files, ".")
}
