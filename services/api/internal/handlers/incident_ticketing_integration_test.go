package handlers_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/crypto"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/outboundintegrations/jira"
)

func TestGraphQLPromoteAlertToIncidentCreatesJiraTicket(t *testing.T) {
	jira.SetHTTPClient(slackRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		require.Contains(t, req.URL.String(), "/rest/api/3/issue")

		return &http.Response{
			StatusCode: http.StatusCreated,
			Body:       io.NopCloser(strings.NewReader(`{"id":"10000","key":"OPS-42","self":"https://example.atlassian.net/rest/api/3/issue/10000"}`)),
			Header:     make(http.Header),
		}, nil
	}))
	t.Cleanup(func() { jira.SetHTTPClient(nil) })

	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)
	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	team := seedTeam(t, pool, admin.OrganizationID, "Platform")
	service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")
	alert := seedTriggeredAlert(t, pool, admin.OrganizationID, service.ID, "memory-high")

	seedJiraTicketingSettings(t, queries, admin.OrganizationID)

	promoteRec := postGraphQL(t, handler, `mutation {
		promoteAlertToIncident(input: {
			alertId: "`+alert.ID.String()+`"
			title: "Checkout degradation"
		}) {
			incidentId
		}
	}`, adminCookie)
	require.Equal(t, 200, promoteRec.Code)

	var promoteResp struct {
		Data struct {
			PromoteAlertToIncident struct {
				IncidentID *string `json:"incidentId"`
			} `json:"promoteAlertToIncident"`
		} `json:"data"`
		Errors []any `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(promoteRec.Body.Bytes(), &promoteResp))
	require.Empty(t, promoteResp.Errors)
	require.NotNil(t, promoteResp.Data.PromoteAlertToIncident.IncidentID)

	incidentID := uuid.MustParse(*promoteResp.Data.PromoteAlertToIncident.IncidentID)
	incident, err := queries.GetIncidentByID(context.Background(), db.GetIncidentByIDParams{
		ID:             incidentID,
		OrganizationID: admin.OrganizationID,
	})
	require.NoError(t, err)
	require.True(t, incident.TicketUrl.Valid)
	require.Equal(t, "https://example.atlassian.net/browse/OPS-42", incident.TicketUrl.String)
}

func TestGraphQLPromoteAlertToIncidentSkipsTicketWhenNotConfigured(t *testing.T) {
	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)
	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	team := seedTeam(t, pool, admin.OrganizationID, "Platform")
	service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")
	alert := seedTriggeredAlert(t, pool, admin.OrganizationID, service.ID, "disk-full")

	promoteRec := postGraphQL(t, handler, `mutation {
		promoteAlertToIncident(input: {
			alertId: "`+alert.ID.String()+`"
			title: "Disk pressure"
		}) {
			incidentId
		}
	}`, adminCookie)
	require.Equal(t, 200, promoteRec.Code)

	var promoteResp struct {
		Data struct {
			PromoteAlertToIncident struct {
				IncidentID *string `json:"incidentId"`
			} `json:"promoteAlertToIncident"`
		} `json:"data"`
		Errors []any `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(promoteRec.Body.Bytes(), &promoteResp))
	require.Empty(t, promoteResp.Errors)
	require.NotNil(t, promoteResp.Data.PromoteAlertToIncident.IncidentID)

	incidentID := uuid.MustParse(*promoteResp.Data.PromoteAlertToIncident.IncidentID)
	incident, err := queries.GetIncidentByID(context.Background(), db.GetIncidentByIDParams{
		ID:             incidentID,
		OrganizationID: admin.OrganizationID,
	})
	require.NoError(t, err)
	require.False(t, incident.TicketUrl.Valid)
}

func seedJiraTicketingSettings(t *testing.T, queries *db.Queries, orgID uuid.UUID) {
	t.Helper()

	secrets := testSecretsBox(t)
	const apiToken = "jira-api-token-secret"
	encrypted, err := secrets.Encrypt([]byte(apiToken))
	require.NoError(t, err)

	_, err = queries.UpsertOrganizationTicketingSettings(context.Background(), db.UpsertOrganizationTicketingSettingsParams{
		OrganizationID:     orgID,
		PluginName:         "jira",
		Config:             []byte(`{"base_url":"https://example.atlassian.net","email":"ops@example.com","project_key":"OPS"}`),
		ApiTokenCiphertext: encrypted.Ciphertext,
		EncryptionKeyID:    encrypted.KeyID,
		TokenHint:          crypto.SecretHint(apiToken),
	})
	require.NoError(t, err)
}
