package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

type graphqlWSMessage struct {
	Type    string          `json:"type"`
	ID      string          `json:"id,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type graphqlWSNextPayload struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message    string                 `json:"message"`
		Extensions map[string]interface{} `json:"extensions"`
	} `json:"errors"`
}

type graphqlWSSubscribePayload struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphqlWSClient struct {
	conn *websocket.Conn
}

func dialGraphQLWS(t *testing.T, handler http.Handler, cookie *http.Cookie) *graphqlWSClient {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/graphql"
	headers := http.Header{}
	if cookie != nil {
		headers.Set("Cookie", cookie.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader:   headers,
		Subprotocols: []string{"graphql-transport-ws"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test done") })

	client := &graphqlWSClient{conn: conn}
	client.init(t)
	return client
}

func (c *graphqlWSClient) init(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, writeGraphQLWSMessage(ctx, c.conn, graphqlWSMessage{Type: "connection_init"}))

	msg, err := readGraphQLWSMessage(ctx, c.conn)
	require.NoError(t, err)
	require.Equal(t, "connection_ack", msg.Type)
}

func (c *graphqlWSClient) subscribe(t *testing.T, id, query string, variables map[string]any) {
	t.Helper()

	payload, err := json.Marshal(graphqlWSSubscribePayload{
		Query:     query,
		Variables: variables,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, writeGraphQLWSMessage(ctx, c.conn, graphqlWSMessage{
		Type:    "subscribe",
		ID:      id,
		Payload: payload,
	}))
}

func (c *graphqlWSClient) waitForNext(t *testing.T, id string, timeout time.Duration) graphqlWSNextPayload {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		remaining := time.Until(deadline)
		ctx, cancel := context.WithTimeout(context.Background(), remaining)

		msg, err := readGraphQLWSMessage(ctx, c.conn)
		cancel()
		if err != nil {
			if time.Now().Before(deadline) {
				continue
			}
			require.NoError(t, err)
		}

		switch msg.Type {
		case "ping":
			pingCtx, pingCancel := context.WithTimeout(context.Background(), time.Second)
			require.NoError(t, writeGraphQLWSMessage(pingCtx, c.conn, graphqlWSMessage{Type: "pong"}))
			pingCancel()
		case "next":
			require.Equal(t, id, msg.ID)
			var payload graphqlWSNextPayload
			require.NoError(t, json.Unmarshal(msg.Payload, &payload))
			return payload
		case "error":
			require.Equal(t, id, msg.ID)
			t.Fatalf("subscription error: %s", string(msg.Payload))
		case "complete":
			t.Fatalf("subscription completed before next payload")
		}
	}

	t.Fatalf("timed out waiting for subscription payload")
	return graphqlWSNextPayload{}
}

func writeGraphQLWSMessage(ctx context.Context, conn *websocket.Conn, msg graphqlWSMessage) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return conn.Write(ctx, websocket.MessageText, payload)
}

func readGraphQLWSMessage(ctx context.Context, conn *websocket.Conn) (graphqlWSMessage, error) {
	_, payload, err := conn.Read(ctx)
	if err != nil {
		return graphqlWSMessage{}, err
	}

	var msg graphqlWSMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		return graphqlWSMessage{}, err
	}
	return msg, nil
}

func TestGraphQLAlertUpdatedSubscriptionReceivesMutationWithinTwoSeconds(t *testing.T) {
	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)

	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	team := seedTeam(t, pool, admin.OrganizationID, "Platform")
	service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")
	alert := seedTriggeredAlert(t, pool, admin.OrganizationID, service.ID, "cpu-high")

	client := dialGraphQLWS(t, handler, adminCookie)
	client.subscribe(t, "1", fmt.Sprintf(`subscription {
		alertUpdated(orgId: "%s") {
			id
			status
		}
	}`, admin.OrganizationID), nil)

	ackRec := postGraphQL(t, handler, `mutation {
		acknowledgeAlert(id: "`+alert.ID.String()+`") {
			id
			status
		}
	}`, adminCookie)
	require.Equal(t, http.StatusOK, ackRec.Code)

	payload := client.waitForNext(t, "1", 2*time.Second)

	var data struct {
		AlertUpdated struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"alertUpdated"`
	}
	require.NoError(t, json.Unmarshal(payload.Data, &data))
	require.Empty(t, payload.Errors)
	require.Equal(t, alert.ID.String(), data.AlertUpdated.ID)
	require.Equal(t, "ACKNOWLEDGED", data.AlertUpdated.Status)
}

func TestGraphQLAlertUpdatedSubscriptionRejectsUnauthenticated(t *testing.T) {
	handler, _, cleanup := newTestHandler(t)
	defer cleanup()

	client := dialGraphQLWS(t, handler, nil)
	client.subscribe(t, "1", `subscription { alertUpdated(orgId: "00000000-0000-7000-8000-000000000001") { id } }`, nil)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		remaining := time.Until(deadline)
		ctx, cancel := context.WithTimeout(context.Background(), remaining)

		msg, err := readGraphQLWSMessage(ctx, client.conn)
		cancel()
		if err != nil {
			continue
		}

		switch msg.Type {
		case "ping":
			pingCtx, pingCancel := context.WithTimeout(context.Background(), time.Second)
			_ = writeGraphQLWSMessage(pingCtx, client.conn, graphqlWSMessage{Type: "pong"})
			pingCancel()
		case "error":
			var errs []struct {
				Extensions map[string]interface{} `json:"extensions"`
			}
			require.NoError(t, json.Unmarshal(msg.Payload, &errs))
			require.NotEmpty(t, errs)
			require.Equal(t, handlers.CodeUnauthenticated, errs[0].Extensions["code"])
			return
		case "next":
			var payload graphqlWSNextPayload
			require.NoError(t, json.Unmarshal(msg.Payload, &payload))
			if len(payload.Errors) > 0 {
				require.Equal(t, handlers.CodeUnauthenticated, payload.Errors[0].Extensions["code"])
				return
			}
			t.Fatalf("expected unauthenticated subscription error, got data: %s", string(payload.Data))
		}
	}

	t.Fatal("timed out waiting for unauthenticated subscription rejection")
}

func TestGraphQLIncidentTimelineUpdatedSubscriptionReceivesNoteWithinTwoSeconds(t *testing.T) {
	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)

	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	team := seedTeam(t, pool, admin.OrganizationID, "Platform")
	service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")
	alert := seedTriggeredAlert(t, pool, admin.OrganizationID, service.ID, "cpu-high")

	promoteRec := postGraphQL(t, handler, `mutation {
		promoteAlertToIncident(input: {
			alertId: "`+alert.ID.String()+`"
			title: "Checkout degradation"
		}) {
			incidentId
		}
	}`, adminCookie)
	require.Equal(t, http.StatusOK, promoteRec.Code)

	var promoteResp struct {
		Data struct {
			PromoteAlertToIncident struct {
				IncidentID *string `json:"incidentId"`
			} `json:"promoteAlertToIncident"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(promoteRec.Body.Bytes(), &promoteResp))
	require.Empty(t, promoteResp.Errors)
	require.NotNil(t, promoteResp.Data.PromoteAlertToIncident.IncidentID)

	incidentID := *promoteResp.Data.PromoteAlertToIncident.IncidentID

	client := dialGraphQLWS(t, handler, adminCookie)
	client.subscribe(t, "1", fmt.Sprintf(`subscription {
		incidentTimelineUpdated(incidentId: "%s") {
			id
			eventType
			body
		}
	}`, incidentID), nil)

	// Allow the subscription resolver to register on the hub before publishing.
	time.Sleep(100 * time.Millisecond)

	noteRec := postGraphQL(t, handler, `mutation {
		addIncidentTimelineNote(input: {
			incidentId: "`+incidentID+`"
			body: "Customer impact confirmed"
		}) {
			id
			eventType
			body
		}
	}`, adminCookie)
	require.Equal(t, http.StatusOK, noteRec.Code)

	payload := client.waitForNext(t, "1", 2*time.Second)

	var data struct {
		IncidentTimelineUpdated struct {
			ID        string `json:"id"`
			EventType string `json:"eventType"`
			Body      string `json:"body"`
		} `json:"incidentTimelineUpdated"`
	}
	require.NoError(t, json.Unmarshal(payload.Data, &data))
	require.Empty(t, payload.Errors)
	require.Equal(t, "NOTE", data.IncidentTimelineUpdated.EventType)
	require.Equal(t, "Customer impact confirmed", data.IncidentTimelineUpdated.Body)
}
