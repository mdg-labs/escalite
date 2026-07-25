package handlers_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
	_ "github.com/mdg-labs/escalite/services/integrations/install"
)

func TestInboundWebhookGenericWebhookCreatesAlert(t *testing.T) {
	handler, pool, cleanup := inboundWebhookTestHandler(t, 0)
	defer cleanup()

	bootstrapAdmin(t, handler)

	admin, err := db.New(pool).GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

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
	require.Equal(t, 202, rec.Code, rec.Body.String())

	queries := db.New(pool)
	alert, err := queries.GetOpenAlertByServiceDedupKeyForResolve(context.Background(), db.GetOpenAlertByServiceDedupKeyForResolveParams{
		ServiceID: service.ID,
		DedupKey:  "host-1-disk",
	})
	require.NoError(t, err)
	require.Equal(t, "triggered", alert.Status)
	require.Equal(t, "Disk usage high", alert.Summary)
	require.Equal(t, "low", alert.Priority)
}

func TestInboundWebhookGenericWebhookMissingMappedFieldReturnsValidation(t *testing.T) {
	handler, pool, cleanup := inboundWebhookTestHandler(t, 0)
	defer cleanup()

	bootstrapAdmin(t, handler)

	admin, err := db.New(pool).GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	team := seedTeam(t, pool, admin.OrganizationID, "Platform")
	service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")

	config := `{"title":"title","dedup_key":"id"}`
	_, token := seedIntegrationKeyWithConfig(t, pool, admin.OrganizationID, service.ID, "generic-webhook", config)

	rec := postInboundWebhook(t, handler, "generic-webhook", token, []byte(`{"title":"Missing dedup key"}`))
	require.Equal(t, 400, rec.Code)

	var errResp struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	require.Equal(t, handlers.CodeValidation, errResp.Code)
}
