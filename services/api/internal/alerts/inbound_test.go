package alerts_test

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

	"github.com/mdg-labs/escalite/services/api/internal/alerts"
	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/migrate"
	"github.com/mdg-labs/escalite/services/integrations"
)

func startPostgres(t *testing.T) (*pgxpool.Pool, func()) {
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
	require.NoError(t, migrate.Up(ctx, databaseURL, slog.Default()))

	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)

	cleanup := func() {
		pool.Close()
		require.NoError(t, testcontainers.TerminateContainer(container))
	}

	return pool, cleanup
}

type inboundFixture struct {
	queries *db.Queries
	key     db.IntegrationKey
	userID  uuid.UUID
}

func seedInboundFixture(t *testing.T, pool *pgxpool.Pool) inboundFixture {
	t.Helper()

	ctx := context.Background()
	queries := db.New(pool)

	orgID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())
	_, err := queries.BootstrapOrganizationWithAdmin(ctx, db.BootstrapOrganizationWithAdminParams{
		OrgID:        orgID,
		OrgName:      "Acme",
		UserID:       userID,
		Email:        "admin@example.com",
		PasswordHash: pgtype.Text{String: "argon2id:test", Valid: true},
	})
	require.NoError(t, err)

	team, err := queries.CreateTeam(ctx, db.CreateTeamParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		Name:           "Platform",
	})
	require.NoError(t, err)

	service, err := queries.CreateService(ctx, db.CreateServiceParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		TeamID:         team.ID,
		Name:           "checkout-api",
	})
	require.NoError(t, err)

	plaintext, hash, prefix, err := auth.NewIntegrationKeyToken()
	require.NoError(t, err)

	keyID := uuid.Must(uuid.NewV7())
	_, err = pool.Exec(ctx, `
		INSERT INTO integration_keys (
			id, service_id, organization_id, token, prefix, plugin_name, config
		) VALUES ($1, $2, $3, $4, $5, $6, '{}'::jsonb)
	`, keyID, service.ID, orgID, hash, prefix, "generic-rest-api")
	require.NoError(t, err)

	key, err := queries.GetActiveIntegrationKeyByTokenHash(ctx, hash)
	require.NoError(t, err)
	require.Equal(t, plaintext[:len(prefix)], key.Prefix)

	return inboundFixture{
		queries: queries,
		key:     key,
		userID:  userID,
	}
}

func TestProcessInboundCollapseIncrementsEventCount(t *testing.T) {
	pool, cleanup := startPostgres(t)
	defer cleanup()

	fixture := seedInboundFixture(t, pool)
	ctx := context.Background()
	logger := slog.Default()

	event := integrations.AlertCreate{
		EventType:   integrations.EventTriggered,
		DedupKey:    "host-1-disk",
		Summary:     "Disk usage high",
		Description: "Volume /data is 95% full",
		Priority:    "low",
	}

	firstID, err := alerts.ProcessInbound(ctx, fixture.queries, logger, fixture.key, event)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, firstID)

	secondID, err := alerts.ProcessInbound(ctx, fixture.queries, logger, fixture.key, event)
	require.NoError(t, err)
	require.Equal(t, firstID, secondID)

	alert, err := fixture.queries.GetAlertByID(ctx, db.GetAlertByIDParams{
		ID:             firstID,
		OrganizationID: fixture.key.OrganizationID,
	})
	require.NoError(t, err)
	require.Equal(t, int32(2), alert.EventCount)
	require.Equal(t, "triggered", alert.Status)

	var alertCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM alerts
		WHERE service_id = $1 AND dedup_key = $2
	`, fixture.key.ServiceID, event.DedupKey).Scan(&alertCount)
	require.NoError(t, err)
	require.Equal(t, 1, alertCount)
}

func TestProcessInboundCollapseAcknowledgedAlertDoesNotCreateNotifications(t *testing.T) {
	pool, cleanup := startPostgres(t)
	defer cleanup()

	fixture := seedInboundFixture(t, pool)
	ctx := context.Background()
	logger := slog.Default()

	event := integrations.AlertCreate{
		EventType: integrations.EventTriggered,
		DedupKey:  "memory-high",
		Summary:   "Memory above threshold",
		Priority:  "high",
	}

	alertID, err := alerts.ProcessInbound(ctx, fixture.queries, logger, fixture.key, event)
	require.NoError(t, err)

	_, err = fixture.queries.AcknowledgeAlert(ctx, db.AcknowledgeAlertParams{
		ID:                   alertID,
		OrganizationID:       fixture.key.OrganizationID,
		EscalationState:      []byte(`{"current_step":1}`),
		AcknowledgedByUserID: pgtype.UUID{Bytes: fixture.userID, Valid: true},
	})
	require.NoError(t, err)

	_, err = fixture.queries.CreateNotificationAttempt(ctx, db.CreateNotificationAttemptParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: fixture.key.OrganizationID,
		AlertID:        alertID,
		Channel:        "email",
		Status:         "sent",
		Recipient:      []byte(`{"type":"user","user_id":"` + fixture.userID.String() + `"}`),
	})
	require.NoError(t, err)

	beforeCount, err := fixture.queries.CountNotificationAttemptsByAlertID(ctx, db.CountNotificationAttemptsByAlertIDParams{
		AlertID:        alertID,
		OrganizationID: fixture.key.OrganizationID,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), beforeCount)

	collapsedID, err := alerts.ProcessInbound(ctx, fixture.queries, logger, fixture.key, event)
	require.NoError(t, err)
	require.Equal(t, alertID, collapsedID)

	afterCount, err := fixture.queries.CountNotificationAttemptsByAlertID(ctx, db.CountNotificationAttemptsByAlertIDParams{
		AlertID:        alertID,
		OrganizationID: fixture.key.OrganizationID,
	})
	require.NoError(t, err)
	require.Equal(t, beforeCount, afterCount)

	alert, err := fixture.queries.GetAlertByID(ctx, db.GetAlertByIDParams{
		ID:             alertID,
		OrganizationID: fixture.key.OrganizationID,
	})
	require.NoError(t, err)
	require.Equal(t, "acknowledged", alert.Status)
	require.Equal(t, int32(2), alert.EventCount)
}
