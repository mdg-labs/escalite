package queue_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"
	"github.com/stretchr/testify/require"

	emailchannel "github.com/mdg-labs/escalite/services/engine/channels/email"
	"github.com/mdg-labs/escalite/services/engine/internal/db"
	engineemail "github.com/mdg-labs/escalite/services/engine/internal/email"
	"github.com/mdg-labs/escalite/services/engine/internal/jobs"
	"github.com/mdg-labs/escalite/services/engine/internal/queue"
	"github.com/mdg-labs/escalite/services/engine/internal/testutil"
)

type recordingStatusPageSender struct {
	messages []engineemail.Message
}

func (r *recordingStatusPageSender) Send(_ context.Context, msg engineemail.Message) error {
	r.messages = append(r.messages, msg)
	return nil
}

func TestStatusPageIncidentNotifyWorkerSendsSubscriberEmails(t *testing.T) {
	ctx := context.Background()
	databaseURL, cleanup, err := testutil.StartPostgres(ctx)
	require.NoError(t, err)
	defer cleanup()

	require.NoError(t, testutil.MigrateUp(ctx, databaseURL, slog.Default()))

	sender := &recordingStatusPageSender{}
	emailchannel.SetSender(sender)
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
	pageID := uuid.Must(uuid.NewV7())
	incidentID := uuid.Must(uuid.NewV7())
	updateID := uuid.Must(uuid.NewV7())
	adminID := uuid.Must(uuid.NewV7())

	_, err = queries.BootstrapOrganizationWithAdmin(ctx, db.BootstrapOrganizationWithAdminParams{
		OrgID:        orgID,
		OrgName:      "Acme",
		UserID:       adminID,
		Email:        "admin@example.com",
		PasswordHash: pgtype.Text{String: "hash", Valid: true},
	})
	require.NoError(t, err)

	_, err = queueClient.Pool().Exec(ctx, `
		INSERT INTO status_pages (id, organization_id, slug, title, enabled)
		VALUES ($1, $2, 'acme-status', 'Acme Status', true)
	`, pageID, orgID)
	require.NoError(t, err)

	_, err = queueClient.Pool().Exec(ctx, `
		INSERT INTO status_page_incidents (id, status_page_id, organization_id, title, status)
		VALUES ($1, $2, $3, 'API outage', 'investigating')
	`, incidentID, pageID, orgID)
	require.NoError(t, err)

	_, err = queueClient.Pool().Exec(ctx, `
		INSERT INTO status_page_incident_updates (
			id, status_page_incident_id, organization_id, body, status
		) VALUES ($1, $2, $3, 'We are investigating elevated errors.', 'investigating')
	`, updateID, incidentID, orgID)
	require.NoError(t, err)

	_, err = queueClient.Pool().Exec(ctx, `
		INSERT INTO status_page_subscriptions (id, status_page_id, organization_id, email)
		VALUES ($1, $2, $3, 'subscriber@example.com')
	`, uuid.Must(uuid.NewV7()), pageID, orgID)
	require.NoError(t, err)

	worker := queue.NewStatusPageIncidentNotifyWorker(slog.Default(), queueClient.Pool())
	err = worker.Work(ctx, &river.Job[jobs.StatusPageIncidentNotifyArgs]{
		Args: jobs.StatusPageIncidentNotifyArgs{
			OrganizationID:       orgID,
			StatusPageIncidentID: incidentID,
			UpdateID:             updateID,
		},
	})
	require.NoError(t, err)
	require.Len(t, sender.messages, 1)
	require.Equal(t, "subscriber@example.com", sender.messages[0].To)
	require.Equal(t, "[Investigating] API outage", sender.messages[0].Subject)
	require.Contains(t, sender.messages[0].TextBody, "We are investigating elevated errors.")
}

func TestStatusPageIncidentNotifyWorkerSkipsWhenSMTPUnset(t *testing.T) {
	ctx := context.Background()
	databaseURL, cleanup, err := testutil.StartPostgres(ctx)
	require.NoError(t, err)
	defer cleanup()

	require.NoError(t, testutil.MigrateUp(ctx, databaseURL, slog.Default()))

	emailchannel.SetSender(nil)

	queueClient, err := queue.New(ctx, queue.Options{
		DatabaseURL: databaseURL,
		Logger:      slog.Default(),
	})
	require.NoError(t, err)
	defer queueClient.Close()

	worker := queue.NewStatusPageIncidentNotifyWorker(slog.Default(), queueClient.Pool())
	err = worker.Work(ctx, &river.Job[jobs.StatusPageIncidentNotifyArgs]{
		Args: jobs.StatusPageIncidentNotifyArgs{
			OrganizationID:       uuid.Must(uuid.NewV7()),
			StatusPageIncidentID: uuid.Must(uuid.NewV7()),
			UpdateID:             uuid.Must(uuid.NewV7()),
		},
	})
	require.NoError(t, err)
}
