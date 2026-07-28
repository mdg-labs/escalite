package incident_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/engine/internal/db"
	"github.com/mdg-labs/escalite/services/engine/internal/escalation"
	"github.com/mdg-labs/escalite/services/engine/internal/queue"
	"github.com/mdg-labs/escalite/services/engine/internal/testutil"
	_ "github.com/mdg-labs/escalite/services/engine/channelsinstall"
)

func TestAutoPromoteCreatesIncidentWithoutDuplicate(t *testing.T) {
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
		orgID, serviceID, _ := seedAutoPromoteService(t, ctx, queueClient.Pool(), queries, 3)

		for i := 0; i < 3; i++ {
			_, err := escalation.CreateTriggeredAlert(ctx, queueClient.Pool(), queueClient, escalation.CreateTriggeredAlertParams{
				AlertID:        uuid.Must(uuid.NewV7()),
				OrganizationID: orgID,
				ServiceID:      serviceID,
				DedupKey:       fmt.Sprintf("auto-promote-%d", i),
				Summary:        "Checkout degradation",
				Priority:       "high",
			})
			require.NoError(a, err)
		}

		var incidentCount int
		err = queueClient.Pool().QueryRow(ctx, `
			SELECT count(*)
			FROM incidents
			WHERE organization_id = $1
			  AND status <> 'resolved'
		`, orgID).Scan(&incidentCount)
		require.NoError(a, err)
		require.Equal(a, 1, incidentCount)

		var attachedCount int
		err = queueClient.Pool().QueryRow(ctx, `
			SELECT count(*)
			FROM alerts
			WHERE organization_id = $1
			  AND service_id = $2
			  AND incident_id IS NOT NULL
		`, orgID, serviceID).Scan(&attachedCount)
		require.NoError(a, err)
		require.Equal(a, 3, attachedCount)

		_, err = escalation.CreateTriggeredAlert(ctx, queueClient.Pool(), queueClient, escalation.CreateTriggeredAlertParams{
			AlertID:        uuid.Must(uuid.NewV7()),
			OrganizationID: orgID,
			ServiceID:      serviceID,
			DedupKey:       "auto-promote-d",
			Summary:        "Disk full",
			Priority:       "high",
		})
		require.NoError(a, err)

		err = queueClient.Pool().QueryRow(ctx, `
			SELECT count(*)
			FROM incidents
			WHERE organization_id = $1
			  AND status <> 'resolved'
		`, orgID).Scan(&incidentCount)
		require.NoError(a, err)
		require.Equal(a, 1, incidentCount)

		err = queueClient.Pool().QueryRow(ctx, `
			SELECT count(*)
			FROM alerts
			WHERE organization_id = $1
			  AND service_id = $2
			  AND incident_id IS NOT NULL
		`, orgID, serviceID).Scan(&attachedCount)
		require.NoError(a, err)
		require.Equal(a, 4, attachedCount)
	})
}

func TestAutoPromoteSuppressesEscalationForConfiguredPriorities(t *testing.T) {
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
		orgID, serviceID, _ := seedAutoPromoteService(t, ctx, queueClient.Pool(), queries, 1)

		alertID := uuid.Must(uuid.NewV7())
		_, err = escalation.CreateTriggeredAlert(ctx, queueClient.Pool(), queueClient, escalation.CreateTriggeredAlertParams{
			AlertID:        alertID,
			OrganizationID: orgID,
			ServiceID:      serviceID,
			DedupKey:       "suppressed-alert",
			Summary:        "Memory pressure",
			Priority:       "high",
		})
		require.NoError(a, err)

		storedAlert, err := queries.GetAlertByID(ctx, db.GetAlertByIDParams{
			ID:             alertID,
			OrganizationID: orgID,
		})
		require.NoError(a, err)
		require.True(a, storedAlert.IncidentID.Valid)

		deadline := time.Now().Add(5 * time.Second)
		for {
			count, err := queries.CountNotificationAttemptsByAlertID(ctx, db.CountNotificationAttemptsByAlertIDParams{
				AlertID:        alertID,
				OrganizationID: orgID,
			})
			require.NoError(a, err)
			if count > 0 {
				a.T().Fatalf("expected no notification attempts for incident-grouped alert, got %d", count)
			}
			if time.Now().After(deadline) {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	})
}

func TestIncidentCloseResumesEscalationForTriggeredAlerts(t *testing.T) {
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
		orgID, serviceID, _ := seedAutoPromoteService(t, ctx, queueClient.Pool(), queries, 1)

		alertID := uuid.Must(uuid.NewV7())
		_, err = escalation.CreateTriggeredAlert(ctx, queueClient.Pool(), queueClient, escalation.CreateTriggeredAlertParams{
			AlertID:        alertID,
			OrganizationID: orgID,
			ServiceID:      serviceID,
			DedupKey:       "resume-on-close",
			Summary:        "Database latency",
			Priority:       "high",
		})
		require.NoError(a, err)

		storedAlert, err := queries.GetAlertByID(ctx, db.GetAlertByIDParams{
			ID:             alertID,
			OrganizationID: orgID,
		})
		require.NoError(a, err)
		require.True(a, storedAlert.IncidentID.Valid)

		deadline := time.Now().Add(5 * time.Second)
		for {
			count, err := queries.CountNotificationAttemptsByAlertID(ctx, db.CountNotificationAttemptsByAlertIDParams{
				AlertID:        alertID,
				OrganizationID: orgID,
			})
			require.NoError(a, err)
			if count > 0 {
				a.T().Fatalf("expected no notification attempts while incident is open, got %d", count)
			}
			if time.Now().After(deadline) {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		incidentID := uuid.UUID(storedAlert.IncidentID.Bytes)
		_, err = queueClient.Pool().Exec(ctx, `
			UPDATE incidents
			SET status = 'resolved',
			    resolved_at = now(),
			    updated_at = now()
			WHERE id = $1
			  AND organization_id = $2
		`, incidentID, orgID)
		require.NoError(a, err)

		require.NoError(a, escalation.ResumeEscalationOnIncidentClose(ctx, queries, queueClient, incidentID, orgID))

		deadline = time.Now().Add(5 * time.Second)
		for {
			count, err := queries.CountNotificationAttemptsByAlertID(ctx, db.CountNotificationAttemptsByAlertIDParams{
				AlertID:        alertID,
				OrganizationID: orgID,
			})
			require.NoError(a, err)
			if count > 0 {
				return
			}
			if time.Now().After(deadline) {
				a.T().Fatal("expected notification attempts after incident close")
			}
			time.Sleep(100 * time.Millisecond)
		}
	})
}

func seedAutoPromoteService(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	queries *db.Queries,
	threshold int32,
) (uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()

	orgID := uuid.Must(uuid.NewV7())
	adminID := uuid.Must(uuid.NewV7())
	oncallID := uuid.Must(uuid.NewV7())
	teamID := uuid.Must(uuid.NewV7())
	serviceID := uuid.Must(uuid.NewV7())
	policyID := uuid.Must(uuid.NewV7())
	stepID := uuid.Must(uuid.NewV7())

	adminAccountID := uuid.Must(uuid.NewV7())
	oncallAccountID := uuid.Must(uuid.NewV7())

	_, err := queries.BootstrapOrganizationWithAdmin(ctx, db.BootstrapOrganizationWithAdminParams{
		OrgID:        orgID,
		OrgName:      "Acme",
		AccountID:    adminAccountID,
		UserID:       adminID,
		Email:        "admin@example.com",
		PasswordHash: pgtype.Text{String: "hash", Valid: true},
	})
	require.NoError(t, err)

	_, err = queries.CreateAccount(ctx, db.CreateAccountParams{
		ID:           oncallAccountID,
		Email:        "oncall@example.com",
		PasswordHash: pgtype.Text{String: "hash", Valid: true},
	})
	require.NoError(t, err)

	_, err = queries.CreateUser(ctx, db.CreateUserParams{
		ID:             oncallID,
		AccountID:      oncallAccountID,
		OrganizationID: orgID,
		Email:          "oncall@example.com",
		Role:           "member",
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

	_, err = pool.Exec(ctx, `
		UPDATE services
		SET auto_promote_enabled = true,
		    auto_promote_alert_threshold = $2,
		    auto_promote_window_seconds = 300,
		    auto_promote_suppress_escalation_priorities = ARRAY['high'::text, 'low'::text]
		WHERE id = $1
	`, serviceID, threshold)
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
	})
	require.NoError(t, err)

	oncallChannels, err := json.Marshal([]string{"email", "push"})
	require.NoError(t, err)

	_, err = queries.CreateEscalationStepTarget(ctx, db.CreateEscalationStepTargetParams{
		ID:               uuid.Must(uuid.NewV7()),
		EscalationStepID: stepID,
		OrganizationID:   orgID,
		TargetType:       "user",
		UserID:           pgtype.UUID{Bytes: oncallID, Valid: true},
		ScheduleID:       pgtype.UUID{},
		WebhookUrl:       pgtype.Text{},
		Channels:         oncallChannels,
	})
	require.NoError(t, err)

	return orgID, serviceID, oncallID
}
