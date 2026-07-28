package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
	"github.com/mdg-labs/escalite/services/api/internal/server"
	_ "github.com/mdg-labs/escalite/services/integrations/install"
)

const (
	testInboundEmailDomain = "inbound.escalite.test"
	testInboundRelaySecret = "relay-secret-for-tests"
)

func inboundEmailTestHandler(t *testing.T, requireAuthenticated bool) (http.Handler, *pgxpool.Pool, func()) {
	t.Helper()

	return newTestHandlerWithOptions(t, testServerOptions{
		InboundEmail: &server.InboundEmailOptions{
			RelaySecret:          testInboundRelaySecret,
			Domain:               testInboundEmailDomain,
			RequireAuthenticated: requireAuthenticated,
		},
	})
}

func postInboundEmail(t *testing.T, handler http.Handler, body []byte, opts ...func(*http.Request)) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inbound/email", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Escalite-Relay-Secret", testInboundRelaySecret)
	for _, opt := range opts {
		opt(req)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestInboundEmailCreatesAlertWithSourceEmail(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := inboundEmailTestHandler(t, true)
		defer cleanup()

		bootstrapAdmin(t, handler)

		admin, err := db.New(pool).GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")

		config := `{"title":"subject","body":"text","dedup_key":"message_id"}`
		_, token := seedIntegrationKeyWithConfig(t, pool, admin.OrganizationID, service.ID, "email-to-alert", config)
		recipient := handlers.InboundEmailAddress(token, testInboundEmailDomain)

		payload, err := json.Marshal(map[string]any{
			"to":            recipient,
			"from":          "monitor@example.com",
			"subject":       "Disk usage high",
			"text":          "Volume /data is 95% full",
			"message_id":    "<disk-alert-1@example.com>",
			"authenticated": true,
		})
		require.NoError(a, err)

		rec := postInboundEmail(t, handler, payload)
		require.Equal(a, http.StatusAccepted, rec.Code, rec.Body.String())

		queries := db.New(pool)
		alert, err := queries.GetOpenAlertByServiceDedupKeyForResolve(context.Background(), db.GetOpenAlertByServiceDedupKeyForResolveParams{
			ServiceID: service.ID,
			DedupKey:  "<disk-alert-1@example.com>",
		})
		require.NoError(a, err)
		require.Equal(a, "triggered", alert.Status)
		require.Equal(a, "Disk usage high", alert.Summary)
		require.Equal(a, "Volume /data is 95% full", alert.Description.String)
	})
}

func TestInboundEmailRejectsMissingRelaySecret(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := inboundEmailTestHandler(t, true)
		defer cleanup()

		rec := postInboundEmail(t, handler, []byte(`{}`), func(req *http.Request) {
			req.Header.Del("X-Escalite-Relay-Secret")
		})
		require.Equal(a, http.StatusUnauthorized, rec.Code)
	})
}

func TestInboundEmailRejectsUnsignedMailWhenRequired(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := inboundEmailTestHandler(t, true)
		defer cleanup()

		bootstrapAdmin(t, handler)

		admin, err := db.New(pool).GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")

		_, token := seedIntegrationKeyWithConfig(t, pool, admin.OrganizationID, service.ID, "email-to-alert", `{"title":"subject","dedup_key":"message_id"}`)
		recipient := handlers.InboundEmailAddress(token, testInboundEmailDomain)

		payload, err := json.Marshal(map[string]any{
			"to":         recipient,
			"subject":    "Unsigned alert",
			"message_id": "<unsigned@example.com>",
		})
		require.NoError(a, err)

		rec := postInboundEmail(t, handler, payload)
		require.Equal(a, http.StatusForbidden, rec.Code)
	})
}

func TestInboundEmailAcceptsAuthenticatedRelayHeader(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := inboundEmailTestHandler(t, true)
		defer cleanup()

		bootstrapAdmin(t, handler)

		admin, err := db.New(pool).GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")

		_, token := seedIntegrationKeyWithConfig(t, pool, admin.OrganizationID, service.ID, "email-to-alert", `{"title":"subject","dedup_key":"message_id"}`)
		recipient := handlers.InboundEmailAddress(token, testInboundEmailDomain)

		payload, err := json.Marshal(map[string]any{
			"to":         recipient,
			"subject":    "Relay signed alert",
			"message_id": "<signed@example.com>",
		})
		require.NoError(a, err)

		rec := postInboundEmail(t, handler, payload, func(req *http.Request) {
			req.Header.Set("X-Escalite-Email-Authenticated", "pass")
		})
		require.Equal(a, http.StatusAccepted, rec.Code, rec.Body.String())
	})
}

func TestInboundEmailInvalidRecipientReturnsValidation(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := inboundEmailTestHandler(t, false)
		defer cleanup()

		payload := []byte(`{"to":"not-an-email","subject":"Hello","message_id":"<x@example.com>","authenticated":true}`)
		rec := postInboundEmail(t, handler, payload)
		require.Equal(a, http.StatusBadRequest, rec.Code)
	})
}
