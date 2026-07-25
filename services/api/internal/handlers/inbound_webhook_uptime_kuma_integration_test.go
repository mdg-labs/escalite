package handlers_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/db"
	_ "github.com/mdg-labs/escalite/services/integrations/install"
)

func TestInboundWebhookUptimeKumaCreatesAndResolvesAlert(t *testing.T) {
	handler, pool, cleanup := inboundWebhookTestHandler(t, 0)
	defer cleanup()

	bootstrapAdmin(t, handler)

	admin, err := db.New(pool).GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	team := seedTeam(t, pool, admin.OrganizationID, "Platform")
	service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")

	_, token := seedIntegrationKey(t, pool, admin.OrganizationID, service.ID, "uptime-kuma")

	down := loadUptimeKumaFixture(t, "down.json")
	rec := postInboundWebhook(t, handler, "uptime-kuma", token, down)
	require.Equal(t, 202, rec.Code, rec.Body.String())

	queries := db.New(pool)
	alert, err := queries.GetOpenAlertByServiceDedupKeyForResolve(context.Background(), db.GetOpenAlertByServiceDedupKeyForResolveParams{
		ServiceID: service.ID,
		DedupKey:  "42",
	})
	require.NoError(t, err)
	require.Equal(t, "triggered", alert.Status)
	require.Equal(t, "checkout-api", alert.Summary)
	require.Equal(t, "high", alert.Priority)

	up := loadUptimeKumaFixture(t, "up.json")
	rec = postInboundWebhook(t, handler, "uptime-kuma", token, up)
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
	require.Equal(t, "uptime-kuma", closed.ResolvedIntegration.String)
}

func loadUptimeKumaFixture(t *testing.T, name string) []byte {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "integrations", "uptimekuma", "testdata", name))
	require.NoError(t, err)
	return raw
}
