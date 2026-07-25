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

func TestInboundWebhookGrafanaCreatesAndResolvesAlert(t *testing.T) {
	handler, pool, cleanup := inboundWebhookTestHandler(t, 0)
	defer cleanup()

	bootstrapAdmin(t, handler)

	admin, err := db.New(pool).GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	team := seedTeam(t, pool, admin.OrganizationID, "Platform")
	service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")

	_, token := seedIntegrationKey(t, pool, admin.OrganizationID, service.ID, "grafana")

	firing := loadGrafanaFixture(t, "firing.json")
	rec := postInboundWebhook(t, handler, "grafana", token, firing)
	require.Equal(t, 202, rec.Code, rec.Body.String())

	queries := db.New(pool)
	alert, err := queries.GetOpenAlertByServiceDedupKey(context.Background(), db.GetOpenAlertByServiceDedupKeyParams{
		ServiceID: service.ID,
		DedupKey:  "c6eadffa33fcdf37",
	})
	require.NoError(t, err)
	require.Equal(t, "triggered", alert.Status)
	require.Equal(t, "High memory usage on db-primary-01", alert.Summary)
	require.Equal(t, "high", alert.Priority)

	resolved := loadGrafanaFixture(t, "resolved.json")
	rec = postInboundWebhook(t, handler, "grafana", token, resolved)
	require.Equal(t, 202, rec.Code, rec.Body.String())

	closed, err := queries.GetAlertByID(context.Background(), db.GetAlertByIDParams{
		ID:             alert.ID,
		OrganizationID: admin.OrganizationID,
	})
	require.NoError(t, err)
	require.Equal(t, "closed", closed.Status)
	require.True(t, closed.ClosedAt.Valid)
	require.True(t, closed.ResolvedAt.Valid)
	require.True(t, closed.ResolvedIntegration.Valid)
	require.Equal(t, "grafana", closed.ResolvedIntegration.String)
}

func loadGrafanaFixture(t *testing.T, name string) []byte {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "integrations", "grafana", "testdata", name))
	require.NoError(t, err)
	return raw
}
