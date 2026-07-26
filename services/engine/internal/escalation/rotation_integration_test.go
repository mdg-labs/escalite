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

func TestTriggeredAlertNotifiesRotationScheduleOnCallUsers(t *testing.T) {
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
	scheduleID := uuid.Must(uuid.NewV7())
	rotationID := uuid.Must(uuid.NewV7())

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

	_, err = queries.CreateSchedule(ctx, db.CreateScheduleParams{
		ID:             scheduleID,
		OrganizationID: orgID,
		TeamID:         teamID,
		Name:           "Primary",
		Timezone:       "UTC",
	})
	require.NoError(t, err)

	participants, err := json.Marshal([]string{oncallID.String(), adminID.String()})
	require.NoError(t, err)
	_, err = queries.CreateRotation(ctx, db.CreateRotationParams{
		ID:             rotationID,
		ScheduleID:     scheduleID,
		OrganizationID: orgID,
		Name:           "Layer 1",
		Layer:          1,
		Rrule:          "FREQ=DAILY;INTERVAL=1",
		Participants:   participants,
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
		TargetType:       "rotation",
		UserID:           pgtype.UUID{},
		ScheduleID:       pgtype.UUID{Bytes: scheduleID, Valid: true},
		WebhookUrl:       pgtype.Text{},
		Channels:         channels,
	})
	require.NoError(t, err)

	alertID := uuid.Must(uuid.NewV7())
	_, err = escalation.CreateTriggeredAlert(ctx, queueClient.Pool(), queueClient, escalation.CreateTriggeredAlertParams{
		AlertID:        alertID,
		OrganizationID: orgID,
		ServiceID:      serviceID,
		DedupKey:       "cpu-high",
		Summary:        "CPU above threshold",
		Priority:       "high",
	})
	require.NoError(t, err)

	deadline := time.Now().Add(5 * time.Second)
	for {
		count, err := queries.CountNotificationAttemptsByAlertID(ctx, db.CountNotificationAttemptsByAlertIDParams{
			AlertID:        alertID,
			OrganizationID: orgID,
		})
		require.NoError(t, err)
		if count == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected 1 notification attempt within 5s, got %d", count)
		}
		time.Sleep(100 * time.Millisecond)
	}

	attempts, err := queries.ListNotificationAttemptsByAlertID(ctx, db.ListNotificationAttemptsByAlertIDParams{
		AlertID:        alertID,
		OrganizationID: orgID,
	})
	require.NoError(t, err)
	require.Len(t, attempts, 1)
	require.Equal(t, "email", attempts[0].Channel)

	var recipient map[string]string
	require.NoError(t, json.Unmarshal(attempts[0].Recipient, &recipient))
	require.Equal(t, oncallID.String(), recipient["user_id"])
	require.Equal(t, "oncall@example.com", recipient["email"])
}

func TestTriggeredAlertSkipsEmptyRotationTargetWithoutPanic(t *testing.T) {
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
	teamID := uuid.Must(uuid.NewV7())
	serviceID := uuid.Must(uuid.NewV7())
	policyID := uuid.Must(uuid.NewV7())
	stepID := uuid.Must(uuid.NewV7())
	scheduleID := uuid.Must(uuid.NewV7())

	_, err = queries.BootstrapOrganizationWithAdmin(ctx, db.BootstrapOrganizationWithAdminParams{
		OrgID:        orgID,
		OrgName:      "Acme",
		AccountID:    uuid.Must(uuid.NewV7()),
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

	_, err = queries.CreateSchedule(ctx, db.CreateScheduleParams{
		ID:             scheduleID,
		OrganizationID: orgID,
		TeamID:         teamID,
		Name:           "Empty",
		Timezone:       "UTC",
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
		TargetType:       "rotation",
		UserID:           pgtype.UUID{},
		ScheduleID:       pgtype.UUID{Bytes: scheduleID, Valid: true},
		WebhookUrl:       pgtype.Text{},
		Channels:         channels,
	})
	require.NoError(t, err)

	alertID := uuid.Must(uuid.NewV7())
	_, err = escalation.CreateTriggeredAlert(ctx, queueClient.Pool(), queueClient, escalation.CreateTriggeredAlertParams{
		AlertID:        alertID,
		OrganizationID: orgID,
		ServiceID:      serviceID,
		DedupKey:       "disk-full",
		Summary:        "Disk full",
		Priority:       "high",
	})
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	count, err := queries.CountNotificationAttemptsByAlertID(ctx, db.CountNotificationAttemptsByAlertIDParams{
		AlertID:        alertID,
		OrganizationID: orgID,
	})
	require.NoError(t, err)
	require.Equal(t, int64(0), count)
}
