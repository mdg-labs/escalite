package handlers_test

import (
	"context"
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/db"
)

func TestGraphQLUpdateServiceAutoPromoteRule(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

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
		require.Equal(a, 200, updateRec.Code, updateRec.Body.String())

		var updateResp struct {
			Data struct {
				UpdateService struct {
					ID              string `json:"id"`
					AutoPromoteRule struct {
						Enabled                      bool     `json:"enabled"`
						AlertThreshold               int      `json:"alertThreshold"`
						WindowSeconds                int      `json:"windowSeconds"`
						SuppressEscalationPriorities []string `json:"suppressEscalationPriorities"`
					} `json:"autoPromoteRule"`
				} `json:"updateService"`
			} `json:"data"`
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(updateRec.Body.Bytes(), &updateResp))
		require.Empty(a, updateResp.Errors)
		require.True(a, updateResp.Data.UpdateService.AutoPromoteRule.Enabled)
		require.Equal(a, 3, updateResp.Data.UpdateService.AutoPromoteRule.AlertThreshold)
		require.Equal(a, 300, updateResp.Data.UpdateService.AutoPromoteRule.WindowSeconds)
		require.ElementsMatch(a, []string{"HIGH", "LOW"}, updateResp.Data.UpdateService.AutoPromoteRule.SuppressEscalationPriorities)

		stored, err := queries.GetServiceByID(context.Background(), db.GetServiceByIDParams{
			ID:             service.ID,
			OrganizationID: admin.OrganizationID,
		})
		require.NoError(a, err)
		require.True(a, stored.AutoPromoteEnabled)
		require.Equal(a, int32(3), stored.AutoPromoteAlertThreshold)
		require.Equal(a, int32(300), stored.AutoPromoteWindowSeconds)
		require.ElementsMatch(a, []string{"high", "low"}, stored.AutoPromoteSuppressEscalationPriorities)
	})
}
