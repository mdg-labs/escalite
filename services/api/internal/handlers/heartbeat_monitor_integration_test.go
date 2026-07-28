package handlers_test

import (
	"context"
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/audit"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func TestGraphQLHeartbeatMonitorCRUD(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")

		createRec := postGraphQL(t, handler, `mutation {
		createHeartbeatMonitor(input: {
			serviceId: "`+service.ID.String()+`"
			name: "Nightly backup"
			intervalSeconds: 3600
			graceSeconds: 300
		}) {
			id
			name
			serviceId
			intervalSeconds
			graceSeconds
			status
			tokenPrefix
			token
		}
	}`, adminCookie)
		require.Equal(a, 200, createRec.Code, createRec.Body.String())

		var createResp struct {
			Data struct {
				CreateHeartbeatMonitor struct {
					ID              string `json:"id"`
					Name            string `json:"name"`
					ServiceID       string `json:"serviceId"`
					IntervalSeconds int    `json:"intervalSeconds"`
					GraceSeconds    int    `json:"graceSeconds"`
					Status          string `json:"status"`
					TokenPrefix     string `json:"tokenPrefix"`
					Token           string `json:"token"`
				} `json:"createHeartbeatMonitor"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(createRec.Body.Bytes(), &createResp))
		require.Equal(a, "Nightly backup", createResp.Data.CreateHeartbeatMonitor.Name)
		require.Equal(a, service.ID.String(), createResp.Data.CreateHeartbeatMonitor.ServiceID)
		require.Equal(a, 3600, createResp.Data.CreateHeartbeatMonitor.IntervalSeconds)
		require.Equal(a, 300, createResp.Data.CreateHeartbeatMonitor.GraceSeconds)
		require.Equal(a, "HEALTHY", createResp.Data.CreateHeartbeatMonitor.Status)
		require.NotEmpty(a, createResp.Data.CreateHeartbeatMonitor.TokenPrefix)
		require.NotEmpty(a, createResp.Data.CreateHeartbeatMonitor.Token)

		monitorID := createResp.Data.CreateHeartbeatMonitor.ID

		events, err := queries.ListAuditEventsByOrganization(context.Background(), admin.OrganizationID)
		require.NoError(a, err)
		foundCreate := false
		for _, event := range events {
			if event.Action == audit.ActionHeartbeatMonitorCreated {
				foundCreate = true
				break
			}
		}
		require.True(a, foundCreate, "expected heartbeat_monitor.created audit event")

		invalidRec := postGraphQL(t, handler, `mutation {
		createHeartbeatMonitor(input: {
			serviceId: "`+service.ID.String()+`"
			name: "Broken"
			intervalSeconds: 0
			graceSeconds: 60
		}) { id }
	}`, adminCookie)
		require.Equal(a, 200, invalidRec.Code)

		var invalidResp struct {
			Errors []struct {
				Extensions map[string]interface{} `json:"extensions"`
			} `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(invalidRec.Body.Bytes(), &invalidResp))
		require.NotEmpty(a, invalidResp.Errors)
		require.Equal(a, handlers.CodeValidation, invalidResp.Errors[0].Extensions["code"])

		listRec := postGraphQL(t, handler, `{
		heartbeatMonitors(serviceId: "`+service.ID.String()+`") {
			id
			name
			intervalSeconds
			graceSeconds
			status
			token
		}
	}`, adminCookie)
		require.Equal(a, 200, listRec.Code, listRec.Body.String())

		var listResp struct {
			Data struct {
				HeartbeatMonitors []struct {
					ID              string  `json:"id"`
					Name            string  `json:"name"`
					IntervalSeconds int     `json:"intervalSeconds"`
					GraceSeconds    int     `json:"graceSeconds"`
					Status          string  `json:"status"`
					Token           *string `json:"token"`
				} `json:"heartbeatMonitors"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(listRec.Body.Bytes(), &listResp))
		require.Len(a, listResp.Data.HeartbeatMonitors, 1)
		require.Equal(a, monitorID, listResp.Data.HeartbeatMonitors[0].ID)
		require.Nil(a, listResp.Data.HeartbeatMonitors[0].Token)

		updateRec := postGraphQL(t, handler, `mutation {
		updateHeartbeatMonitor(input: {
			id: "`+monitorID+`"
			name: "Updated backup"
			intervalSeconds: 7200
			graceSeconds: 600
		}) {
			id
			name
			intervalSeconds
			graceSeconds
		}
	}`, adminCookie)
		require.Equal(a, 200, updateRec.Code, updateRec.Body.String())

		var updateResp struct {
			Data struct {
				UpdateHeartbeatMonitor struct {
					Name            string `json:"name"`
					IntervalSeconds int    `json:"intervalSeconds"`
					GraceSeconds    int    `json:"graceSeconds"`
				} `json:"updateHeartbeatMonitor"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(updateRec.Body.Bytes(), &updateResp))
		require.Equal(a, "Updated backup", updateResp.Data.UpdateHeartbeatMonitor.Name)
		require.Equal(a, 7200, updateResp.Data.UpdateHeartbeatMonitor.IntervalSeconds)
		require.Equal(a, 600, updateResp.Data.UpdateHeartbeatMonitor.GraceSeconds)

		events, err = queries.ListAuditEventsByOrganization(context.Background(), admin.OrganizationID)
		require.NoError(a, err)
		foundUpdate := false
		for _, event := range events {
			if event.Action == audit.ActionHeartbeatMonitorUpdated {
				foundUpdate = true
				break
			}
		}
		require.True(a, foundUpdate, "expected heartbeat_monitor.updated audit event")

		deleteRec := postGraphQL(t, handler, `mutation {
		deleteHeartbeatMonitor(id: "`+monitorID+`")
	}`, adminCookie)
		require.Equal(a, 200, deleteRec.Code, deleteRec.Body.String())

		var deleteResp struct {
			Data struct {
				DeleteHeartbeatMonitor bool `json:"deleteHeartbeatMonitor"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(deleteRec.Body.Bytes(), &deleteResp))
		require.True(a, deleteResp.Data.DeleteHeartbeatMonitor)

		events, err = queries.ListAuditEventsByOrganization(context.Background(), admin.OrganizationID)
		require.NoError(a, err)
		foundDelete := false
		for _, event := range events {
			if event.Action == audit.ActionHeartbeatMonitorDeleted {
				foundDelete = true
				break
			}
		}
		require.True(a, foundDelete, "expected heartbeat_monitor.deleted audit event")
	})
}

func TestGraphQLHeartbeatMonitorRequiresAdmin(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		_ = bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")
		member := seedMemberUser(t, pool, admin.OrganizationID, "member@example.com", "member-password-123")
		memberCookie := loginUser(t, handler, member.Email, "member-password-123")

		rec := postGraphQL(t, handler, `mutation {
		createHeartbeatMonitor(input: {
			serviceId: "`+service.ID.String()+`"
			name: "Nightly backup"
			intervalSeconds: 3600
			graceSeconds: 300
		}) { id }
	}`, memberCookie)
		require.Equal(a, 200, rec.Code)

		var resp struct {
			Errors []struct {
				Extensions map[string]interface{} `json:"extensions"`
			} `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.NotEmpty(a, resp.Errors)
		require.Equal(a, handlers.CodeForbidden, resp.Errors[0].Extensions["code"])
	})
}
