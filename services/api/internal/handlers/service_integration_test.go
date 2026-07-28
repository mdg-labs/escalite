package handlers_test

import (
	"context"
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/audit"
	"github.com/mdg-labs/escalite/services/api/internal/db"
)

func TestGraphQLServiceCRUD(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")

		createRec := postGraphQL(t, handler, `mutation {
		createService(input: {
			teamId: "`+team.ID.String()+`"
			name: "Payments API"
		}) {
			id
			name
			teamId
		}
	}`, adminCookie)
		require.Equal(a, 200, createRec.Code, createRec.Body.String())

		var createResp struct {
			Data struct {
				CreateService struct {
					ID     string `json:"id"`
					Name   string `json:"name"`
					TeamID string `json:"teamId"`
				} `json:"createService"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(createRec.Body.Bytes(), &createResp))
		require.Equal(a, "Payments API", createResp.Data.CreateService.Name)
		require.Equal(a, team.ID.String(), createResp.Data.CreateService.TeamID)

		serviceID := createResp.Data.CreateService.ID

		listRec := postGraphQL(t, handler, `query {
		services {
			id
			name
		}
	}`, adminCookie)
		require.Equal(a, 200, listRec.Code, listRec.Body.String())

		var listResp struct {
			Data struct {
				Services []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"services"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(listRec.Body.Bytes(), &listResp))
		require.Len(a, listResp.Data.Services, 1)
		require.Equal(a, serviceID, listResp.Data.Services[0].ID)

		otherTeam := seedTeam(t, pool, admin.OrganizationID, "Infrastructure")

		reassignRec := postGraphQL(t, handler, `mutation {
		updateService(input: {
			id: "`+serviceID+`"
			teamId: "`+otherTeam.ID.String()+`"
		}) {
			id
			teamId
		}
	}`, adminCookie)
		require.Equal(a, 200, reassignRec.Code, reassignRec.Body.String())

		var reassignResp struct {
			Data struct {
				UpdateService struct {
					ID     string `json:"id"`
					TeamID string `json:"teamId"`
				} `json:"updateService"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(reassignRec.Body.Bytes(), &reassignResp))
		require.Equal(a, serviceID, reassignResp.Data.UpdateService.ID)
		require.Equal(a, otherTeam.ID.String(), reassignResp.Data.UpdateService.TeamID)

		updateRec := postGraphQL(t, handler, `mutation {
		updateService(input: {
			id: "`+serviceID+`"
			name: "Payments API v2"
		}) {
			id
			name
		}
	}`, adminCookie)
		require.Equal(a, 200, updateRec.Code, updateRec.Body.String())

		deleteRec := postGraphQL(t, handler, `mutation {
		deleteService(id: "`+serviceID+`")
	}`, adminCookie)
		require.Equal(a, 200, deleteRec.Code, deleteRec.Body.String())

		var deleteResp struct {
			Data struct {
				DeleteService bool `json:"deleteService"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(deleteRec.Body.Bytes(), &deleteResp))
		require.True(a, deleteResp.Data.DeleteService)

		listAfterDeleteRec := postGraphQL(t, handler, `query {
		services {
			id
		}
	}`, adminCookie)
		require.Equal(a, 200, listAfterDeleteRec.Code, listAfterDeleteRec.Body.String())

		var listAfterDeleteResp struct {
			Data struct {
				Services []struct {
					ID string `json:"id"`
				} `json:"services"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(listAfterDeleteRec.Body.Bytes(), &listAfterDeleteResp))
		require.Empty(a, listAfterDeleteResp.Data.Services)

		var deletedAtValid bool
		err = pool.QueryRow(context.Background(),
			`SELECT deleted_at IS NOT NULL FROM services WHERE id = $1`,
			serviceID,
		).Scan(&deletedAtValid)
		require.NoError(a, err)
		require.True(a, deletedAtValid)

		events, err := queries.ListAuditEventsByOrganization(context.Background(), admin.OrganizationID)
		require.NoError(a, err)

		var sawCreated, sawUpdated, sawDeleted bool
		for _, event := range events {
			switch event.Action {
			case audit.ActionServiceCreated:
				sawCreated = true
			case audit.ActionServiceUpdated:
				sawUpdated = true
			case audit.ActionServiceDeleted:
				sawDeleted = true
			}
		}
		require.True(a, sawCreated)
		require.True(a, sawUpdated)
		require.True(a, sawDeleted)
	})
}
