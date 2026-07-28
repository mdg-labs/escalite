package statuspage_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/engine/internal/db"
	engineemail "github.com/mdg-labs/escalite/services/engine/internal/email"
	"github.com/mdg-labs/escalite/services/engine/internal/statuspage"
)

type recordingSender struct {
	messages []engineemail.Message
}

func (r *recordingSender) Send(_ context.Context, msg engineemail.Message) error {
	r.messages = append(r.messages, msg)
	return nil
}

func TestNotifySubscribersSendsIncidentUpdateEmail(t *testing.T) {
	ctx := context.Background()
	sender := &recordingSender{}

	orgID := uuid.Must(uuid.NewV7())
	pageID := uuid.Must(uuid.NewV7())
	incidentID := uuid.Must(uuid.NewV7())
	updateID := uuid.Must(uuid.NewV7())

	queries := db.New(testQueries(ctx, t, orgID, pageID, incidentID, updateID))

	err := statuspage.NotifySubscribers(ctx, queries, sender, statuspage.NotifyConfig{
		SigningKey: []byte("0123456789abcdef0123456789abcdef"),
	}, statuspage.NotifyParams{
		OrganizationID:       orgID,
		StatusPageIncidentID: incidentID,
		UpdateID:             updateID,
	})
	require.NoError(t, err)
	require.Len(t, sender.messages, 2)

	for _, msg := range sender.messages {
		require.Equal(t, "[Investigating] API outage", msg.Subject)
		require.Contains(t, msg.TextBody, "Acme Status")
		require.Contains(t, msg.TextBody, "API outage")
		require.Contains(t, msg.TextBody, "We are investigating elevated errors.")
		require.Contains(t, msg.TextBody, "Unsubscribe from incident emails:")
		require.Contains(t, msg.TextBody, "/unsubscribe?token=")
		require.Contains(t, msg.HTMLBody, "We are investigating elevated errors.")
		require.Contains(t, msg.HTMLBody, "Unsubscribe from incident emails")
		require.Contains(t, msg.HTMLBody, "/unsubscribe?token=")
	}

	require.Equal(t, "subscriber-a@example.com", sender.messages[0].To)
	require.Equal(t, "subscriber-b@example.com", sender.messages[1].To)
}

func TestNotifySubscribersSkipsWhenSMTPUnset(t *testing.T) {
	ctx := context.Background()

	orgID := uuid.Must(uuid.NewV7())
	pageID := uuid.Must(uuid.NewV7())
	incidentID := uuid.Must(uuid.NewV7())
	updateID := uuid.Must(uuid.NewV7())

	queries := db.New(testQueries(ctx, t, orgID, pageID, incidentID, updateID))

	err := statuspage.NotifySubscribers(ctx, queries, nil, statuspage.NotifyConfig{}, statuspage.NotifyParams{
		OrganizationID:       orgID,
		StatusPageIncidentID: incidentID,
		UpdateID:             updateID,
	})
	require.NoError(t, err)
}

func testQueries(
	ctx context.Context,
	t *testing.T,
	orgID, pageID, incidentID, updateID uuid.UUID,
) db.DBTX {
	t.Helper()

	databaseURL, cleanup, err := startPostgres(ctx)
	require.NoError(t, err)
	t.Cleanup(cleanup)

	pool, err := openPool(ctx, databaseURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	require.NoError(t, migrateUp(ctx, databaseURL))

	queries := db.New(pool)
	adminID := uuid.Must(uuid.NewV7())
	_, err = queries.BootstrapOrganizationWithAdmin(ctx, db.BootstrapOrganizationWithAdminParams{
		OrgID:        orgID,
		OrgName:      "Acme",
		UserID:       adminID,
		Email:        "admin@example.com",
		PasswordHash: pgtype.Text{String: "hash", Valid: true},
	})
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO status_pages (id, organization_id, slug, title, enabled)
		VALUES ($1, $2, 'acme-status', 'Acme Status', true)
	`, pageID, orgID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO status_page_incidents (id, status_page_id, organization_id, title, status)
		VALUES ($1, $2, $3, 'API outage', 'investigating')
	`, incidentID, pageID, orgID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO status_page_incident_updates (
			id, status_page_incident_id, organization_id, body, status
		) VALUES ($1, $2, $3, 'We are investigating elevated errors.', 'investigating')
	`, updateID, incidentID, orgID)
	require.NoError(t, err)

	for _, email := range []string{"subscriber-a@example.com", "subscriber-b@example.com"} {
		_, err = pool.Exec(ctx, `
			INSERT INTO status_page_subscriptions (id, status_page_id, organization_id, email)
			VALUES ($1, $2, $3, $4)
		`, uuid.Must(uuid.NewV7()), pageID, orgID, email)
		require.NoError(t, err)
	}

	return pool
}
