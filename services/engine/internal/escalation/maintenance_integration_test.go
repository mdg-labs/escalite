package escalation_test

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
	"github.com/mdg-labs/escalite/services/engine/internal/escalation"
	"github.com/mdg-labs/escalite/services/engine/internal/queue"
	"github.com/mdg-labs/escalite/services/engine/internal/testutil"
	_ "github.com/mdg-labs/escalite/services/engine/channelsinstall"
)

func TestTriggeredAlertSkipsNotificationsDuringMaintenance(t *testing.T) {
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

		orgID := uuid.Must(uuid.NewV7())
		adminID := uuid.Must(uuid.NewV7())
		teamID := uuid.Must(uuid.NewV7())
		serviceID := uuid.Must(uuid.NewV7())
		policyID := uuid.Must(uuid.NewV7())
		stepID := uuid.Must(uuid.NewV7())

		_, err = queries.BootstrapOrganizationWithAdmin(ctx, db.BootstrapOrganizationWithAdminParams{
			OrgID:        orgID,
			OrgName:      "Acme",
			UserID:       adminID,
			Email:        "admin@example.com",
			PasswordHash: pgtype.Text{String: "hash", Valid: true},
		})
		require.NoError(a, err)

		_, err = queries.CreateTeam(ctx, db.CreateTeamParams{
			ID:             teamID,
			OrganizationID: orgID,
			Name:           "Platform",
		})
		require.NoError(a, err)

		_, err = queries.CreateService(ctx, db.CreateServiceParams{
			ID:             serviceID,
			OrganizationID: orgID,
			TeamID:         teamID,
			Name:           "checkout-api",
		})
		require.NoError(a, err)

		now := time.Now().UTC()
		_, err = queueClient.Pool().Exec(ctx, `
			INSERT INTO maintenance_windows (
				id, organization_id, service_id, description, starts_at, ends_at,
				suppress_notifications, suppress_ingestion
			) VALUES ($1, $2, $3, $4, $5, $6, true, false)
		`, uuid.Must(uuid.NewV7()), orgID, serviceID, "Deploy window", now.Add(-time.Hour), now.Add(time.Hour))
		require.NoError(a, err)

		_, err = queries.CreateEscalationPolicy(ctx, db.CreateEscalationPolicyParams{
			ID:             policyID,
			OrganizationID: orgID,
			ServiceID:      serviceID,
			Name:           "Default",
		})
		require.NoError(a, err)

		_, err = queries.CreateEscalationStep(ctx, db.CreateEscalationStepParams{
			ID:                 stepID,
			EscalationPolicyID: policyID,
			OrganizationID:     orgID,
			StepOrder:          1,
			DelayMinutes:       0,
		})
		require.NoError(a, err)

		channels, err := json.Marshal([]string{"email"})
		require.NoError(a, err)
		_, err = queries.CreateEscalationStepTarget(ctx, db.CreateEscalationStepTargetParams{
			ID:               uuid.Must(uuid.NewV7()),
			EscalationStepID: stepID,
			OrganizationID:   orgID,
			TargetType:       "user",
			UserID:           pgtype.UUID{Bytes: adminID, Valid: true},
			Channels:         channels,
		})
		require.NoError(a, err)

		alertID := uuid.Must(uuid.NewV7())
		alert, err := escalation.CreateTriggeredAlert(ctx, queueClient.Pool(), queueClient, escalation.CreateTriggeredAlertParams{
			AlertID:        alertID,
			OrganizationID: orgID,
			ServiceID:      serviceID,
			DedupKey:       "deploy",
			Summary:        "Deploy in progress",
			Priority:       "high",
		})
		require.NoError(a, err)
		require.Equal(a, "triggered", alert.Status)

		deadline := time.Now().Add(3 * time.Second)
		for {
			attempts, err := queries.ListNotificationAttemptsByAlertID(ctx, db.ListNotificationAttemptsByAlertIDParams{
				AlertID:        alertID,
				OrganizationID: orgID,
			})
			require.NoError(a, err)
			if len(attempts) == 0 {
				break
			}
			if time.Now().After(deadline) {
				a.T().Fatalf("expected no notification attempts during maintenance, got %d", len(attempts))
			}
			time.Sleep(100 * time.Millisecond)
		}
	})
}
