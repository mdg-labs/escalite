package queue_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/allure-framework/allure-go/testify/require"

	emailchannel "github.com/mdg-labs/escalite/services/engine/channels/email"
	pushchannel "github.com/mdg-labs/escalite/services/engine/channels/push"
	webhookchannel "github.com/mdg-labs/escalite/services/engine/channels/webhook"
	_ "github.com/mdg-labs/escalite/services/engine/channelsinstall"
	"github.com/mdg-labs/escalite/services/engine/internal/db"
	engineemail "github.com/mdg-labs/escalite/services/engine/internal/email"
	"github.com/mdg-labs/escalite/services/engine/internal/jobs"
	"github.com/mdg-labs/escalite/services/engine/internal/queue"
	"github.com/mdg-labs/escalite/services/engine/internal/testutil"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

type failingSender struct{}

func (failingSender) Send(context.Context, engineemail.Message) error {
	return errSMTPDown
}

var errSMTPDown = &smtpDownError{}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) Do(req *http.Request) (*http.Response, error) {
	return fn(req)
}

type smtpDownError struct{}

func (e *smtpDownError) Error() string { return "smtp connection refused" }

func TestNotifyWorkerRecordsFailedAttemptOnSendError(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		ctx := context.Background()
		databaseURL, cleanup, err := testutil.StartPostgres(ctx)
		require.NoError(a, err)
		defer cleanup()

		require.NoError(a, testutil.MigrateUp(ctx, databaseURL, slog.Default()))

		emailchannel.SetSender(failingSender{})
		a.T().Cleanup(func() { emailchannel.SetSender(nil) })

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
		alertID := uuid.Must(uuid.NewV7())
		attemptID := uuid.Must(uuid.NewV7())

		_, err = queries.BootstrapOrganizationWithAdmin(ctx, db.BootstrapOrganizationWithAdminParams{
			OrgID:        orgID,
			OrgName:      "Notify Org",
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
			Name:           "API",
		})
		require.NoError(a, err)

		_, err = queries.CreateTriggeredAlert(ctx, db.CreateTriggeredAlertParams{
			ID:              alertID,
			OrganizationID:  orgID,
			ServiceID:       serviceID,
			DedupKey:        "disk-full",
			Summary:         "Disk full",
			Description:     pgtype.Text{String: "Volume /data is full", Valid: true},
			Priority:        "high",
			EscalationState: []byte(`{}`),
		})
		require.NoError(a, err)

		recipient, err := json.Marshal(map[string]string{
			"type":  "user",
			"email": "admin@example.com",
		})
		require.NoError(a, err)

		_, err = queries.CreateNotificationAttempt(ctx, db.CreateNotificationAttemptParams{
			ID:               attemptID,
			OrganizationID:   orgID,
			AlertID:          alertID,
			EscalationStepID: pgtype.UUID{},
			Channel:          "email",
			Status:           "pending",
			Recipient:        recipient,
		})
		require.NoError(a, err)

		worker := queue.NewNotifyWorker(slog.Default(), queueClient.Pool(), nil)
		err = worker.Work(ctx, &river.Job[jobs.NotifyArgs]{
			JobRow: &rivertype.JobRow{
				Attempt:     1,
				MaxAttempts: jobs.NotifyMaxAttempts,
			},
			Args: jobs.NotifyArgs{
				NotificationAttemptID: attemptID,
				OrganizationID:      orgID,
			},
		})
		require.NoError(a, err)

		attempt, err := queries.GetNotificationAttemptByID(ctx, db.GetNotificationAttemptByIDParams{
			ID:             attemptID,
			OrganizationID: orgID,
		})
		require.NoError(a, err)
		require.Equal(a, "failed", attempt.Status)
		require.True(a, attempt.ErrorMessage.Valid)
		require.Contains(a, attempt.ErrorMessage.String, "smtp connection refused")
	})
}

func TestNotifyWorkerDeliversWebhookAttempt(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		ctx := context.Background()
		databaseURL, cleanup, err := testutil.StartPostgres(ctx)
		require.NoError(a, err)
		defer cleanup()

		require.NoError(a, testutil.MigrateUp(ctx, databaseURL, slog.Default()))

		var received struct {
			AlertID string `json:"alert_id"`
			Service string `json:"service"`
			Status  string `json:"status"`
			Title   string `json:"title"`
			Body    string `json:"body"`
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.NoError(a, json.NewDecoder(r.Body).Decode(&received))
			w.WriteHeader(http.StatusOK)
		}))
		a.T().Cleanup(server.Close)

		webhookchannel.SetHTTPClient(server.Client())
		a.T().Cleanup(func() { webhookchannel.SetHTTPClient(nil) })

		queueClient, err := queue.New(ctx, queue.Options{
			DatabaseURL: databaseURL,
			Logger:      slog.Default(),
		})
		require.NoError(a, err)
		defer queueClient.Close()

		queries := db.New(queueClient.Pool())
		orgID := uuid.Must(uuid.NewV7())
		adminID := uuid.Must(uuid.NewV7())
		teamID := uuid.Must(uuid.NewV7())
		serviceID := uuid.Must(uuid.NewV7())
		alertID := uuid.Must(uuid.NewV7())
		attemptID := uuid.Must(uuid.NewV7())

		_, err = queries.BootstrapOrganizationWithAdmin(ctx, db.BootstrapOrganizationWithAdminParams{
			OrgID:        orgID,
			OrgName:      "Webhook Org",
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

		_, err = queries.CreateTriggeredAlert(ctx, db.CreateTriggeredAlertParams{
			ID:              alertID,
			OrganizationID:  orgID,
			ServiceID:       serviceID,
			DedupKey:        "disk-full",
			Summary:         "Disk full",
			Description:     pgtype.Text{String: "Volume /data is full", Valid: true},
			Priority:        "high",
			EscalationState: []byte(`{}`),
		})
		require.NoError(a, err)

		recipient, err := json.Marshal(map[string]string{
			"type": "webhook",
			"url":  server.URL,
		})
		require.NoError(a, err)

		_, err = queries.CreateNotificationAttempt(ctx, db.CreateNotificationAttemptParams{
			ID:               attemptID,
			OrganizationID:   orgID,
			AlertID:          alertID,
			EscalationStepID: pgtype.UUID{},
			Channel:          "webhook",
			Status:           "pending",
			Recipient:        recipient,
		})
		require.NoError(a, err)

		worker := queue.NewNotifyWorker(slog.Default(), queueClient.Pool(), nil)
		err = worker.Work(ctx, &river.Job[jobs.NotifyArgs]{
			Args: jobs.NotifyArgs{
				NotificationAttemptID: attemptID,
				OrganizationID:      orgID,
			},
		})
		require.NoError(a, err)

		attempt, err := queries.GetNotificationAttemptByID(ctx, db.GetNotificationAttemptByIDParams{
			ID:             attemptID,
			OrganizationID: orgID,
		})
		require.NoError(a, err)
		require.Equal(a, "sent", attempt.Status)
		require.Equal(a, alertID.String(), received.AlertID)
		require.Equal(a, "checkout-api", received.Service)
		require.Equal(a, "triggered", received.Status)
		require.Equal(a, "Disk full", received.Title)
		require.Equal(a, "Volume /data is full", received.Body)
	})
}

func TestNotifyWorkerMarksPushFailedOnInvalidExpoToken(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		ctx := context.Background()
		databaseURL, cleanup, err := testutil.StartPostgres(ctx)
		require.NoError(a, err)
		defer cleanup()

		require.NoError(a, testutil.MigrateUp(ctx, databaseURL, slog.Default()))

		pushchannel.SetHTTPClient(roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"data": [{
						"status": "error",
						"message": "\"ExponentPushToken[invalid]\" is not a registered push notification recipient",
						"details": { "error": "DeviceNotRegistered" }
					}]
				}`)),
				Header: make(http.Header),
			}, nil
		}))
		a.T().Cleanup(func() { pushchannel.SetHTTPClient(nil) })

		queueClient, err := queue.New(ctx, queue.Options{
			DatabaseURL: databaseURL,
			Logger:      slog.Default(),
		})
		require.NoError(a, err)
		defer queueClient.Close()

		queries := db.New(queueClient.Pool())
		orgID := uuid.Must(uuid.NewV7())
		adminID := uuid.Must(uuid.NewV7())
		teamID := uuid.Must(uuid.NewV7())
		serviceID := uuid.Must(uuid.NewV7())
		alertID := uuid.Must(uuid.NewV7())
		attemptID := uuid.Must(uuid.NewV7())

		_, err = queries.BootstrapOrganizationWithAdmin(ctx, db.BootstrapOrganizationWithAdminParams{
			OrgID:        orgID,
			OrgName:      "Push Org",
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
			Name:           "API",
		})
		require.NoError(a, err)

		_, err = queries.CreateTriggeredAlert(ctx, db.CreateTriggeredAlertParams{
			ID:              alertID,
			OrganizationID:  orgID,
			ServiceID:       serviceID,
			DedupKey:        "disk-full",
			Summary:         "Disk full",
			Description:     pgtype.Text{String: "Volume /data is full", Valid: true},
			Priority:        "high",
			EscalationState: []byte(`{}`),
		})
		require.NoError(a, err)

		recipient, err := json.Marshal(map[string]any{
			"type":   "user",
			"user_id": adminID.String(),
			"config": map[string]string{
				"expo_push_token": "ExponentPushToken[invalid]",
			},
		})
		require.NoError(a, err)

		_, err = queries.CreateNotificationAttempt(ctx, db.CreateNotificationAttemptParams{
			ID:               attemptID,
			OrganizationID:   orgID,
			AlertID:          alertID,
			EscalationStepID: pgtype.UUID{},
			Channel:          "push",
			Status:           "pending",
			Recipient:        recipient,
		})
		require.NoError(a, err)

		worker := queue.NewNotifyWorker(slog.Default(), queueClient.Pool(), nil)
		err = worker.Work(ctx, &river.Job[jobs.NotifyArgs]{
			JobRow: &rivertype.JobRow{
				Attempt:     1,
				MaxAttempts: jobs.NotifyMaxAttempts,
			},
			Args: jobs.NotifyArgs{
				NotificationAttemptID: attemptID,
				OrganizationID:      orgID,
			},
		})
		require.NoError(a, err)

		attempt, err := queries.GetNotificationAttemptByID(ctx, db.GetNotificationAttemptByIDParams{
			ID:             attemptID,
			OrganizationID: orgID,
		})
		require.NoError(a, err)
		require.Equal(a, "failed", attempt.Status)
		require.True(a, attempt.ErrorMessage.Valid)
		require.Contains(a, attempt.ErrorMessage.String, "DeviceNotRegistered")
	})
}
