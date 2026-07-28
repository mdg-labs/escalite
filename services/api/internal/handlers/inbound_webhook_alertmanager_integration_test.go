package handlers_test

import (
	"context"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"os"
	"path/filepath"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/db"
	_ "github.com/mdg-labs/escalite/services/integrations/install"
	_ "github.com/mdg-labs/escalite/services/integrations/testplugin"
)

func TestInboundWebhookAlertmanagerCreatesAndResolvesAlert(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := inboundWebhookTestHandler(t, 0)
		defer cleanup()

		bootstrapAdmin(t, handler)

		admin, err := db.New(pool).GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")

		_, token := seedIntegrationKey(t, pool, admin.OrganizationID, service.ID, "prometheus-alertmanager")

		firing := loadAlertmanagerFixture(t, "firing_v4.json")
		rec := postInboundWebhook(t, handler, "prometheus-alertmanager", token, firing)
		require.Equal(a, 202, rec.Code, rec.Body.String())

		queries := db.New(pool)
		alert, err := queries.GetOpenAlertByServiceDedupKeyForResolve(context.Background(), db.GetOpenAlertByServiceDedupKeyForResolveParams{
			ServiceID: service.ID,
			DedupKey:  "1a30ba71cca2921f",
		})
		require.NoError(a, err)
		require.Equal(a, "triggered", alert.Status)
		require.Equal(a, "BlackBox Probe Failure: https://server.example.org", alert.Summary)
		require.Equal(a, "high", alert.Priority)

		rec = postInboundWebhook(t, handler, "prometheus-alertmanager", token, firing)
		require.Equal(a, 202, rec.Code, rec.Body.String())

		collapsed, err := queries.GetOpenAlertByServiceDedupKeyForResolve(context.Background(), db.GetOpenAlertByServiceDedupKeyForResolveParams{
			ServiceID: service.ID,
			DedupKey:  "1a30ba71cca2921f",
		})
		require.NoError(a, err)
		require.Equal(a, alert.ID, collapsed.ID)
		require.Equal(a, int32(2), collapsed.EventCount)

		resolved := loadAlertmanagerFixture(t, "resolved_v4.json")
		rec = postInboundWebhook(t, handler, "prometheus-alertmanager", token, resolved)
		require.Equal(a, 202, rec.Code, rec.Body.String())

		closed, err := queries.GetAlertByID(context.Background(), db.GetAlertByIDParams{
			ID:             alert.ID,
			OrganizationID: admin.OrganizationID,
		})
		require.NoError(a, err)
		require.Equal(a, "closed", closed.Status)
		require.True(a, closed.ClosedAt.Valid)
		require.True(a, closed.ResolvedAt.Valid)
		require.True(a, closed.ResolvedIntegration.Valid)
		require.Equal(a, "prometheus-alertmanager", closed.ResolvedIntegration.String)
	})
}

func loadAlertmanagerFixture(t *testing.T, name string) []byte {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "integrations", "alertmanager", "testdata", name))
	require.NoError(t, err)
	return raw
}
