package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
	_ "github.com/mdg-labs/escalite/services/integrations/install"
)

func TestInboundWebhookDatadogCreatesAndResolvesAlert(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := inboundWebhookTestHandler(t, 0)
		defer cleanup()

		bootstrapAdmin(t, handler)

		admin, err := db.New(pool).GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")

		_, token := seedIntegrationKey(t, pool, admin.OrganizationID, service.ID, "datadog")

		firing := loadDatadogFixture(t, "triggered.json")
		rec := postInboundWebhook(t, handler, "datadog", token, firing)
		require.Equal(a, 202, rec.Code, rec.Body.String())

		queries := db.New(pool)
		alert, err := queries.GetOpenAlertByServiceDedupKeyForResolve(context.Background(), db.GetOpenAlertByServiceDedupKeyForResolveParams{
			ServiceID: service.ID,
			DedupKey:  "monitor:12345|tags:env:prod,service:api",
		})
		require.NoError(a, err)
		require.Equal(a, "triggered", alert.Status)
		require.Equal(a, "CPU usage above 90%", alert.Summary)
		require.Equal(a, "low", alert.Priority)

		recovered := loadDatadogFixture(t, "recovered.json")
		rec = postInboundWebhook(t, handler, "datadog", token, recovered)
		require.Equal(a, 202, rec.Code, rec.Body.String())

		closed, err := queries.GetAlertByID(context.Background(), db.GetAlertByIDParams{
			ID:             alert.ID,
			OrganizationID: admin.OrganizationID,
		})
		require.NoError(a, err)
		require.Equal(a, "closed", closed.Status)
		require.True(a, closed.ClosedAt.Valid)
	})
}

func TestInboundWebhookDatadogInvalidSignatureReturnsUnauthorized(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := inboundWebhookTestHandler(t, 0)
		defer cleanup()

		bootstrapAdmin(t, handler)

		admin, err := db.New(pool).GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")

		config := `{"signature_secret":"top-secret"}`
		_, token := seedIntegrationKeyWithConfig(t, pool, admin.OrganizationID, service.ID, "datadog", config)

		payload := loadDatadogFixture(t, "triggered.json")
		req := httptest.NewRequest(http.MethodPost, "/webhook/datadog/"+token, bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Escalite-Signature", "invalid")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		require.Equal(a, http.StatusUnauthorized, rec.Code)

		var errResp struct {
			Code string `json:"code"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &errResp))
		require.Equal(a, handlers.CodeUnauthenticated, errResp.Code)
	})
}

func TestInboundWebhookDatadogAcceptsValidSignature(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := inboundWebhookTestHandler(t, 0)
		defer cleanup()

		bootstrapAdmin(t, handler)

		admin, err := db.New(pool).GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")

		secret := "top-secret"
		config := `{"signature_secret":"top-secret"}`
		_, token := seedIntegrationKeyWithConfig(t, pool, admin.OrganizationID, service.ID, "datadog", config)

		payload := loadDatadogFixture(t, "triggered.json")
		req := httptest.NewRequest(http.MethodPost, "/webhook/datadog/"+token, bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Escalite-Signature", secret)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		require.Equal(a, http.StatusAccepted, rec.Code, rec.Body.String())
	})
}

func loadDatadogFixture(t *testing.T, name string) []byte {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "integrations", "datadog", "testdata", name))
	require.NoError(t, err)
	return raw
}
