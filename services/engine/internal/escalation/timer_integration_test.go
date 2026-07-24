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

func TestEscalationTimerAdvancesCurrentStep(t *testing.T) {
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
	fixture := bootstrapTwoStepEscalation(t, ctx, queries, 0)

	alertID := uuid.Must(uuid.NewV7())
	_, err = escalation.CreateTriggeredAlert(ctx, queueClient.Pool(), queueClient, escalation.CreateTriggeredAlertParams{
		AlertID:        alertID,
		OrganizationID: fixture.orgID,
		ServiceID:      fixture.serviceID,
		DedupKey:       "disk-full",
		Summary:        "Disk usage critical",
		Priority:       "high",
	})
	require.NoError(t, err)

	waitForNotificationCount(t, ctx, queries, alertID, fixture.orgID, 1)

	deadline := time.Now().Add(10 * time.Second)
	for {
		alert, err := queries.GetAlertByID(ctx, db.GetAlertByIDParams{
			ID:             alertID,
			OrganizationID: fixture.orgID,
		})
		require.NoError(t, err)

		var state escalation.State
		require.NoError(t, json.Unmarshal(alert.EscalationState, &state))
		if state.CurrentStep == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected current_step=2 within 10s, got %d", state.CurrentStep)
		}
		time.Sleep(100 * time.Millisecond)
	}

	attempts, err := queries.ListNotificationAttemptsByAlertID(ctx, db.ListNotificationAttemptsByAlertIDParams{
		AlertID:        alertID,
		OrganizationID: fixture.orgID,
	})
	require.NoError(t, err)
	require.Len(t, attempts, 2)
	require.Equal(t, fixture.step2ID, uuid.UUID(attempts[1].EscalationStepID.Bytes))
}

func TestAcknowledgedAlertCancelsPendingEscalation(t *testing.T) {
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
	fixture := bootstrapTwoStepEscalation(t, ctx, queries, 30)

	alertID := uuid.Must(uuid.NewV7())
	_, err = escalation.CreateTriggeredAlert(ctx, queueClient.Pool(), queueClient, escalation.CreateTriggeredAlertParams{
		AlertID:        alertID,
		OrganizationID: fixture.orgID,
		ServiceID:      fixture.serviceID,
		DedupKey:       "memory-high",
		Summary:        "Memory above threshold",
		Priority:       "high",
	})
	require.NoError(t, err)

	waitForNotificationCount(t, ctx, queries, alertID, fixture.orgID, 1)

	alert, err := queries.GetAlertByID(ctx, db.GetAlertByIDParams{
		ID:             alertID,
		OrganizationID: fixture.orgID,
	})
	require.NoError(t, err)

	var beforeAck escalation.State
	require.NoError(t, json.Unmarshal(alert.EscalationState, &beforeAck))
	require.NotNil(t, beforeAck.PendingEscalationJobID)

	acknowledged, err := escalation.AcknowledgeAlert(ctx, queries, queueClient, alertID, fixture.orgID, fixture.adminID)
	require.NoError(t, err)
	require.Equal(t, "acknowledged", acknowledged.Status)
	require.NotNil(t, acknowledged.AcknowledgedAt.Valid)

	var afterAck escalation.State
	require.NoError(t, json.Unmarshal(acknowledged.EscalationState, &afterAck))
	require.Nil(t, afterAck.PendingEscalationJobID)
	require.Nil(t, afterAck.NextEscalationAt)

	time.Sleep(2 * time.Second)

	count, err := queries.CountNotificationAttemptsByAlertID(ctx, db.CountNotificationAttemptsByAlertIDParams{
		AlertID:        alertID,
		OrganizationID: fixture.orgID,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}

type escalationFixture struct {
	orgID     uuid.UUID
	adminID   uuid.UUID
	serviceID uuid.UUID
	step2ID   uuid.UUID
}

func bootstrapTwoStepEscalation(t *testing.T, ctx context.Context, queries *db.Queries, step1DelayMinutes int32) escalationFixture {
	t.Helper()

	orgID := uuid.Must(uuid.NewV7())
	adminID := uuid.Must(uuid.NewV7())
	oncallID := uuid.Must(uuid.NewV7())
	teamID := uuid.Must(uuid.NewV7())
	serviceID := uuid.Must(uuid.NewV7())
	policyID := uuid.Must(uuid.NewV7())
	step1ID := uuid.Must(uuid.NewV7())
	step2ID := uuid.Must(uuid.NewV7())

	_, err := queries.BootstrapOrganizationWithAdmin(ctx, db.BootstrapOrganizationWithAdminParams{
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
		ID:                 step1ID,
		EscalationPolicyID: policyID,
		OrganizationID:     orgID,
		StepOrder:          1,
		DelayMinutes:       step1DelayMinutes,
		RepeatLastStep:     false,
		MaxRepeats:         pgtype.Int4{},
	})
	require.NoError(t, err)

	_, err = queries.CreateEscalationStep(ctx, db.CreateEscalationStepParams{
		ID:                 step2ID,
		EscalationPolicyID: policyID,
		OrganizationID:     orgID,
		StepOrder:          2,
		DelayMinutes:       5,
		RepeatLastStep:     false,
		MaxRepeats:         pgtype.Int4{},
	})
	require.NoError(t, err)

	channels, err := json.Marshal([]string{"email"})
	require.NoError(t, err)

	_, err = queries.CreateEscalationStepTarget(ctx, db.CreateEscalationStepTargetParams{
		ID:               uuid.Must(uuid.NewV7()),
		EscalationStepID: step1ID,
		OrganizationID:   orgID,
		TargetType:       "user",
		UserID:           pgtype.UUID{Bytes: adminID, Valid: true},
		ScheduleID:       pgtype.UUID{},
		WebhookUrl:       pgtype.Text{},
		Channels:         channels,
	})
	require.NoError(t, err)

	_, err = queries.CreateEscalationStepTarget(ctx, db.CreateEscalationStepTargetParams{
		ID:               uuid.Must(uuid.NewV7()),
		EscalationStepID: step2ID,
		OrganizationID:   orgID,
		TargetType:       "user",
		UserID:           pgtype.UUID{Bytes: oncallID, Valid: true},
		ScheduleID:       pgtype.UUID{},
		WebhookUrl:       pgtype.Text{},
		Channels:         channels,
	})
	require.NoError(t, err)

	return escalationFixture{
		orgID:     orgID,
		adminID:   adminID,
		serviceID: serviceID,
		step2ID:   step2ID,
	}
}

func TestEscalationRepeatExhaustedAtMaxRepeats(t *testing.T) {
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
	fixture := bootstrapSingleStepRepeatEscalation(t, ctx, queries, 0, 1)

	alertID := uuid.Must(uuid.NewV7())
	_, err = escalation.CreateTriggeredAlert(ctx, queueClient.Pool(), queueClient, escalation.CreateTriggeredAlertParams{
		AlertID:        alertID,
		OrganizationID: fixture.orgID,
		ServiceID:      fixture.serviceID,
		DedupKey:       "repeat-cap",
		Summary:        "Repeat boundary",
		Priority:       "high",
	})
	require.NoError(t, err)

	waitForNotificationCount(t, ctx, queries, alertID, fixture.orgID, 1)

	deadline := time.Now().Add(10 * time.Second)
	for {
		alert, err := queries.GetAlertByID(ctx, db.GetAlertByIDParams{
			ID:             alertID,
			OrganizationID: fixture.orgID,
		})
		require.NoError(t, err)

		var state escalation.State
		require.NoError(t, json.Unmarshal(alert.EscalationState, &state))
		if state.EscalatedExhausted {
			require.Equal(t, 1, state.RepeatCount)
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected escalated_exhausted within 10s, got state %+v", state)
		}
		time.Sleep(100 * time.Millisecond)
	}

	count, err := queries.CountNotificationAttemptsByAlertID(ctx, db.CountNotificationAttemptsByAlertIDParams{
		AlertID:        alertID,
		OrganizationID: fixture.orgID,
	})
	require.NoError(t, err)
	require.Equal(t, int64(2), count)
}

func bootstrapSingleStepRepeatEscalation(
	t *testing.T,
	ctx context.Context,
	queries *db.Queries,
	stepDelayMinutes int32,
	maxRepeats int32,
) escalationFixture {
	t.Helper()

	orgID := uuid.Must(uuid.NewV7())
	adminID := uuid.Must(uuid.NewV7())
	teamID := uuid.Must(uuid.NewV7())
	serviceID := uuid.Must(uuid.NewV7())
	policyID := uuid.Must(uuid.NewV7())
	stepID := uuid.Must(uuid.NewV7())

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
		DelayMinutes:       stepDelayMinutes,
		RepeatLastStep:     true,
		MaxRepeats:         pgtype.Int4{Int32: maxRepeats, Valid: true},
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

	return escalationFixture{
		orgID:     orgID,
		adminID:   adminID,
		serviceID: serviceID,
		step2ID:   stepID,
	}
}

func waitForNotificationCount(
	t *testing.T,
	ctx context.Context,
	queries *db.Queries,
	alertID, orgID uuid.UUID,
	expected int64,
) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for {
		count, err := queries.CountNotificationAttemptsByAlertID(ctx, db.CountNotificationAttemptsByAlertIDParams{
			AlertID:        alertID,
			OrganizationID: orgID,
		})
		require.NoError(t, err)
		if count == expected {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected %d notification attempts within 5s, got %d", expected, count)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
