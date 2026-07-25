package handlers_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/db"
	_ "github.com/mdg-labs/escalite/services/integrations/install"
	_ "github.com/mdg-labs/escalite/services/integrations/testplugin"
)

func TestInboundWebhookAlertmanagerCreatesAndResolvesAlert(t *testing.T) {
	handler, pool, cleanup := inboundWebhookTestHandler(t, 0)
	defer cleanup()

	bootstrapAdmin(t, handler)

	admin, err := db.New(pool).GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	team := seedTeam(t, pool, admin.OrganizationID, "Platform")
	service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")

	_, token := seedIntegrationKey(t, pool, admin.OrganizationID, service.ID, "prometheus-alertmanager")

	firing := loadAlertmanagerFixture(t, "firing_v4.json")
	rec := postInboundWebhook(t, handler, "prometheus-alertmanager", token, firing)
	require.Equal(t, 202, rec.Code, rec.Body.String())

	queries := db.New(pool)
	alert, err := queries.GetOpenAlertByServiceDedupKey(context.Background(), db.GetOpenAlertByServiceDedupKeyParams{
		ServiceID: service.ID,
		DedupKey:  "1a30ba71cca2921f",
	})
	require.NoError(t, err)
	require.Equal(t, "triggered", alert.Status)
	require.Equal(t, "BlackBox Probe Failure: https://server.example.org", alert.Summary)
	require.Equal(t, "high", alert.Priority)

	resolved := loadAlertmanagerFixture(t, "resolved_v4.json")
	rec = postInboundWebhook(t, handler, "prometheus-alertmanager", token, resolved)
	require.Equal(t, 202, rec.Code, rec.Body.String())

	closed, err := queries.GetAlertByID(context.Background(), db.GetAlertByIDParams{
		ID:             alert.ID,
		OrganizationID: admin.OrganizationID,
	})
	require.NoError(t, err)
	require.Equal(t, "closed", closed.Status)
	require.True(t, closed.ClosedAt.Valid)
}

func loadAlertmanagerFixture(t *testing.T, name string) []byte {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "integrations", "alertmanager", "testdata", name))
	require.NoError(t, err)
	return raw
}
