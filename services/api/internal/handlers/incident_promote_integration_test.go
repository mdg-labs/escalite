package handlers_test

import (
	"context"
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"
	"github.com/google/uuid"

	"github.com/mdg-labs/escalite/services/api/internal/db"
)

func TestGraphQLPromoteAlertToIncidentCreatesDeclaredTimelineEvent(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")
		alert := seedTriggeredAlert(t, pool, admin.OrganizationID, service.ID, "memory-high")

		promoteRec := postGraphQL(t, handler, `mutation {
		promoteAlertToIncident(input: {
			alertId: "`+alert.ID.String()+`"
			title: "Checkout degradation"
		}) {
			id
			incidentId
			summary
		}
	}`, adminCookie)
		require.Equal(a, 200, promoteRec.Code)

		var promoteResp struct {
			Data struct {
				PromoteAlertToIncident struct {
					ID         string  `json:"id"`
					IncidentID *string `json:"incidentId"`
					Summary    string  `json:"summary"`
				} `json:"promoteAlertToIncident"`
			} `json:"data"`
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(promoteRec.Body.Bytes(), &promoteResp))
		require.Empty(a, promoteResp.Errors)
		require.Equal(a, alert.ID.String(), promoteResp.Data.PromoteAlertToIncident.ID)
		require.Equal(a, alert.Summary, promoteResp.Data.PromoteAlertToIncident.Summary)
		require.NotNil(a, promoteResp.Data.PromoteAlertToIncident.IncidentID)

		incidentID := *promoteResp.Data.PromoteAlertToIncident.IncidentID

		storedAlert, err := queries.GetAlertByID(context.Background(), db.GetAlertByIDParams{
			ID:             alert.ID,
			OrganizationID: admin.OrganizationID,
		})
		require.NoError(a, err)
		require.True(a, storedAlert.IncidentID.Valid)
		require.Equal(a, incidentID, uuid.UUID(storedAlert.IncidentID.Bytes).String())

		events, err := queries.ListTimelineEventsByIncidentID(context.Background(), db.ListTimelineEventsByIncidentIDParams{
			IncidentID:     storedAlert.IncidentID.Bytes,
			OrganizationID: admin.OrganizationID,
		})
		require.NoError(a, err)
		require.Len(a, events, 1)
		require.Equal(a, "declared", events[0].EventType)
		require.Equal(a, "Checkout degradation", events[0].Body)

		secondAlert := seedTriggeredAlert(t, pool, admin.OrganizationID, service.ID, "disk-full")

		attachRec := postGraphQL(t, handler, `mutation {
		promoteAlertToIncident(input: {
			alertId: "`+secondAlert.ID.String()+`"
			incidentId: "`+incidentID+`"
		}) {
			id
			incidentId
		}
	}`, adminCookie)
		require.Equal(a, 200, attachRec.Code)

		var attachResp struct {
			Data struct {
				PromoteAlertToIncident struct {
					ID         string  `json:"id"`
					IncidentID *string `json:"incidentId"`
				} `json:"promoteAlertToIncident"`
			} `json:"data"`
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(attachRec.Body.Bytes(), &attachResp))
		require.Empty(a, attachResp.Errors)
		require.NotNil(a, attachResp.Data.PromoteAlertToIncident.IncidentID)
		require.Equal(a, incidentID, *attachResp.Data.PromoteAlertToIncident.IncidentID)

		incidentAlerts, err := queries.ListAlertsByIncidentID(context.Background(), db.ListAlertsByIncidentIDParams{
			IncidentID:     storedAlert.IncidentID,
			OrganizationID: admin.OrganizationID,
		})
		require.NoError(a, err)
		require.Len(a, incidentAlerts, 2)
	})
}
