package realtime_test

import (
	"context"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"
	"time"

	"github.com/allure-framework/allure-go/testify/require"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/migrate"
	"github.com/mdg-labs/escalite/services/api/internal/realtime"
	"github.com/mdg-labs/escalite/services/api/internal/testutil"
)

func TestAlertStatusChangeEmitsNotify(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		databaseURL, cleanup := testutil.StartPostgres(t)
		defer cleanup()

		ctx := context.Background()
		require.NoError(a, migrate.Up(ctx, databaseURL, nil))

		bridge := realtime.NewBridge(databaseURL, nil)
		bridge.Start(ctx)
		defer bridge.Close()

		orgID := uuid.Must(uuid.NewV7())
		teamID := uuid.Must(uuid.NewV7())
		serviceID := uuid.Must(uuid.NewV7())
		alertID := uuid.Must(uuid.NewV7())

		conn, err := pgx.Connect(ctx, databaseURL)
		require.NoError(a, err)
		defer conn.Close(ctx)

		_, err = conn.Exec(ctx, `INSERT INTO organizations (id, name) VALUES ($1, 'Acme')`, orgID)
		require.NoError(a, err)
		_, err = conn.Exec(ctx, `INSERT INTO teams (id, organization_id, name) VALUES ($1, $2, 'Platform')`, teamID, orgID)
		require.NoError(a, err)
		_, err = conn.Exec(ctx, `INSERT INTO services (id, organization_id, team_id, name) VALUES ($1, $2, $3, 'api')`, serviceID, orgID, teamID)
		require.NoError(a, err)

		events, cancel := bridge.Hub.SubscribeAlerts(orgID)
		defer cancel()

		_, err = conn.Exec(ctx, `
		INSERT INTO alerts (
			id, organization_id, service_id, status, dedup_key, summary, priority, escalation_state
		) VALUES ($1, $2, $3, 'triggered', 'cpu-high', 'CPU high', 'high', '{}'::jsonb)
	`, alertID, orgID, serviceID)
		require.NoError(a, err)

		select {
		case evt := <-events:
			require.Equal(a, alertID, evt.AlertID)
			require.Equal(a, orgID, evt.OrganizationID)
			require.Equal(a, "triggered", evt.Status)
			require.Equal(a, "INSERT", evt.Op)
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for insert notify")
		}

		_, err = conn.Exec(ctx, `
		UPDATE alerts SET status = 'acknowledged', acknowledged_at = now()
		WHERE id = $1 AND organization_id = $2
	`, alertID, orgID)
		require.NoError(a, err)

		select {
		case evt := <-events:
			require.Equal(a, alertID, evt.AlertID)
			require.Equal(a, "acknowledged", evt.Status)
			require.Equal(a, "UPDATE", evt.Op)
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for status update notify")
		}
	})
}

func TestAcknowledgeAlertMutationEmitsNotify(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		databaseURL, cleanup := testutil.StartPostgres(t)
		defer cleanup()

		ctx := context.Background()
		require.NoError(a, migrate.Up(ctx, databaseURL, nil))

		bridge := realtime.NewBridge(databaseURL, nil)
		bridge.Start(ctx)
		defer bridge.Close()

		orgID := uuid.Must(uuid.NewV7())
		teamID := uuid.Must(uuid.NewV7())
		serviceID := uuid.Must(uuid.NewV7())
		userID := uuid.Must(uuid.NewV7())
		alertID := uuid.Must(uuid.NewV7())

		conn, err := pgx.Connect(ctx, databaseURL)
		require.NoError(a, err)
		defer conn.Close(ctx)

		_, err = conn.Exec(ctx, `INSERT INTO organizations (id, name) VALUES ($1, 'Acme')`, orgID)
		require.NoError(a, err)
		_, err = conn.Exec(ctx, `INSERT INTO teams (id, organization_id, name) VALUES ($1, $2, 'Platform')`, teamID, orgID)
		require.NoError(a, err)
		_, err = conn.Exec(ctx, `INSERT INTO services (id, organization_id, team_id, name) VALUES ($1, $2, $3, 'api')`, serviceID, orgID, teamID)
		require.NoError(a, err)
		_, err = conn.Exec(ctx, `INSERT INTO accounts (id, email) VALUES ($1, 'admin@example.com')`, userID)
		require.NoError(a, err)
		_, err = conn.Exec(ctx, `INSERT INTO users (id, account_id, organization_id, email, role) VALUES ($1, $2, $3, 'admin@example.com', 'admin')`, userID, userID, orgID)
		require.NoError(a, err)

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
		require.NoError(a, err)

		select {
		case evt := <-events:
			require.Equal(a, "INSERT", evt.Op)
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for insert notify")
		}

		_, err = queries.AcknowledgeAlert(ctx, db.AcknowledgeAlertParams{
			ID:                   alertID,
			OrganizationID:       orgID,
			EscalationState:      []byte(`{}`),
			AcknowledgedByUserID: pgtype.UUID{Bytes: userID, Valid: true},
		})
		require.NoError(a, err)

		select {
		case evt := <-events:
			require.Equal(a, alertID, evt.AlertID)
			require.Equal(a, "acknowledged", evt.Status)
			require.Equal(a, "UPDATE", evt.Op)
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for acknowledge notify")
		}
	})
}
