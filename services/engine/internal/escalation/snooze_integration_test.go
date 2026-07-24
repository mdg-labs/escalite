package escalation_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/engine/internal/db"
	"github.com/mdg-labs/escalite/services/engine/internal/escalation"
	"github.com/mdg-labs/escalite/services/engine/internal/queue"
	"github.com/mdg-labs/escalite/services/engine/internal/testutil"
)

func TestSnoozeEscalationShiftsNextEscalationAt(t *testing.T) {
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
		DedupKey:       "snooze-test",
		Summary:        "Snooze target",
		Priority:       "high",
	})
	require.NoError(t, err)

	waitForNotificationCount(t, ctx, queries, alertID, fixture.orgID, 1)

	before, err := queries.GetAlertByID(ctx, db.GetAlertByIDParams{
		ID:             alertID,
		OrganizationID: fixture.orgID,
	})
	require.NoError(t, err)

	var beforeState escalation.State
	require.NoError(t, json.Unmarshal(before.EscalationState, &beforeState))
	require.NotNil(t, beforeState.NextEscalationAt)

	snoozeMinutes := int32(15)
	_, err = escalation.SnoozeEscalation(ctx, queries, queueClient, queueClient, alertID, fixture.orgID, snoozeMinutes)
	require.NoError(t, err)

	after, err := queries.GetAlertByID(ctx, db.GetAlertByIDParams{
		ID:             alertID,
		OrganizationID: fixture.orgID,
	})
	require.NoError(t, err)

	var afterState escalation.State
	require.NoError(t, json.Unmarshal(after.EscalationState, &afterState))
	require.NotNil(t, afterState.NextEscalationAt)
	require.NotNil(t, afterState.PendingEscalationJobID)

	expected := beforeState.NextEscalationAt.Add(time.Duration(snoozeMinutes) * time.Minute)
	require.WithinDuration(t, expected, *afterState.NextEscalationAt, 2*time.Second)
}

func TestReEscalateAlertResetsToStep1AndEnqueuesNotifications(t *testing.T) {
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
		DedupKey:       "re-escalate-test",
		Summary:        "Re-escalate target",
		Priority:       "high",
	})
	require.NoError(t, err)

	waitForNotificationCount(t, ctx, queries, alertID, fixture.orgID, 1)

	acknowledged, err := escalation.AcknowledgeAlert(ctx, queries, queueClient, alertID, fixture.orgID, fixture.adminID)
	require.NoError(t, err)
	require.Equal(t, "acknowledged", acknowledged.Status)

	_, err = escalation.ReEscalateAlert(ctx, queries, queueClient, queueClient, alertID, fixture.orgID)
	require.NoError(t, err)

	waitForNotificationCount(t, ctx, queries, alertID, fixture.orgID, 2)

	reEscalated, err := queries.GetAlertByID(ctx, db.GetAlertByIDParams{
		ID:             alertID,
		OrganizationID: fixture.orgID,
	})
	require.NoError(t, err)
	require.Equal(t, "triggered", reEscalated.Status)
	require.False(t, reEscalated.AcknowledgedAt.Valid)

	var state escalation.State
	require.NoError(t, json.Unmarshal(reEscalated.EscalationState, &state))
	require.Equal(t, 1, state.CurrentStep)
}
