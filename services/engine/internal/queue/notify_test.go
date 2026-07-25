package queue_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"

	emailchannel "github.com/mdg-labs/escalite/services/engine/channels/email"
	_ "github.com/mdg-labs/escalite/services/engine/channelsinstall"
	"github.com/mdg-labs/escalite/services/engine/internal/db"
	engineemail "github.com/mdg-labs/escalite/services/engine/internal/email"
	"github.com/mdg-labs/escalite/services/engine/internal/jobs"
	"github.com/mdg-labs/escalite/services/engine/internal/queue"
	"github.com/mdg-labs/escalite/services/engine/internal/testutil"
	"github.com/riverqueue/river"
)

type failingSender struct{}

func (failingSender) Send(context.Context, engineemail.Message) error {
	return errSMTPDown
}

var errSMTPDown = &smtpDownError{}

type smtpDownError struct{}

func (e *smtpDownError) Error() string { return "smtp connection refused" }

func TestNotifyWorkerRecordsFailedAttemptOnSendError(t *testing.T) {
	ctx := context.Background()
	databaseURL, cleanup, err := testutil.StartPostgres(ctx)
	require.NoError(t, err)
	defer cleanup()

	require.NoError(t, testutil.MigrateUp(ctx, databaseURL, slog.Default()))

	emailchannel.SetSender(failingSender{})
	t.Cleanup(func() { emailchannel.SetSender(nil) })

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
	alertID := uuid.Must(uuid.NewV7())
	attemptID := uuid.Must(uuid.NewV7())

	_, err = queries.BootstrapOrganizationWithAdmin(ctx, db.BootstrapOrganizationWithAdminParams{
		OrgID:        orgID,
		OrgName:      "Notify Org",
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
		Name:           "API",
	})
	require.NoError(t, err)

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
	require.NoError(t, err)

	recipient, err := json.Marshal(map[string]string{
		"type":  "user",
		"email": "admin@example.com",
	})
	require.NoError(t, err)

	_, err = queries.CreateNotificationAttempt(ctx, db.CreateNotificationAttemptParams{
		ID:               attemptID,
		OrganizationID:   orgID,
		AlertID:          alertID,
		EscalationStepID: pgtype.UUID{},
		Channel:          "email",
		Status:           "pending",
		Recipient:        recipient,
	})
	require.NoError(t, err)

	worker := queue.NewNotifyWorker(slog.Default(), queueClient.Pool())
	err = worker.Work(ctx, &river.Job[jobs.NotifyArgs]{
		Args: jobs.NotifyArgs{
			NotificationAttemptID: attemptID,
			OrganizationID:      orgID,
		},
	})
	require.NoError(t, err)

	attempt, err := queries.GetNotificationAttemptByID(ctx, db.GetNotificationAttemptByIDParams{
		ID:             attemptID,
		OrganizationID: orgID,
	})
	require.NoError(t, err)
	require.Equal(t, "failed", attempt.Status)
	require.True(t, attempt.ErrorMessage.Valid)
	require.Contains(t, attempt.ErrorMessage.String, "smtp connection refused")
}
