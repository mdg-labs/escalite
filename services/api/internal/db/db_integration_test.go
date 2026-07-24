package db_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/migrate"
)

func startPostgres(t *testing.T) (string, func()) {
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

	databaseURL, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	cleanup := func() {
		require.NoError(t, testcontainers.TerminateContainer(container))
	}

	return databaseURL, cleanup
}

func TestQueriesAgainstPostgres(t *testing.T) {
	databaseURL, cleanup := startPostgres(t)
	defer cleanup()

	ctx := context.Background()
	require.NoError(t, migrate.Up(ctx, databaseURL, slog.Default()))

	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	queries := db.New(pool)

	ok, err := queries.PingDatabase(ctx)
	require.NoError(t, err)
	require.Equal(t, int32(1), ok)

	ready, err := queries.CoreSchemaReady(ctx)
	require.NoError(t, err)
	require.True(t, ready)

	sessionsReady, err := queries.SessionsSchemaReady(ctx)
	require.NoError(t, err)
	require.True(t, sessionsReady)

	count, err := queries.CountOrganizations(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(0), count)

	orgID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())
	email := "admin@example.com"
	passwordHash := "argon2id:test"

	bootstrap, err := queries.BootstrapOrganizationWithAdmin(ctx, db.BootstrapOrganizationWithAdminParams{
		OrgID:        orgID,
		OrgName:      "Acme",
		UserID:       userID,
		Email:        email,
		PasswordHash: pgtype.Text{String: passwordHash, Valid: true},
	})
	require.NoError(t, err)
	require.Equal(t, orgID, bootstrap.OrganizationID)
	require.Equal(t, "Acme", bootstrap.OrganizationName)
	require.Equal(t, userID, bootstrap.UserID)
	require.Equal(t, email, bootstrap.UserEmail)
	require.Equal(t, "admin", bootstrap.UserRole)

	count, err = queries.CountOrganizations(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), count)

	byEmail, err := queries.GetUserByEmail(ctx, db.GetUserByEmailParams{
		OrganizationID: orgID,
		Email:          email,
	})
	require.NoError(t, err)
	require.Equal(t, userID, byEmail.ID)

	sessionID := uuid.Must(uuid.NewV7())
	expiresAt := time.Now().UTC().Add(24 * time.Hour)
	userAgent := "integration-test"

	session, err := queries.CreateSession(ctx, db.CreateSessionParams{
		ID:             sessionID,
		UserID:         userID,
		OrganizationID: orgID,
		ExpiresAt:      pgtype.Timestamptz{Time: expiresAt, Valid: true},
		UserAgent:      pgtype.Text{String: userAgent, Valid: true},
	})
	require.NoError(t, err)
	require.Equal(t, sessionID, session.ID)

	active, err := queries.GetSessionByID(ctx, db.GetSessionByIDParams{
		ID:             sessionID,
		OrganizationID: orgID,
	})
	require.NoError(t, err)
	require.Equal(t, userID, active.UserID)

	require.NoError(t, queries.RevokeSession(ctx, db.RevokeSessionParams{
		ID:             sessionID,
		OrganizationID: orgID,
	}))

	_, err = queries.GetSessionByID(ctx, db.GetSessionByIDParams{
		ID:             sessionID,
		OrganizationID: orgID,
	})
	require.Error(t, err)
}
