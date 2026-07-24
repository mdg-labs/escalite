package escalation_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/engine/internal/db"
	"github.com/mdg-labs/escalite/services/engine/internal/escalation"
	"github.com/mdg-labs/escalite/services/engine/internal/queue"
	"github.com/mdg-labs/escalite/services/engine/internal/testutil"
)

func TestTriggeredAlertSchedulesStep1NotificationsWithin5s(t *testing.T) {
	ctx := context.Background()
	databaseURL, cleanup, err := testutil.StartPostgres(ctx)
	require.NoError(t, err)
	defer cleanup()

	require.NoError(t, testutil.MigrateUp(ctx, databaseURL, slog.Default()))

	queueClient, err := queue.New(ctx, queue.Options{
		DatabaseURL: databaseURL,
		Logger:      slog.Default(),
	})
	require.NoError(t, err)
	defer queueClient.Close()

	require.NoError(t, queueClient.Start(ctx))
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

	_, err = queries.BootstrapOrganizationWithAdmin(ctx, db.BootstrapOrganizationWithAdminParams{
		OrgID:        orgID,
		OrgName:      "Acme",
		UserID:       adminID,
		Email:        "admin@example.com",
		PasswordHash: pgtype.Text{String: "hash", Valid: true},
	})
	require.NoError(t, err)

	_, err = queries.CreateUser(ctx, db.CreateUserParams{
		ID:             oncallID,
		OrganizationID: orgID,
		Email:          "oncall@example.com",
		PasswordHash:   pgtype.Text{String: "hash", Valid: true},
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

	adminChannels, err := json.Marshal([]string{"email", "push"})
	require.NoError(t, err)
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
	require.NoError(t, err)

	oncallChannels, err := json.Marshal([]string{"email"})
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

	alertID := uuid.Must(uuid.NewV7())
	alert, err := escalation.CreateTriggeredAlert(ctx, queueClient.Pool(), queueClient, escalation.CreateTriggeredAlertParams{
		AlertID:        alertID,
		OrganizationID: orgID,
		ServiceID:      serviceID,
		DedupKey:       "cpu-high",
		Summary:        "CPU above threshold",
		Priority:       "high",
	})
	require.NoError(t, err)
	require.Equal(t, "triggered", alert.Status)

	deadline := time.Now().Add(5 * time.Second)
	for {
		count, err := queries.CountNotificationAttemptsByAlertID(ctx, db.CountNotificationAttemptsByAlertIDParams{
			AlertID:        alertID,
			OrganizationID: orgID,
		})
		require.NoError(t, err)
		if count == 3 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected 3 notification attempts within 5s, got %d", count)
		}
		time.Sleep(100 * time.Millisecond)
	}

	attempts, err := queries.ListNotificationAttemptsByAlertID(ctx, db.ListNotificationAttemptsByAlertIDParams{
		AlertID:        alertID,
		OrganizationID: orgID,
	})
	require.NoError(t, err)
	require.Len(t, attempts, 3)

	channelCounts := map[string]int{}
	for _, attempt := range attempts {
		require.Equal(t, "pending", attempt.Status)
		require.True(t, attempt.EscalationStepID.Valid)
		require.Equal(t, stepID, uuid.UUID(attempt.EscalationStepID.Bytes))
		channelCounts[attempt.Channel]++
	}
	require.Equal(t, 2, channelCounts["email"])
	require.Equal(t, 1, channelCounts["push"])

	updatedAlert, err := queries.GetAlertByID(ctx, db.GetAlertByIDParams{
		ID:             alertID,
		OrganizationID: orgID,
	})
	require.NoError(t, err)

	var escalationState map[string]int
	require.NoError(t, json.Unmarshal(updatedAlert.EscalationState, &escalationState))
	require.Equal(t, 1, escalationState["current_step"])
}
