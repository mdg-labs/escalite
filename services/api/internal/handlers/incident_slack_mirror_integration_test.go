package handlers_test

import (
	"context"
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/allure-framework/allure-go/testify/require"
	"github.com/google/uuid"

	"github.com/mdg-labs/escalite/services/api/internal/crypto"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/engine/channels/slackchannel"
)

type slackPostMessagePayload struct {
	Channel  string `json:"channel"`
	Text     string `json:"text"`
	ThreadTS string `json:"thread_ts"`
}

type slackAPIMock struct {
	mu sync.Mutex

	anchorTS        string
	threadReplies   []slackPostMessagePayload
	channelMessages []slackPostMessagePayload
	threadNotify    chan slackPostMessagePayload
}

func newSlackAPIMock() *slackAPIMock {
	return &slackAPIMock{
		anchorTS:     "1710000000.000100",
		threadNotify: make(chan slackPostMessagePayload, 4),
	}
}

func (m *slackAPIMock) install(t *testing.T) {
	t.Helper()
	slackchannel.SetHTTPClient(slackRoundTripFunc(m.roundTrip))
	t.Cleanup(func() { slackchannel.SetHTTPClient(nil) })
}

func (m *slackAPIMock) roundTrip(req *http.Request) (*http.Response, error) {
	switch req.URL.String() {
	case "https://slack.com/api/conversations.create":
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"ok":true,"channel":{"id":"CINCIDENT1"}}`)),
			Header:     make(http.Header),
		}, nil
	case "https://slack.com/api/chat.postMessage":
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}

		var payload slackPostMessagePayload
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, err
		}

		m.mu.Lock()
		if payload.ThreadTS != "" {
			m.threadReplies = append(m.threadReplies, payload)
			select {
			case m.threadNotify <- payload:
			default:
			}
		} else {
			m.channelMessages = append(m.channelMessages, payload)
		}
		m.mu.Unlock()

		ts := m.anchorTS
		if payload.ThreadTS != "" {
			ts = "1710000000.000200"
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"ok":true,"ts":"` + ts + `"}`)),
			Header:     make(http.Header),
		}, nil
	default:
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(strings.NewReader(`{"ok":false,"error":"unknown_method"}`)),
			Header:     make(http.Header),
		}, nil
	}
}

func seedSlackSettings(t *testing.T, queries *db.Queries, orgID uuid.UUID) {
	t.Helper()

	secrets := testSecretsBox(t)
	const botToken = "xoxb-1234567890abcdefghijklmnop"
	encrypted, err := secrets.Encrypt([]byte(botToken))
	require.NoError(t, err)
	_, err = queries.UpsertOrganizationSlackSettings(context.Background(), db.UpsertOrganizationSlackSettingsParams{
		OrganizationID:     orgID,
		BotTokenCiphertext: encrypted.Ciphertext,
		EncryptionKeyID:    encrypted.KeyID,
		TokenHint:          crypto.SecretHint(botToken),
	})
	require.NoError(t, err)
}

func promoteAlertToIncident(t *testing.T, handler http.Handler, adminCookie *http.Cookie, alertID uuid.UUID) uuid.UUID {
	t.Helper()

	promoteRec := postGraphQL(t, handler, `mutation {
		promoteAlertToIncident(input: {
			alertId: "`+alertID.String()+`"
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

	return uuid.MustParse(*promoteResp.Data.PromoteAlertToIncident.IncidentID)
}

func TestGraphQLIncidentNoteMirrorsToSlackThreadWithinFiveSeconds(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		mock := newSlackAPIMock()
		mock.install(t)

		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)
		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")
		alert := seedTriggeredAlert(t, pool, admin.OrganizationID, service.ID, "memory-high")
		seedSlackSettings(t, queries, admin.OrganizationID)

		incidentID := promoteAlertToIncident(t, handler, adminCookie, alert.ID)

		incident, err := queries.GetIncidentByID(context.Background(), db.GetIncidentByIDParams{
			ID:             incidentID,
			OrganizationID: admin.OrganizationID,
		})
		require.NoError(a, err)
		require.True(a, incident.SlackChannelID.Valid)
		require.True(a, incident.SlackThreadTs.Valid)
		require.Equal(a, mock.anchorTS, incident.SlackThreadTs.String)

		noteBody := "Rollback completed and metrics are recovering"
		noteRec := postGraphQL(t, handler, `mutation {
		addIncidentTimelineNote(input: {
			incidentId: "`+incidentID.String()+`"
			body: "`+noteBody+`"
		}) {
			id
			body
		}
	}`, adminCookie)
		require.Equal(a, 200, noteRec.Code)

		var noteResp struct {
			Data struct {
				AddIncidentTimelineNote struct {
					ID   string `json:"id"`
					Body string `json:"body"`
				} `json:"addIncidentTimelineNote"`
			} `json:"data"`
			Errors []any `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(noteRec.Body.Bytes(), &noteResp))
		require.Empty(a, noteResp.Errors)

		select {
		case payload := <-mock.threadNotify:
			require.Equal(a, "CINCIDENT1", payload.Channel)
			require.Equal(a, mock.anchorTS, payload.ThreadTS)
			require.Contains(a, payload.Text, noteBody)
			require.Contains(a, payload.Text, admin.Email)
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for slack thread mirror")
		}
	})
}

func TestGraphQLResolveIncidentPostsSlackSummaryToChannel(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		mock := newSlackAPIMock()
		mock.install(t)

		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)
		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")
		alert := seedTriggeredAlert(t, pool, admin.OrganizationID, service.ID, "disk-full")
		seedSlackSettings(t, queries, admin.OrganizationID)

		incidentID := promoteAlertToIncident(t, handler, adminCookie, alert.ID)

		noteBody := "Root cause identified in cache layer"
		noteRec := postGraphQL(t, handler, `mutation {
		addIncidentTimelineNote(input: {
			incidentId: "`+incidentID.String()+`"
			body: "`+noteBody+`"
		}) {
			id
		}
	}`, adminCookie)
		require.Equal(a, 200, noteRec.Code)

		select {
		case <-mock.threadNotify:
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for note mirror before resolve test")
		}

		resolveRec := postGraphQL(t, handler, `mutation {
		updateIncidentStatus(input: {
			id: "`+incidentID.String()+`"
			status: RESOLVED
			body: "Incident fully mitigated"
		}) {
			id
			status
		}
	}`, adminCookie)
		require.Equal(a, 200, resolveRec.Code)

		deadline := time.Now().Add(5 * time.Second)
		var summary slackPostMessagePayload
		for time.Now().Before(deadline) {
			mock.mu.Lock()
			for _, message := range mock.channelMessages {
				if strings.Contains(message.Text, "Incident resolved") {
					summary = message
					mock.mu.Unlock()
					goto foundSummary
				}
			}
			mock.mu.Unlock()
			time.Sleep(50 * time.Millisecond)
		}
		t.Fatal("timed out waiting for slack resolve summary")

	foundSummary:
		require.Equal(a, "CINCIDENT1", summary.Channel)
		require.Empty(a, summary.ThreadTS)
		require.Contains(a, summary.Text, "Incident resolved")
		require.Contains(a, summary.Text, noteBody)
	})
}
