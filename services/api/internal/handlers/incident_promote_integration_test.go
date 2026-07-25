package handlers_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/db"
)

func TestGraphQLPromoteAlertToIncidentCreatesDeclaredTimelineEvent(t *testing.T) {
	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)

	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

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
	require.Equal(t, 200, promoteRec.Code)

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
	require.NoError(t, json.Unmarshal(promoteRec.Body.Bytes(), &promoteResp))
	require.Empty(t, promoteResp.Errors)
	require.Equal(t, alert.ID.String(), promoteResp.Data.PromoteAlertToIncident.ID)
	require.Equal(t, alert.Summary, promoteResp.Data.PromoteAlertToIncident.Summary)
	require.NotNil(t, promoteResp.Data.PromoteAlertToIncident.IncidentID)

	incidentID := *promoteResp.Data.PromoteAlertToIncident.IncidentID

	storedAlert, err := queries.GetAlertByID(context.Background(), db.GetAlertByIDParams{
		ID:             alert.ID,
		OrganizationID: admin.OrganizationID,
	})
	require.NoError(t, err)
	require.True(t, storedAlert.IncidentID.Valid)
	require.Equal(t, incidentID, uuid.UUID(storedAlert.IncidentID.Bytes).String())

	events, err := queries.ListTimelineEventsByIncidentID(context.Background(), db.ListTimelineEventsByIncidentIDParams{
		IncidentID:     storedAlert.IncidentID.Bytes,
		OrganizationID: admin.OrganizationID,
	})
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "declared", events[0].EventType)
	require.Equal(t, "Checkout degradation", events[0].Body)

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
	require.Equal(t, 200, attachRec.Code)

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
	require.NoError(t, json.Unmarshal(attachRec.Body.Bytes(), &attachResp))
	require.Empty(t, attachResp.Errors)
	require.NotNil(t, attachResp.Data.PromoteAlertToIncident.IncidentID)
	require.Equal(t, incidentID, *attachResp.Data.PromoteAlertToIncident.IncidentID)

	incidentAlerts, err := queries.ListAlertsByIncidentID(context.Background(), db.ListAlertsByIncidentIDParams{
		IncidentID:     storedAlert.IncidentID,
		OrganizationID: admin.OrganizationID,
	})
	require.NoError(t, err)
	require.Len(t, incidentAlerts, 2)
}
