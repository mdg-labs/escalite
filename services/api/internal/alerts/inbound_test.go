package alerts_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/mdg-labs/escalite/services/api/internal/alerts"
	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/migrate"
	"github.com/mdg-labs/escalite/services/api/internal/queue"
	"github.com/mdg-labs/escalite/services/integrations"
)

func startPostgres(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	ctx := context.Background()

	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("escalite"),
		postgres.WithUsername("escalite"),
		postgres.WithPassword("escalite"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)

	databaseURL, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	require.NoError(t, migrate.Up(ctx, databaseURL, slog.Default()))

	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)

	cleanup := func() {
		pool.Close()
		require.NoError(t, testcontainers.TerminateContainer(container))
	}

	return pool, cleanup
}

type inboundFixture struct {
	queries *db.Queries
	key     db.IntegrationKey
	userID  uuid.UUID
}

func seedInboundFixture(t *testing.T, pool *pgxpool.Pool) inboundFixture {
	t.Helper()

	ctx := context.Background()
	queries := db.New(pool)

	orgID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())
	_, err := queries.BootstrapOrganizationWithAdmin(ctx, db.BootstrapOrganizationWithAdminParams{
		OrgID:        orgID,
		OrgName:      "Acme",
		UserID:       userID,
		Email:        "admin@example.com",
		PasswordHash: pgtype.Text{String: "argon2id:test", Valid: true},
	})
	require.NoError(t, err)

	team, err := queries.CreateTeam(ctx, db.CreateTeamParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		Name:           "Platform",
	})
	require.NoError(t, err)

	service, err := queries.CreateService(ctx, db.CreateServiceParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		TeamID:         team.ID,
		Name:           "checkout-api",
	})
	require.NoError(t, err)

	plaintext, hash, prefix, err := auth.NewIntegrationKeyToken()
	require.NoError(t, err)

	keyID := uuid.Must(uuid.NewV7())
	_, err = pool.Exec(ctx, `
		INSERT INTO integration_keys (
			id, service_id, organization_id, token, prefix, plugin_name, config
		) VALUES ($1, $2, $3, $4, $5, $6, '{}'::jsonb)
	`, keyID, service.ID, orgID, hash, prefix, "generic-rest-api")
	require.NoError(t, err)

	key, err := queries.GetActiveIntegrationKeyByTokenHash(ctx, hash)
	require.NoError(t, err)
	require.Equal(t, plaintext[:len(prefix)], key.Prefix)

	return inboundFixture{
		queries: queries,
		key:     key,
		userID:  userID,
	}
}

func newInboundJobs(t *testing.T, pool *pgxpool.Pool) *queue.Producer {
	t.Helper()
	jobs, err := queue.NewProducer(context.Background(), pool, slog.Default())
	require.NoError(t, err)
	return jobs
}

func seedEscalationPolicyWithUsers(
	t *testing.T,
	pool *pgxpool.Pool,
	fixture inboundFixture,
	oncallID uuid.UUID,
) uuid.UUID {
	t.Helper()

	ctx := context.Background()
	queries := fixture.queries

	policyID := uuid.Must(uuid.NewV7())
	stepID := uuid.Must(uuid.NewV7())

	_, err := queries.CreateEscalationPolicy(ctx, db.CreateEscalationPolicyParams{
		ID:             policyID,
		OrganizationID: fixture.key.OrganizationID,
		ServiceID:      fixture.key.ServiceID,
		Name:           "Default",
	})
	require.NoError(t, err)

	_, err = queries.CreateEscalationStep(ctx, db.CreateEscalationStepParams{
		ID:                 stepID,
		EscalationPolicyID: policyID,
		OrganizationID:     fixture.key.OrganizationID,
		StepOrder:          1,
		DelayMinutes:       0,
		RepeatLastStep:     false,
		MaxRepeats:         pgtype.Int4{},
	})
	require.NoError(t, err)

	adminChannels, err := json.Marshal([]string{"email"})
	require.NoError(t, err)
	_, err = queries.CreateEscalationStepTarget(ctx, db.CreateEscalationStepTargetParams{
		ID:               uuid.Must(uuid.NewV7()),
		EscalationStepID: stepID,
		OrganizationID:   fixture.key.OrganizationID,
		TargetType:       "user",
		UserID:           pgtype.UUID{Bytes: fixture.userID, Valid: true},
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
		OrganizationID:   fixture.key.OrganizationID,
		TargetType:       "user",
		UserID:           pgtype.UUID{Bytes: oncallID, Valid: true},
		ScheduleID:       pgtype.UUID{},
		WebhookUrl:       pgtype.Text{},
		Channels:         oncallChannels,
	})
	require.NoError(t, err)

	return stepID
}

func TestProcessInboundCollapseIncrementsEventCount(t *testing.T) {
	pool, cleanup := startPostgres(t)
	defer cleanup()

	fixture := seedInboundFixture(t, pool)
	ctx := context.Background()
	logger := slog.Default()

	event := integrations.AlertCreate{
		EventType:   integrations.EventTriggered,
		DedupKey:    "host-1-disk",
		Summary:     "Disk usage high",
		Description: "Volume /data is 95% full",
		Priority:    "low",
	}

	firstID, err := alerts.ProcessInbound(ctx, fixture.queries, logger, fixture.key, event, nil)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, firstID)

	secondID, err := alerts.ProcessInbound(ctx, fixture.queries, logger, fixture.key, event, nil)
	require.NoError(t, err)
	require.Equal(t, firstID, secondID)

	alert, err := fixture.queries.GetAlertByID(ctx, db.GetAlertByIDParams{
		ID:             firstID,
		OrganizationID: fixture.key.OrganizationID,
	})
	require.NoError(t, err)
	require.Equal(t, int32(2), alert.EventCount)
	require.Equal(t, "triggered", alert.Status)

	var alertCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM alerts
		WHERE service_id = $1 AND dedup_key = $2
	`, fixture.key.ServiceID, event.DedupKey).Scan(&alertCount)
	require.NoError(t, err)
	require.Equal(t, 1, alertCount)
}

func TestProcessInboundCollapseAcknowledgedAlertDoesNotCreateNotifications(t *testing.T) {
	pool, cleanup := startPostgres(t)
	defer cleanup()

	fixture := seedInboundFixture(t, pool)
	jobs := newInboundJobs(t, pool)
	oncallID := uuid.Must(uuid.NewV7())
	_, err := fixture.queries.CreateUser(context.Background(), db.CreateUserParams{
		ID:             oncallID,
		OrganizationID: fixture.key.OrganizationID,
		Email:          "oncall@example.com",
		PasswordHash:   pgtype.Text{String: "argon2id:test", Valid: true},
		Role:           "member",
	})
	require.NoError(t, err)
	seedEscalationPolicyWithUsers(t, pool, fixture, oncallID)

	ctx := context.Background()
	logger := slog.Default()
	deps := &alerts.InboundDeps{Pool: pool, Jobs: jobs}

	event := integrations.AlertCreate{
		EventType: integrations.EventTriggered,
		DedupKey:  "memory-high",
		Summary:   "Memory above threshold",
		Priority:  "high",
	}

	alertID, err := alerts.ProcessInbound(ctx, fixture.queries, logger, fixture.key, event, nil)
	require.NoError(t, err)

	_, err = fixture.queries.AcknowledgeAlert(ctx, db.AcknowledgeAlertParams{
		ID:                   alertID,
		OrganizationID:       fixture.key.OrganizationID,
		EscalationState:      []byte(`{"current_step":1}`),
		AcknowledgedByUserID: pgtype.UUID{Bytes: fixture.userID, Valid: true},
	})
	require.NoError(t, err)

	_, err = fixture.queries.CreateNotificationAttempt(ctx, db.CreateNotificationAttemptParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: fixture.key.OrganizationID,
		AlertID:        alertID,
		Channel:        "email",
		Status:         "sent",
		Recipient:      []byte(`{"type":"user","user_id":"` + fixture.userID.String() + `"}`),
	})
	require.NoError(t, err)
	_, err = fixture.queries.CreateNotificationAttempt(ctx, db.CreateNotificationAttemptParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: fixture.key.OrganizationID,
		AlertID:        alertID,
		Channel:        "email",
		Status:         "sent",
		Recipient:      []byte(`{"type":"user","user_id":"` + oncallID.String() + `"}`),
	})
	require.NoError(t, err)

	beforeCount, err := fixture.queries.CountNotificationAttemptsByAlertID(ctx, db.CountNotificationAttemptsByAlertIDParams{
		AlertID:        alertID,
		OrganizationID: fixture.key.OrganizationID,
	})
	require.NoError(t, err)
	require.Equal(t, int64(2), beforeCount)

	collapsedID, err := alerts.ProcessInbound(ctx, fixture.queries, logger, fixture.key, event, deps)
	require.NoError(t, err)
	require.Equal(t, alertID, collapsedID)

	afterCount, err := fixture.queries.CountNotificationAttemptsByAlertID(ctx, db.CountNotificationAttemptsByAlertIDParams{
		AlertID:        alertID,
		OrganizationID: fixture.key.OrganizationID,
	})
	require.NoError(t, err)
	require.Equal(t, beforeCount+1, afterCount)

	attempts, err := fixture.queries.ListNotificationAttemptsByAlertID(ctx, db.ListNotificationAttemptsByAlertIDParams{
		AlertID:        alertID,
		OrganizationID: fixture.key.OrganizationID,
	})
	require.NoError(t, err)

	adminAttempts := 0
	oncallAttempts := 0
	for _, attempt := range attempts {
		var recipient struct {
			UserID string `json:"user_id"`
		}
		require.NoError(t, json.Unmarshal(attempt.Recipient, &recipient))
		switch recipient.UserID {
		case fixture.userID.String():
			adminAttempts++
		case oncallID.String():
			oncallAttempts++
		}
	}
	require.Equal(t, 1, adminAttempts)
	require.Equal(t, 2, oncallAttempts)

	alert, err := fixture.queries.GetAlertByID(ctx, db.GetAlertByIDParams{
		ID:             alertID,
		OrganizationID: fixture.key.OrganizationID,
	})
	require.NoError(t, err)
	require.Equal(t, "acknowledged", alert.Status)
	require.Equal(t, int32(2), alert.EventCount)
}

func TestProcessInboundOutsideDedupWindowCreatesNewAlertRow(t *testing.T) {
	pool, cleanup := startPostgres(t)
	defer cleanup()

	fixture := seedInboundFixture(t, pool)
	ctx := context.Background()
	logger := slog.Default()

	event := integrations.AlertCreate{
		EventType: integrations.EventTriggered,
		DedupKey:  "host-1-disk",
		Summary:   "Disk usage high",
		Priority:  "low",
	}

	firstID, err := alerts.ProcessInbound(ctx, fixture.queries, logger, fixture.key, event, nil)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, firstID)

	_, err = pool.Exec(ctx, `
		UPDATE alerts
		SET updated_at = now() - interval '10 minutes'
		WHERE id = $1
	`, firstID)
	require.NoError(t, err)

	secondID, err := alerts.ProcessInbound(ctx, fixture.queries, logger, fixture.key, event, nil)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, secondID)
	require.NotEqual(t, firstID, secondID)

	var alertCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM alerts
		WHERE service_id = $1 AND dedup_key = $2
	`, fixture.key.ServiceID, event.DedupKey).Scan(&alertCount)
	require.NoError(t, err)
	require.Equal(t, 2, alertCount)

	firstAlert, err := fixture.queries.GetAlertByID(ctx, db.GetAlertByIDParams{
		ID:             firstID,
		OrganizationID: fixture.key.OrganizationID,
	})
	require.NoError(t, err)
	require.Equal(t, int32(1), firstAlert.EventCount)

	secondAlert, err := fixture.queries.GetAlertByID(ctx, db.GetAlertByIDParams{
		ID:             secondID,
		OrganizationID: fixture.key.OrganizationID,
	})
	require.NoError(t, err)
	require.Equal(t, int32(1), secondAlert.EventCount)
}

func TestProcessInboundServiceDefaultDedupWindowSeconds(t *testing.T) {
	pool, cleanup := startPostgres(t)
	defer cleanup()

	fixture := seedInboundFixture(t, pool)
	ctx := context.Background()

	service, err := fixture.queries.GetServiceByID(ctx, db.GetServiceByIDParams{
		ID:             fixture.key.ServiceID,
		OrganizationID: fixture.key.OrganizationID,
	})
	require.NoError(t, err)
	require.Equal(t, int32(300), service.DedupWindowSeconds)
}

func TestProcessInboundCustomDedupWindowCollapsesWithinWindow(t *testing.T) {
	pool, cleanup := startPostgres(t)
	defer cleanup()

	fixture := seedInboundFixture(t, pool)
	ctx := context.Background()
	logger := slog.Default()

	_, err := pool.Exec(ctx, `
		UPDATE services
		SET dedup_window_seconds = 600
		WHERE id = $1
	`, fixture.key.ServiceID)
	require.NoError(t, err)

	event := integrations.AlertCreate{
		EventType: integrations.EventTriggered,
		DedupKey:  "cpu-spike",
		Summary:   "CPU above threshold",
		Priority:  "high",
	}

	firstID, err := alerts.ProcessInbound(ctx, fixture.queries, logger, fixture.key, event, nil)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		UPDATE alerts
		SET updated_at = now() - interval '7 minutes'
		WHERE id = $1
	`, firstID)
	require.NoError(t, err)

	secondID, err := alerts.ProcessInbound(ctx, fixture.queries, logger, fixture.key, event, nil)
	require.NoError(t, err)
	require.Equal(t, firstID, secondID)

	alert, err := fixture.queries.GetAlertByID(ctx, db.GetAlertByIDParams{
		ID:             firstID,
		OrganizationID: fixture.key.OrganizationID,
	})
	require.NoError(t, err)
	require.Equal(t, int32(2), alert.EventCount)
}

func TestProcessInboundCustomDedupWindowOutsideCreatesNewRow(t *testing.T) {
	pool, cleanup := startPostgres(t)
	defer cleanup()

	fixture := seedInboundFixture(t, pool)
	ctx := context.Background()
	logger := slog.Default()

	_, err := pool.Exec(ctx, `
		UPDATE services
		SET dedup_window_seconds = 120
		WHERE id = $1
	`, fixture.key.ServiceID)
	require.NoError(t, err)

	event := integrations.AlertCreate{
		EventType: integrations.EventTriggered,
		DedupKey:  "memory-leak",
		Summary:   "Memory climbing",
		Priority:  "high",
	}

	firstID, err := alerts.ProcessInbound(ctx, fixture.queries, logger, fixture.key, event, nil)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		UPDATE alerts
		SET updated_at = now() - interval '3 minutes'
		WHERE id = $1
	`, firstID)
	require.NoError(t, err)

	secondID, err := alerts.ProcessInbound(ctx, fixture.queries, logger, fixture.key, event, nil)
	require.NoError(t, err)
	require.NotEqual(t, firstID, secondID)

	var alertCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM alerts
		WHERE service_id = $1 AND dedup_key = $2
	`, fixture.key.ServiceID, event.DedupKey).Scan(&alertCount)
	require.NoError(t, err)
	require.Equal(t, 2, alertCount)
}

func TestProcessInboundResolveSetsResolvedAtAndIntegration(t *testing.T) {
	pool, cleanup := startPostgres(t)
	defer cleanup()

	fixture := seedInboundFixture(t, pool)
	ctx := context.Background()
	logger := slog.Default()

	triggered := integrations.AlertCreate{
		EventType: integrations.EventTriggered,
		DedupKey:  "host-1-disk",
		Summary:   "Disk usage high",
		Priority:  "low",
	}
	alertID, err := alerts.ProcessInbound(ctx, fixture.queries, logger, fixture.key, triggered, nil)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, alertID)

	resolved := integrations.AlertCreate{
		EventType: integrations.EventResolved,
		DedupKey:  "host-1-disk",
	}
	_, err = alerts.ProcessInbound(ctx, fixture.queries, logger, fixture.key, resolved, nil)
	require.NoError(t, err)

	alert, err := fixture.queries.GetAlertByID(ctx, db.GetAlertByIDParams{
		ID:             alertID,
		OrganizationID: fixture.key.OrganizationID,
	})
	require.NoError(t, err)
	require.Equal(t, "closed", alert.Status)
	require.True(t, alert.ClosedAt.Valid)
	require.True(t, alert.ResolvedAt.Valid)
	require.True(t, alert.ResolvedIntegration.Valid)
	require.Equal(t, "generic-rest-api", alert.ResolvedIntegration.String)
}

func TestProcessInboundResolveUnknownDedupKeyNoOp(t *testing.T) {
	pool, cleanup := startPostgres(t)
	defer cleanup()

	fixture := seedInboundFixture(t, pool)
	ctx := context.Background()
	logger := slog.Default()

	resolved := integrations.AlertCreate{
		EventType: integrations.EventResolved,
		DedupKey:  "missing-key",
	}
	_, err := alerts.ProcessInbound(ctx, fixture.queries, logger, fixture.key, resolved, nil)
	require.NoError(t, err)

	var alertCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM alerts
		WHERE service_id = $1 AND dedup_key = $2
	`, fixture.key.ServiceID, resolved.DedupKey).Scan(&alertCount)
	require.NoError(t, err)
	require.Equal(t, 0, alertCount)
}

func TestProcessInboundResolveIdempotentWhenAlreadyClosed(t *testing.T) {
	pool, cleanup := startPostgres(t)
	defer cleanup()

	fixture := seedInboundFixture(t, pool)
	ctx := context.Background()
	logger := slog.Default()

	event := integrations.AlertCreate{
		EventType: integrations.EventTriggered,
		DedupKey:  "cpu-high",
		Summary:   "CPU above threshold",
		Priority:  "high",
	}
	alertID, err := alerts.ProcessInbound(ctx, fixture.queries, logger, fixture.key, event, nil)
	require.NoError(t, err)

	resolved := integrations.AlertCreate{
		EventType: integrations.EventResolved,
		DedupKey:  "cpu-high",
	}
	_, err = alerts.ProcessInbound(ctx, fixture.queries, logger, fixture.key, resolved, nil)
	require.NoError(t, err)

	_, err = alerts.ProcessInbound(ctx, fixture.queries, logger, fixture.key, resolved, nil)
	require.NoError(t, err)

	alert, err := fixture.queries.GetAlertByID(ctx, db.GetAlertByIDParams{
		ID:             alertID,
		OrganizationID: fixture.key.OrganizationID,
	})
	require.NoError(t, err)
	require.Equal(t, "closed", alert.Status)
	require.True(t, alert.ResolvedAt.Valid)
}
