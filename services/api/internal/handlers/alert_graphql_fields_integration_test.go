package handlers_test

import (
	"context"
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/db"
)

// TestGraphQLAlertsQueryResolvesNestedFields proves the alerts(limit) query
// resolves the custom Alert.service / Alert.escalationState /
// Alert.notificationAttempts fields (gqlgen resolver bindings) instead of
// returning a GraphQL null error for the non-nullable `service` field.
// Regression test for EL-268.
func TestGraphQLAlertsQueryResolvesNestedFields(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")
		alert := seedTriggeredAlert(t, pool, admin.OrganizationID, service.ID, "cpu-high")

		rec := postGraphQL(t, handler, `{
		alerts(limit: 10) {
			id
			service { id name }
			status
			escalationState {
				currentStep
				nextEscalationAt
				escalatedExhausted
			}
			notificationAttempts {
				id
				channel
				status
				sentAt
				createdAt
			}
		}
	}`, adminCookie)
		require.Equal(a, 200, rec.Code)

		var resp struct {
			Data struct {
				Alerts []struct {
					ID      string `json:"id"`
					Service struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"service"`
					Status          string `json:"status"`
					EscalationState *struct {
						CurrentStep        int     `json:"currentStep"`
						NextEscalationAt   *string `json:"nextEscalationAt"`
						EscalatedExhausted bool    `json:"escalatedExhausted"`
					} `json:"escalationState"`
					NotificationAttempts []struct {
						ID        string  `json:"id"`
						Channel   string  `json:"channel"`
						Status    string  `json:"status"`
						SentAt    *string `json:"sentAt"`
						CreatedAt string  `json:"createdAt"`
					} `json:"notificationAttempts"`
				} `json:"alerts"`
			} `json:"data"`
			Errors []struct {
				Message    string                 `json:"message"`
				Extensions map[string]interface{} `json:"extensions"`
			} `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Empty(a, resp.Errors)
		require.NotEmpty(a, resp.Data.Alerts)

		var found bool
		for _, got := range resp.Data.Alerts {
			if got.ID != alert.ID.String() {
				continue
			}
			found = true
			require.Equal(a, service.ID.String(), got.Service.ID)
			require.Equal(a, service.Name, got.Service.Name)
			require.Equal(a, "TRIGGERED", got.Status)
			require.NotNil(a, got.EscalationState)
			require.Equal(a, 1, got.EscalationState.CurrentStep)
			require.NotNil(a, got.NotificationAttempts)
		}
		require.True(a, found, "seeded alert not found in alerts(limit) response")
	})
}
