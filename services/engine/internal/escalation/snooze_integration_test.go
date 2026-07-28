package escalation_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"
	"time"

	"github.com/google/uuid"
	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/engine/internal/db"
	"github.com/mdg-labs/escalite/services/engine/internal/escalation"
	"github.com/mdg-labs/escalite/services/engine/internal/queue"
	"github.com/mdg-labs/escalite/services/engine/internal/testutil"
)

func TestSnoozeEscalationShiftsNextEscalationAt(t *testing.T) {
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
		require.NoError(a, err)

		waitForNotificationCount(t, ctx, queries, alertID, fixture.orgID, 1)

		beforeState := waitForPendingEscalationJob(t, ctx, queries, alertID, fixture.orgID)
		require.NotNil(a, beforeState.NextEscalationAt)

		snoozeMinutes := int32(15)
		_, err = escalation.SnoozeEscalation(ctx, queries, queueClient, queueClient, alertID, fixture.orgID, snoozeMinutes)
		require.NoError(a, err)

		after, err := queries.GetAlertByID(ctx, db.GetAlertByIDParams{
			ID:             alertID,
			OrganizationID: fixture.orgID,
		})
		require.NoError(a, err)

		var afterState escalation.State
		require.NoError(a, json.Unmarshal(after.EscalationState, &afterState))
		require.NotNil(a, afterState.NextEscalationAt)
		require.NotNil(a, afterState.PendingEscalationJobID)

		expected := beforeState.NextEscalationAt.Add(time.Duration(snoozeMinutes) * time.Minute)
		require.WithinDuration(a, expected, *afterState.NextEscalationAt, 2*time.Second)
	})
}

func TestReEscalateAlertResetsToStep1AndEnqueuesNotifications(t *testing.T) {
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
		require.NoError(a, err)

		waitForNotificationCount(t, ctx, queries, alertID, fixture.orgID, 1)

		acknowledged, err := escalation.AcknowledgeAlert(ctx, queries, queueClient, alertID, fixture.orgID, fixture.adminID)
		require.NoError(a, err)
		require.Equal(a, "acknowledged", acknowledged.Status)

		_, err = escalation.ReEscalateAlert(ctx, queries, queueClient, queueClient, alertID, fixture.orgID)
		require.NoError(a, err)

		waitForNotificationCount(t, ctx, queries, alertID, fixture.orgID, 2)

		reEscalated, err := queries.GetAlertByID(ctx, db.GetAlertByIDParams{
			ID:             alertID,
			OrganizationID: fixture.orgID,
		})
		require.NoError(a, err)
		require.Equal(a, "triggered", reEscalated.Status)
		require.False(a, reEscalated.AcknowledgedAt.Valid)

		var state escalation.State
		require.NoError(a, json.Unmarshal(reEscalated.EscalationState, &state))
		require.Equal(a, 1, state.CurrentStep)
	})
}
