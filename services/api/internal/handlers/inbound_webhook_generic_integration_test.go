package handlers_test

import (
	"context"
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
	_ "github.com/mdg-labs/escalite/services/integrations/install"
)

func TestInboundWebhookGenericWebhookCreatesAlert(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := inboundWebhookTestHandler(t, 0)
		defer cleanup()

		bootstrapAdmin(t, handler)

		admin, err := db.New(pool).GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")

		config := `{"title":"title","body":"message","dedup_key":"id","priority":"severity"}`
		_, token := seedIntegrationKeyWithConfig(t, pool, admin.OrganizationID, service.ID, "generic-webhook", config)

		payload := []byte(`{
		"title": "Disk usage high",
		"message": "Volume /data is 95% full",
		"id": "host-1-disk",
		"severity": "low"
	}`)
		rec := postInboundWebhook(t, handler, "generic-webhook", token, payload)
		require.Equal(a, 202, rec.Code, rec.Body.String())

		queries := db.New(pool)
		alert, err := queries.GetOpenAlertByServiceDedupKeyForResolve(context.Background(), db.GetOpenAlertByServiceDedupKeyForResolveParams{
			ServiceID: service.ID,
			DedupKey:  "host-1-disk",
		})
		require.NoError(a, err)
		require.Equal(a, "triggered", alert.Status)
		require.Equal(a, "Disk usage high", alert.Summary)
		require.Equal(a, "low", alert.Priority)
	})
}

func TestInboundWebhookGenericWebhookMissingMappedFieldReturnsValidation(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := inboundWebhookTestHandler(t, 0)
		defer cleanup()

		bootstrapAdmin(t, handler)

		admin, err := db.New(pool).GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")

		config := `{"title":"title","dedup_key":"id"}`
		_, token := seedIntegrationKeyWithConfig(t, pool, admin.OrganizationID, service.ID, "generic-webhook", config)

		rec := postInboundWebhook(t, handler, "generic-webhook", token, []byte(`{"title":"Missing dedup key"}`))
		require.Equal(a, 400, rec.Code)

		var errResp struct {
			Code string `json:"code"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &errResp))
		require.Equal(a, handlers.CodeValidation, errResp.Code)
	})
}
