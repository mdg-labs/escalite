package realtime_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/migrate"
	"github.com/mdg-labs/escalite/services/api/internal/realtime"
)

func startRealtimePostgres(t *testing.T) (string, func()) {
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

func TestAlertStatusChangeEmitsNotify(t *testing.T) {
	databaseURL, cleanup := startRealtimePostgres(t)
	defer cleanup()

	ctx := context.Background()
	require.NoError(t, migrate.Up(ctx, databaseURL, nil))

	bridge := realtime.NewBridge(databaseURL, nil)
	bridge.Start(ctx)
	defer bridge.Close()

	orgID := uuid.Must(uuid.NewV7())
	teamID := uuid.Must(uuid.NewV7())
	serviceID := uuid.Must(uuid.NewV7())
	alertID := uuid.Must(uuid.NewV7())

	conn, err := pgx.Connect(ctx, databaseURL)
	require.NoError(t, err)
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, `INSERT INTO organizations (id, name) VALUES ($1, 'Acme')`, orgID)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, `INSERT INTO teams (id, organization_id, name) VALUES ($1, $2, 'Platform')`, teamID, orgID)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, `INSERT INTO services (id, organization_id, team_id, name) VALUES ($1, $2, $3, 'api')`, serviceID, orgID, teamID)
	require.NoError(t, err)

	events, cancel := bridge.Hub.SubscribeAlerts(orgID)
	defer cancel()

	_, err = conn.Exec(ctx, `
		INSERT INTO alerts (
			id, organization_id, service_id, status, dedup_key, summary, priority, escalation_state
		) VALUES ($1, $2, $3, 'triggered', 'cpu-high', 'CPU high', 'high', '{}'::jsonb)
	`, alertID, orgID, serviceID)
	require.NoError(t, err)

	select {
	case evt := <-events:
		require.Equal(t, alertID, evt.AlertID)
		require.Equal(t, orgID, evt.OrganizationID)
		require.Equal(t, "triggered", evt.Status)
		require.Equal(t, "INSERT", evt.Op)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for insert notify")
	}

	_, err = conn.Exec(ctx, `
		UPDATE alerts SET status = 'acknowledged', acknowledged_at = now()
		WHERE id = $1 AND organization_id = $2
	`, alertID, orgID)
	require.NoError(t, err)

	select {
	case evt := <-events:
		require.Equal(t, alertID, evt.AlertID)
		require.Equal(t, "acknowledged", evt.Status)
		require.Equal(t, "UPDATE", evt.Op)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for status update notify")
	}
}

func TestAcknowledgeAlertMutationEmitsNotify(t *testing.T) {
	databaseURL, cleanup := startRealtimePostgres(t)
	defer cleanup()

	ctx := context.Background()
	require.NoError(t, migrate.Up(ctx, databaseURL, nil))

	bridge := realtime.NewBridge(databaseURL, nil)
	bridge.Start(ctx)
	defer bridge.Close()

	orgID := uuid.Must(uuid.NewV7())
	teamID := uuid.Must(uuid.NewV7())
	serviceID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())
	alertID := uuid.Must(uuid.NewV7())

	conn, err := pgx.Connect(ctx, databaseURL)
	require.NoError(t, err)
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, `INSERT INTO organizations (id, name) VALUES ($1, 'Acme')`, orgID)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, `INSERT INTO teams (id, organization_id, name) VALUES ($1, $2, 'Platform')`, teamID, orgID)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, `INSERT INTO services (id, organization_id, team_id, name) VALUES ($1, $2, $3, 'api')`, serviceID, orgID, teamID)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, `INSERT INTO users (id, organization_id, email, role, password_hash) VALUES ($1, $2, 'admin@example.com', 'admin', 'hash')`, userID, orgID)
	require.NoError(t, err)

	events, cancel := bridge.Hub.SubscribeAlerts(orgID)
	defer cancel()

	queries := db.New(conn)
	_, err = queries.CreateTriggeredAlert(ctx, db.CreateTriggeredAlertParams{
		ID:              alertID,
		OrganizationID:  orgID,
		ServiceID:       serviceID,
		DedupKey:        "disk-full",
		Summary:         "Disk full",
		Priority:        "high",
		EscalationState: []byte(`{}`),
	})
	require.NoError(t, err)

	select {
	case evt := <-events:
		require.Equal(t, "INSERT", evt.Op)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for insert notify")
	}

	_, err = queries.AcknowledgeAlert(ctx, db.AcknowledgeAlertParams{
		ID:                   alertID,
		OrganizationID:       orgID,
		EscalationState:      []byte(`{}`),
		AcknowledgedByUserID: pgtype.UUID{Bytes: userID, Valid: true},
	})
	require.NoError(t, err)

	select {
	case evt := <-events:
		require.Equal(t, alertID, evt.AlertID)
		require.Equal(t, "acknowledged", evt.Status)
		require.Equal(t, "UPDATE", evt.Op)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for acknowledge notify")
	}
}
