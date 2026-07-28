package db_test

import (
	"context"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"log/slog"
	"testing"
	"time"

	"github.com/allure-framework/allure-go/testify/require"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/migrate"
	"github.com/mdg-labs/escalite/services/api/internal/testutil"
)

func TestQueriesAgainstPostgres(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		databaseURL, cleanup := testutil.StartPostgres(t)
		defer cleanup()

		ctx := context.Background()
		require.NoError(a, migrate.Up(ctx, databaseURL, slog.Default()))

		pool, err := pgxpool.New(ctx, databaseURL)
		require.NoError(a, err)
		t.Cleanup(pool.Close)

		queries := db.New(pool)

		ok, err := queries.PingDatabase(ctx)
		require.NoError(a, err)
		require.Equal(a, int32(1), ok)

		ready, err := queries.CoreSchemaReady(ctx)
		require.NoError(a, err)
		require.True(a, ready)

		sessionsReady, err := queries.SessionsSchemaReady(ctx)
		require.NoError(a, err)
		require.True(a, sessionsReady)

		count, err := queries.CountOrganizations(ctx)
		require.NoError(a, err)
		require.Equal(a, int64(0), count)

		orgID := uuid.Must(uuid.NewV7())
		accountID := uuid.Must(uuid.NewV7())
		userID := uuid.Must(uuid.NewV7())
		email := "admin@example.com"
		passwordHash := "argon2id:test"

		bootstrap, err := queries.BootstrapOrganizationWithAdmin(ctx, db.BootstrapOrganizationWithAdminParams{
			OrgID:        orgID,
			OrgName:      "Acme",
			AccountID:    accountID,
			UserID:       userID,
			Email:        email,
			PasswordHash: pgtype.Text{String: passwordHash, Valid: true},
		})
		require.NoError(a, err)
		require.Equal(a, orgID, bootstrap.OrganizationID)
		require.Equal(a, "Acme", bootstrap.OrganizationName)
		require.Equal(a, userID, bootstrap.UserID)
		require.Equal(a, email, bootstrap.UserEmail)
		require.Equal(a, "admin", bootstrap.UserRole)

		count, err = queries.CountOrganizations(ctx)
		require.NoError(a, err)
		require.Equal(a, int64(1), count)

		byEmail, err := queries.GetUserByEmail(ctx, db.GetUserByEmailParams{
			OrganizationID: orgID,
			Email:          email,
		})
		require.NoError(a, err)
		require.Equal(a, userID, byEmail.ID)

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
		require.NoError(a, err)
		require.Equal(a, sessionID, session.ID)

		active, err := queries.GetSessionByID(ctx, db.GetSessionByIDParams{
			ID:             sessionID,
			OrganizationID: orgID,
		})
		require.NoError(a, err)
		require.Equal(a, userID, active.UserID)

		require.NoError(a, queries.RevokeSession(ctx, db.RevokeSessionParams{
			ID:             sessionID,
			OrganizationID: orgID,
		}))

		_, err = queries.GetSessionByID(ctx, db.GetSessionByIDParams{
			ID:             sessionID,
			OrganizationID: orgID,
		})
		require.Error(a, err)
	})
}
