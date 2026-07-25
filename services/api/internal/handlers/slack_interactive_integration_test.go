package handlers_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
	"github.com/mdg-labs/escalite/services/api/internal/server"
	"github.com/mdg-labs/escalite/services/engine/channels/slackdm"
)

const testSlackSigningSecret = "test-slack-signing-secret"

func newTestHandlerWithSlackInteractive(t *testing.T) (http.Handler, *pgxpool.Pool, func()) {
	t.Helper()

	return newTestHandlerWithOptions(t, testServerOptions{
		SlackInteractive: &server.SlackInteractiveOptions{
			SigningSecret: testSlackSigningSecret,
		},
	})
}

func signSlackInteractiveRequest(t *testing.T, body []byte) *http.Request {
	t.Helper()

	timestamp := handlers.SlackInteractiveTimestamp()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/slack/interactive", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Slack-Request-Timestamp", timestamp)
	req.Header.Set("X-Slack-Signature", handlers.SignSlackBody(testSlackSigningSecret, timestamp, body))
	return req
}

func seedSlackWorkspace(t *testing.T, pool db.DBTX, orgID uuid.UUID, workspaceID, slackUserID string, userID uuid.UUID) {
	t.Helper()

	queries := db.New(pool)
	secrets := testSecretsBox(t)

	ciphertext, err := secrets.Encrypt([]byte("xoxb-test-token"))
	require.NoError(t, err)

	_, err = queries.UpsertOrganizationSlackOAuthInstall(context.Background(), db.UpsertOrganizationSlackOAuthInstallParams{
		OrganizationID:     orgID,
		BotTokenCiphertext: ciphertext.Ciphertext,
		EncryptionKeyID:    ciphertext.KeyID,
		TokenHint:          "oken",
		WorkspaceID:        pgtype.Text{String: workspaceID, Valid: true},
		WorkspaceName:      pgtype.Text{String: "Test Workspace", Valid: true},
		BotUserID:          pgtype.Text{String: "B123", Valid: true},
		Scope:              pgtype.Text{String: "chat:write", Valid: true},
	})
	require.NoError(t, err)

	_, err = queries.UpsertUserContactMethod(context.Background(), db.UpsertUserContactMethodParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		UserID:         userID,
		Channel:        "slack-dm",
		Config:         []byte(`{"slack_user_id":"` + slackUserID + `"}`),
	})
	require.NoError(t, err)
}

func seedIncidentLinkedAlert(t *testing.T, pool db.DBTX, orgID, teamID, serviceID, userID uuid.UUID) (db.Alert, db.Incident) {
	t.Helper()

	queries := db.New(pool)
	incidentID := uuid.Must(uuid.NewV7())
	incident, err := queries.CreateIncident(context.Background(), db.CreateIncidentParams{
		ID:              incidentID,
		OrganizationID:  orgID,
		TeamID:          teamID,
		Title:           "Checkout degradation",
		CreatedByUserID: userID,
	})
	require.NoError(t, err)

	alert := seedTriggeredAlert(t, pool, orgID, serviceID, "slack-interactive-ack")
	alert, err = queries.AssignAlertToIncident(context.Background(), db.AssignAlertToIncidentParams{
		ID:             alert.ID,
		OrganizationID: orgID,
		IncidentID:     pgtype.UUID{Bytes: incident.ID, Valid: true},
	})
	require.NoError(t, err)

	return alert, incident
}

func buildSlackAckPayload(t *testing.T, workspaceID, slackUserID string, alertID uuid.UUID) []byte {
	t.Helper()

	payload := map[string]any{
		"type": "block_actions",
		"user": map[string]string{"id": slackUserID},
		"team": map[string]string{"id": workspaceID},
		"actions": []map[string]string{
			{
				"action_id": slackdm.ActionAck,
				"value":     slackdm.ButtonValue(alertID.String()),
			},
		},
	}
	body, err := handlers.BuildSlackInteractiveFormBody(payload)
	require.NoError(t, err)
	return body
}

func TestSlackInteractiveRouteDisabledWithoutSigningSecret(t *testing.T) {
	handler, _, cleanup := newTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/slack/interactive", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestSlackInteractiveRejectsInvalidSignature(t *testing.T) {
	handler, _, cleanup := newTestHandlerWithSlackInteractive(t)
	defer cleanup()

	body := buildSlackAckPayload(t, "T123", "U123", uuid.Must(uuid.NewV7()))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/slack/interactive", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Slack-Request-Timestamp", handlers.SlackInteractiveTimestamp())
	req.Header.Set("X-Slack-Signature", "v0=invalid")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestSlackInteractiveAcknowledgesIncidentLinkedAlert(t *testing.T) {
	handler, pool, cleanup := newTestHandlerWithSlackInteractive(t)
	defer cleanup()

	_ = bootstrapAdmin(t, handler)

	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	const workspaceID = "TWORKSPACE123"
	const slackUserID = "USLACKUSER123"
	seedSlackWorkspace(t, pool, admin.OrganizationID, workspaceID, slackUserID, admin.ID)

	team := seedTeam(t, pool, admin.OrganizationID, "Platform")
	service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")
	alert, _ := seedIncidentLinkedAlert(t, pool, admin.OrganizationID, team.ID, service.ID, admin.ID)

	body := buildSlackAckPayload(t, workspaceID, slackUserID, alert.ID)
	req := signSlackInteractiveRequest(t, body)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	stored, err := queries.GetAlertByID(context.Background(), db.GetAlertByIDParams{
		ID:             alert.ID,
		OrganizationID: admin.OrganizationID,
	})
	require.NoError(t, err)
	require.Equal(t, "acknowledged", stored.Status)
	require.True(t, stored.AcknowledgedByUserID.Valid)
	require.Equal(t, admin.ID, uuid.UUID(stored.AcknowledgedByUserID.Bytes))
}

func TestVerifySlackSignatureAcceptsValidRequest(t *testing.T) {
	body := []byte(`payload=%7B%22type%22%3A%22block_actions%22%7D`)
	timestamp := handlers.SlackInteractiveTimestamp()
	signature := handlers.SignSlackBody(testSlackSigningSecret, timestamp, body)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("X-Slack-Request-Timestamp", timestamp)
	req.Header.Set("X-Slack-Signature", signature)

	err := handlers.VerifySlackSignature(testSlackSigningSecret, body, req.Header)
	require.NoError(t, err)
}

func TestVerifySlackSignatureRejectsTamperedBody(t *testing.T) {
	body := []byte(`payload=%7B%22type%22%3A%22block_actions%22%7D`)
	timestamp := handlers.SlackInteractiveTimestamp()
	signature := handlers.SignSlackBody(testSlackSigningSecret, timestamp, body)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("payload=tampered")))
	req.Header.Set("X-Slack-Request-Timestamp", timestamp)
	req.Header.Set("X-Slack-Signature", signature)

	err := handlers.VerifySlackSignature(testSlackSigningSecret, []byte("payload=tampered"), req.Header)
	require.Error(t, err)
	require.ErrorIs(t, err, handlers.ErrInvalidSlackSignature)
}
