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

func TestTriggeredAlertSchedulesStep1NotificationsWithin5s(t *testing.T) {
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
		oncallID := uuid.Must(uuid.NewV7())
		teamID := uuid.Must(uuid.NewV7())
		serviceID := uuid.Must(uuid.NewV7())
		policyID := uuid.Must(uuid.NewV7())
		stepID := uuid.Must(uuid.NewV7())

		adminAccountID := uuid.Must(uuid.NewV7())
		oncallAccountID := uuid.Must(uuid.NewV7())

		_, err = queries.BootstrapOrganizationWithAdmin(ctx, db.BootstrapOrganizationWithAdminParams{
			OrgID:        orgID,
			OrgName:      "Acme",
			AccountID:    adminAccountID,
			UserID:       adminID,
			Email:        "admin@example.com",
			PasswordHash: pgtype.Text{String: "hash", Valid: true},
		})
		require.NoError(a, err)

		_, err = queries.CreateAccount(ctx, db.CreateAccountParams{
			ID:           oncallAccountID,
			Email:        "oncall@example.com",
			PasswordHash: pgtype.Text{String: "hash", Valid: true},
		})
		require.NoError(a, err)

		_, err = queries.CreateUser(ctx, db.CreateUserParams{
			ID:             oncallID,
			AccountID:      oncallAccountID,
			OrganizationID: orgID,
			Email:          "oncall@example.com",
			Role:           "member",
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
			RepeatLastStep:     false,
			MaxRepeats:         pgtype.Int4{},
		})
		require.NoError(a, err)

		adminChannels, err := json.Marshal([]string{"email", "push"})
		require.NoError(a, err)
		_, err = queries.CreateEscalationStepTarget(ctx, db.CreateEscalationStepTargetParams{
			ID:               uuid.Must(uuid.NewV7()),
			EscalationStepID: stepID,
			OrganizationID:   orgID,
			TargetType:       "user",
			UserID:           pgtype.UUID{Bytes: adminID, Valid: true},
			ScheduleID:       pgtype.UUID{},
			WebhookUrl:       pgtype.Text{},
			Channels:         adminChannels,
		})
		require.NoError(a, err)

		oncallChannels, err := json.Marshal([]string{"email"})
		require.NoError(a, err)
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
		require.NoError(a, err)

		alertID := uuid.Must(uuid.NewV7())
		alert, err := escalation.CreateTriggeredAlert(ctx, queueClient.Pool(), queueClient, escalation.CreateTriggeredAlertParams{
			AlertID:        alertID,
			OrganizationID: orgID,
			ServiceID:      serviceID,
			DedupKey:       "cpu-high",
			Summary:        "CPU above threshold",
			Priority:       "high",
		})
		require.NoError(a, err)
		require.Equal(a, "triggered", alert.Status)

		deadline := time.Now().Add(5 * time.Second)
		for {
			attempts, err := queries.ListNotificationAttemptsByAlertID(ctx, db.ListNotificationAttemptsByAlertIDParams{
				AlertID:        alertID,
				OrganizationID: orgID,
			})
			require.NoError(a, err)
			if len(attempts) != 3 {
				if time.Now().After(deadline) {
					a.T().Fatalf("expected 3 notification attempts within 5s, got %d", len(attempts))
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}

			terminal := 0
			for _, attempt := range attempts {
				switch attempt.Status {
				case "sent", "failed":
					terminal++
				}
			}
			if terminal == 3 {
				break
			}
			if time.Now().After(deadline) {
				a.T().Fatalf("expected 3 terminal notification attempts within 5s, got %d", terminal)
			}
			time.Sleep(100 * time.Millisecond)
		}

		attempts, err := queries.ListNotificationAttemptsByAlertID(ctx, db.ListNotificationAttemptsByAlertIDParams{
			AlertID:        alertID,
			OrganizationID: orgID,
		})
		require.NoError(a, err)
		require.Len(a, attempts, 3)

		channelCounts := map[string]int{}
		terminalStatuses := map[string]int{}
		for _, attempt := range attempts {
			require.True(a, attempt.EscalationStepID.Valid)
			require.Equal(a, stepID, uuid.UUID(attempt.EscalationStepID.Bytes))
			channelCounts[attempt.Channel]++
			terminalStatuses[attempt.Status]++
		}
		require.Equal(a, 2, channelCounts["email"])
		require.Equal(a, 1, channelCounts["push"])
		require.Equal(a, 3, terminalStatuses["failed"]+terminalStatuses["sent"])

		updatedAlert, err := queries.GetAlertByID(ctx, db.GetAlertByIDParams{
			ID:             alertID,
			OrganizationID: orgID,
		})
		require.NoError(a, err)

		var escalationState map[string]int
		require.NoError(a, json.Unmarshal(updatedAlert.EscalationState, &escalationState))
		require.Equal(a, 1, escalationState["current_step"])
	})
}

func TestTriggeredAlertUsesHighPriorityNotificationRuleOrdering(t *testing.T) {
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
			AccountID:    uuid.Must(uuid.NewV7()),
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
			RepeatLastStep:     false,
			MaxRepeats:         pgtype.Int4{},
		})
		require.NoError(a, err)

		adminChannels, err := json.Marshal([]string{"email", "push"})
		require.NoError(a, err)
		_, err = queries.CreateEscalationStepTarget(ctx, db.CreateEscalationStepTargetParams{
			ID:               uuid.Must(uuid.NewV7()),
			EscalationStepID: stepID,
			OrganizationID:   orgID,
			TargetType:       "user",
			UserID:           pgtype.UUID{Bytes: adminID, Valid: true},
			ScheduleID:       pgtype.UUID{},
			WebhookUrl:       pgtype.Text{},
			Channels:         adminChannels,
		})
		require.NoError(a, err)

		highRuleSteps, err := json.Marshal([]map[string]any{
			{"channel": "push", "delay_minutes": 0},
			{"channel": "email", "delay_minutes": 2},
		})
		require.NoError(a, err)
		_, err = queries.UpsertUserNotificationRule(ctx, db.UpsertUserNotificationRuleParams{
			ID:             uuid.Must(uuid.NewV7()),
			OrganizationID: orgID,
			UserID:         adminID,
			Priority:       "high",
			Steps:          highRuleSteps,
		})
		require.NoError(a, err)

		alertID := uuid.Must(uuid.NewV7())
		_, err = escalation.CreateTriggeredAlert(ctx, queueClient.Pool(), queueClient, escalation.CreateTriggeredAlertParams{
			AlertID:        alertID,
			OrganizationID: orgID,
			ServiceID:      serviceID,
			DedupKey:       "cpu-high",
			Summary:        "CPU above threshold",
			Priority:       "high",
		})
		require.NoError(a, err)

		deadline := time.Now().Add(5 * time.Second)
		for {
			attempts, err := queries.ListNotificationAttemptsByAlertID(ctx, db.ListNotificationAttemptsByAlertIDParams{
				AlertID:        alertID,
				OrganizationID: orgID,
			})
			require.NoError(a, err)
			if len(attempts) == 2 {
				break
			}
			if time.Now().After(deadline) {
				a.T().Fatalf("expected 2 notification attempts within 5s, got %d", len(attempts))
			}
			time.Sleep(100 * time.Millisecond)
		}

		attempts, err := queries.ListNotificationAttemptsByAlertID(ctx, db.ListNotificationAttemptsByAlertIDParams{
			AlertID:        alertID,
			OrganizationID: orgID,
		})
		require.NoError(a, err)
		require.Len(a, attempts, 2)
		require.Equal(a, "push", attempts[0].Channel)
		require.Equal(a, "email", attempts[1].Channel)
	})
}
