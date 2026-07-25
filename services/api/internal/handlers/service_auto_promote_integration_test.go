package handlers_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/db"
)

func TestGraphQLUpdateServiceAutoPromoteRule(t *testing.T) {
	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)

	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	team := seedTeam(t, pool, admin.OrganizationID, "Platform")
	service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")

	updateRec := postGraphQL(t, handler, `mutation {
		updateService(input: {
			id: "`+service.ID.String()+`"
			name: "checkout-api"
			autoPromoteEnabled: true
			autoPromoteAlertThreshold: 3
			autoPromoteWindowSeconds: 300
			autoPromoteSuppressEscalationPriorities: [HIGH, LOW]
		}) {
			id
			autoPromoteRule {
				enabled
				alertThreshold
				windowSeconds
				suppressEscalationPriorities
			}
		}
	}`, adminCookie)
	require.Equal(t, 200, updateRec.Code, updateRec.Body.String())

	var updateResp struct {
		Data struct {
			UpdateService struct {
				ID              string `json:"id"`
				AutoPromoteRule struct {
					Enabled                        bool     `json:"enabled"`
					AlertThreshold                 int      `json:"alertThreshold"`
					WindowSeconds                  int      `json:"windowSeconds"`
					SuppressEscalationPriorities []string `json:"suppressEscalationPriorities"`
				} `json:"autoPromoteRule"`
			} `json:"updateService"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(updateRec.Body.Bytes(), &updateResp))
	require.Empty(t, updateResp.Errors)
	require.True(t, updateResp.Data.UpdateService.AutoPromoteRule.Enabled)
	require.Equal(t, 3, updateResp.Data.UpdateService.AutoPromoteRule.AlertThreshold)
	require.Equal(t, 300, updateResp.Data.UpdateService.AutoPromoteRule.WindowSeconds)
	require.ElementsMatch(t, []string{"HIGH", "LOW"}, updateResp.Data.UpdateService.AutoPromoteRule.SuppressEscalationPriorities)

	stored, err := queries.GetServiceByID(context.Background(), db.GetServiceByIDParams{
		ID:             service.ID,
		OrganizationID: admin.OrganizationID,
	})
	require.NoError(t, err)
	require.True(t, stored.AutoPromoteEnabled)
	require.Equal(t, int32(3), stored.AutoPromoteAlertThreshold)
	require.Equal(t, int32(300), stored.AutoPromoteWindowSeconds)
	require.ElementsMatch(t, []string{"high", "low"}, stored.AutoPromoteSuppressEscalationPriorities)
}
