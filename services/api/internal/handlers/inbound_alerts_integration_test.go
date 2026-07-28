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
	"github.com/google/uuid"

	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
	_ "github.com/mdg-labs/escalite/services/integrations/install"
)

func postInboundAlert(t *testing.T, handler http.Handler, token string, body []byte) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/alerts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestInboundAlertsMissingAuthorizationReturnsUnauthenticated(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := inboundWebhookTestHandler(t, 0)
		defer cleanup()

		rec := postInboundAlert(t, handler, "", []byte(`{"summary":"Test","dedup_key":"abc"}`))
		require.Equal(a, http.StatusUnauthorized, rec.Code)

		var errResp struct {
			Error string `json:"error"`
			Code  string `json:"code"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &errResp))
		require.Equal(a, handlers.CodeUnauthenticated, errResp.Code)
		require.NotEmpty(a, errResp.Error)
	})
}

func TestInboundAlertsValidPayloadCreatesAlert(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := inboundWebhookTestHandler(t, 0)
		defer cleanup()

		bootstrapAdmin(t, handler)

		admin, err := db.New(pool).GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")

		_, token := seedIntegrationKey(t, pool, admin.OrganizationID, service.ID, "generic-rest-api")

		payload := []byte(`{
		"summary": "Disk usage high",
		"description": "Volume /data is 95% full",
		"dedup_key": "host-1-disk",
		"priority": "low"
	}`)
		rec := postInboundAlert(t, handler, token, payload)
		require.Equal(a, http.StatusCreated, rec.Code, rec.Body.String())

		var resp struct {
			ID string `json:"id"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.NotEmpty(a, resp.ID)
		_, err = uuid.Parse(resp.ID)
		require.NoError(a, err)

		queries := db.New(pool)
		alert, err := queries.GetOpenAlertByServiceDedupKeyForResolve(context.Background(), db.GetOpenAlertByServiceDedupKeyForResolveParams{
			ServiceID: service.ID,
			DedupKey:  "host-1-disk",
		})
		require.NoError(a, err)
		require.Equal(a, resp.ID, alert.ID.String())
		require.Equal(a, "triggered", alert.Status)
		require.Equal(a, "Disk usage high", alert.Summary)
		require.Equal(a, "low", alert.Priority)
	})
}

func TestInboundAlertsDuplicateDedupKeyCollapsesAlert(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := inboundWebhookTestHandler(t, 0)
		defer cleanup()

		bootstrapAdmin(t, handler)

		admin, err := db.New(pool).GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")

		_, token := seedIntegrationKey(t, pool, admin.OrganizationID, service.ID, "generic-rest-api")

		payload := []byte(`{
		"summary": "Disk usage high",
		"description": "Volume /data is 95% full",
		"dedup_key": "host-1-disk",
		"priority": "low"
	}`)
		rec := postInboundAlert(t, handler, token, payload)
		require.Equal(a, http.StatusCreated, rec.Code, rec.Body.String())

		var firstResp struct {
			ID string `json:"id"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &firstResp))

		rec = postInboundAlert(t, handler, token, payload)
		require.Equal(a, http.StatusCreated, rec.Code, rec.Body.String())

		var secondResp struct {
			ID string `json:"id"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &secondResp))
		require.Equal(a, firstResp.ID, secondResp.ID)

		queries := db.New(pool)
		alert, err := queries.GetOpenAlertByServiceDedupKeyForResolve(context.Background(), db.GetOpenAlertByServiceDedupKeyForResolveParams{
			ServiceID: service.ID,
			DedupKey:  "host-1-disk",
		})
		require.NoError(a, err)
		require.Equal(a, firstResp.ID, alert.ID.String())
		require.Equal(a, int32(2), alert.EventCount)
	})
}

func TestInboundAlertsRejectsNonGenericRESTKey(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := inboundWebhookTestHandler(t, 0)
		defer cleanup()

		bootstrapAdmin(t, handler)

		admin, err := db.New(pool).GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")

		_, token := seedIntegrationKey(t, pool, admin.OrganizationID, service.ID, "test-plugin")

		rec := postInboundAlert(t, handler, token, []byte(`{"summary":"Test","dedup_key":"abc"}`))
		require.Equal(a, http.StatusUnauthorized, rec.Code)

		var errResp struct {
			Code string `json:"code"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &errResp))
		require.Equal(a, handlers.CodeUnauthenticated, errResp.Code)
	})
}

func TestInboundAlertsResolveUnknownDedupKeyReturnsOK(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := inboundWebhookTestHandler(t, 0)
		defer cleanup()

		bootstrapAdmin(t, handler)

		admin, err := db.New(pool).GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")

		_, token := seedIntegrationKey(t, pool, admin.OrganizationID, service.ID, "generic-rest-api")

		payload := []byte(`{
		"summary": "Recovered",
		"dedup_key": "unknown-key",
		"event_type": "resolved"
	}`)
		rec := postInboundAlert(t, handler, token, payload)
		require.Equal(a, http.StatusOK, rec.Code, rec.Body.String())

		var resp struct {
			Status string `json:"status"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Equal(a, "resolved", resp.Status)
	})
}
