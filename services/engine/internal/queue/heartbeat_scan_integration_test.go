package queue_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/engine/internal/db"
	"github.com/mdg-labs/escalite/services/engine/internal/heartbeat"
	"github.com/mdg-labs/escalite/services/engine/internal/queue"
	"github.com/mdg-labs/escalite/services/engine/internal/testutil"
)

func TestHeartbeatScanTriggersOverdueMonitor(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		ctx := context.Background()
		databaseURL, cleanup, err := testutil.StartPostgres(ctx)
		require.NoError(a, err)
		defer cleanup()

		require.NoError(a, testutil.MigrateUp(ctx, databaseURL, slog.Default()))

		queueClient, err := queue.New(ctx, queue.Options{
			DatabaseURL: databaseURL,
			Logger:      slog.Default(),
		})
		require.NoError(a, err)
		defer queueClient.Close()

		require.NoError(a, queueClient.Start(ctx))
		defer func() {
			stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = queueClient.Stop(stopCtx)
		}()

		queries := db.New(queueClient.Pool())
		orgID, serviceID, monitorID := seedHeartbeatScanFixture(t, ctx, queries)

		_, err = queueClient.Pool().Exec(ctx,
			`UPDATE heartbeat_monitors
			 SET last_ping_at = now() - interval '2 minutes',
			     status = 'healthy'
			 WHERE id = $1`,
			monitorID,
		)
		require.NoError(a, err)

		worker := queue.NewHeartbeatScanWorker(slog.Default(), queueClient.Pool(), queueClient)
		require.NoError(a, worker.Work(ctx, nil))

		alert, err := queries.GetAlertByServiceDedupKey(ctx, db.GetAlertByServiceDedupKeyParams{
			ServiceID:      serviceID,
			OrganizationID: orgID,
			DedupKey:       monitorID.String(),
		})
		require.NoError(a, err)
		require.Equal(a, "triggered", alert.Status)
		require.Equal(a, monitorID.String(), alert.DedupKey)

		source, err := heartbeat.AlertSourceFromState(alert.EscalationState)
		require.NoError(a, err)
		require.Equal(a, "heartbeat", source)

		var status string
		require.NoError(a, queueClient.Pool().QueryRow(ctx,
			`SELECT status FROM heartbeat_monitors WHERE id = $1`, monitorID,
		).Scan(&status))
		require.Equal(a, "triggered", status)
	})
}

func TestHeartbeatScanMarksMonitorOverdueBeforeTrigger(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		ctx := context.Background()
		databaseURL, cleanup, err := testutil.StartPostgres(ctx)
		require.NoError(a, err)
		defer cleanup()

		require.NoError(a, testutil.MigrateUp(ctx, databaseURL, slog.Default()))

		queueClient, err := queue.New(ctx, queue.Options{
			DatabaseURL: databaseURL,
			Logger:      slog.Default(),
		})
		require.NoError(a, err)
		defer queueClient.Close()

		queries := db.New(queueClient.Pool())
		_, _, monitorID := seedHeartbeatScanFixture(t, ctx, queries)

		_, err = queueClient.Pool().Exec(ctx,
			`UPDATE heartbeat_monitors
			 SET last_ping_at = now() - interval '45 seconds',
			     status = 'healthy',
			     interval_seconds = 30,
			     grace_seconds = 60
			 WHERE id = $1`,
			monitorID,
		)
		require.NoError(a, err)

		require.NoError(a, heartbeat.ScanOverdueMonitors(ctx, queueClient.Pool(), queueClient, slog.Default()))

		var status string
		require.NoError(a, queueClient.Pool().QueryRow(ctx,
			`SELECT status FROM heartbeat_monitors WHERE id = $1`, monitorID,
		).Scan(&status))
		require.Equal(a, "overdue", status)
	})
}

func seedHeartbeatScanFixture(
	t *testing.T,
	ctx context.Context,
	queries *db.Queries,
) (orgID, serviceID, monitorID uuid.UUID) {
	t.Helper()

	orgID = uuid.Must(uuid.NewV7())
	adminID := uuid.Must(uuid.NewV7())
	teamID := uuid.Must(uuid.NewV7())
	serviceID = uuid.Must(uuid.NewV7())
	policyID := uuid.Must(uuid.NewV7())
	stepID := uuid.Must(uuid.NewV7())
	monitorID = uuid.Must(uuid.NewV7())

	_, err := queries.BootstrapOrganizationWithAdmin(ctx, db.BootstrapOrganizationWithAdminParams{
		OrgID:        orgID,
		OrgName:      "Acme",
		UserID:       adminID,
		Email:        "admin@example.com",
		PasswordHash: pgtype.Text{String: "hash", Valid: true},
	})
	require.NoError(t, err)

	_, err = queries.CreateTeam(ctx, db.CreateTeamParams{
		ID:             teamID,
		OrganizationID: orgID,
		Name:           "Platform",
	})
	require.NoError(t, err)

	_, err = queries.CreateService(ctx, db.CreateServiceParams{
		ID:             serviceID,
		OrganizationID: orgID,
		TeamID:         teamID,
		Name:           "checkout-api",
	})
	require.NoError(t, err)

	_, err = queries.CreateEscalationPolicy(ctx, db.CreateEscalationPolicyParams{
		ID:             policyID,
		OrganizationID: orgID,
		ServiceID:      serviceID,
		Name:           "Default",
	})
	require.NoError(t, err)

	_, err = queries.CreateEscalationStep(ctx, db.CreateEscalationStepParams{
		ID:                 stepID,
		EscalationPolicyID: policyID,
		OrganizationID:     orgID,
		StepOrder:          1,
		DelayMinutes:       0,
		RepeatLastStep:     false,
		MaxRepeats:         pgtype.Int4{},
	})
	require.NoError(t, err)

	channels, err := json.Marshal([]string{"email"})
	require.NoError(t, err)
	_, err = queries.CreateEscalationStepTarget(ctx, db.CreateEscalationStepTargetParams{
		ID:               uuid.Must(uuid.NewV7()),
		EscalationStepID: stepID,
		OrganizationID:   orgID,
		TargetType:       "user",
		UserID:           pgtype.UUID{Bytes: adminID, Valid: true},
		ScheduleID:       pgtype.UUID{},
		WebhookUrl:       pgtype.Text{},
		Channels:         channels,
	})
	require.NoError(t, err)

	_, err = queries.CreateHeartbeatMonitor(ctx, db.CreateHeartbeatMonitorParams{
		ID:              monitorID,
		OrganizationID:  orgID,
		ServiceID:       serviceID,
		Name:            "nightly-backup",
		IntervalSeconds: 30,
		GraceSeconds:    30,
		TokenHash:       "hash-" + monitorID.String(),
		Prefix:          "hb_",
		Status:          "healthy",
		LastPingAt:      pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	})
	require.NoError(t, err)

	return orgID, serviceID, monitorID
}
