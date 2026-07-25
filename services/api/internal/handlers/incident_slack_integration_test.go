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
	"github.com/mdg-labs/escalite/services/engine/channels/slackchannel"
)

type slackRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn slackRoundTripFunc) Do(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestGraphQLPromoteAlertToIncidentCreatesSlackChannel(t *testing.T) {
	mock := newSlackAPIMock()
	mock.install(t)

	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)
	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	team := seedTeam(t, pool, admin.OrganizationID, "Platform")
	service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")
	alert := seedTriggeredAlert(t, pool, admin.OrganizationID, service.ID, "memory-high")

	seedSlackSettings(t, queries, admin.OrganizationID)

	promoteRec := postGraphQL(t, handler, `mutation {
		promoteAlertToIncident(input: {
			alertId: "`+alert.ID.String()+`"
			title: "Checkout degradation"
		}) {
			id
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
	require.True(t, incident.SlackChannelID.Valid)
	require.Equal(t, "CINCIDENT1", incident.SlackChannelID.String)
	require.True(t, incident.SlackThreadTs.Valid)
	require.Equal(t, mock.anchorTS, incident.SlackThreadTs.String)
}

func TestGraphQLPromoteAlertToIncidentKeepsIncidentWhenSlackChannelFails(t *testing.T) {
	slackchannel.SetHTTPClient(slackRoundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"ok":false,"error":"name_taken"}`)),
			Header:     make(http.Header),
		}, nil
	}))
	t.Cleanup(func() { slackchannel.SetHTTPClient(nil) })

	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)
	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	team := seedTeam(t, pool, admin.OrganizationID, "Platform")
	service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")
	alert := seedTriggeredAlert(t, pool, admin.OrganizationID, service.ID, "disk-full")

	secrets := testSecretsBox(t)
	const botToken = "xoxb-1234567890abcdefghijklmnop"
	encrypted, err := secrets.Encrypt([]byte(botToken))
	require.NoError(t, err)
	_, err = queries.UpsertOrganizationSlackSettings(context.Background(), db.UpsertOrganizationSlackSettingsParams{
		OrganizationID:     admin.OrganizationID,
		BotTokenCiphertext: encrypted.Ciphertext,
		EncryptionKeyID:    encrypted.KeyID,
		TokenHint:          crypto.SecretHint(botToken),
	})
	require.NoError(t, err)

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
	require.False(t, incident.SlackChannelID.Valid)
}
