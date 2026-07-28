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

func TestGraphQLUpdateOrganization(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		updateRec := postGraphQL(t, handler, `mutation {
		updateOrganization(input: { name: "Acme Corp" }) {
			id
			name
		}
	}`, adminCookie)
		require.Equal(a, 200, updateRec.Code, updateRec.Body.String())

		var updateResp struct {
			Data struct {
				UpdateOrganization struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"updateOrganization"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(updateRec.Body.Bytes(), &updateResp))
		require.Equal(a, admin.OrganizationID.String(), updateResp.Data.UpdateOrganization.ID)
		require.Equal(a, "Acme Corp", updateResp.Data.UpdateOrganization.Name)

		events, err := queries.ListAuditEventsByOrganization(context.Background(), admin.OrganizationID)
		require.NoError(a, err)

		foundUpdate := false
		for _, event := range events {
			if event.Action == audit.ActionOrganizationUpdated {
				foundUpdate = true
				break
			}
		}
		require.True(a, foundUpdate)
	})
}

func TestGraphQLUpdateOrganizationRejectsEmptyName(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		rec := postGraphQL(t, handler, `mutation {
		updateOrganization(input: { name: "   " }) {
			id
		}
	}`, adminCookie)
		require.Equal(a, 200, rec.Code)

		var resp struct {
			Errors []struct {
				Message    string                 `json:"message"`
				Extensions map[string]interface{} `json:"extensions"`
			} `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.NotEmpty(a, resp.Errors)
		require.Equal(a, handlers.CodeValidation, resp.Errors[0].Extensions["code"])
	})
}

func TestGraphQLUpdateOrganizationRequiresAdmin(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		_ = bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		_ = seedMemberUser(t, pool, admin.OrganizationID, "member@example.com", "member-password-123")
		memberCookie := loginUser(t, handler, "member@example.com", "member-password-123")

		rec := postGraphQL(t, handler, `mutation {
		updateOrganization(input: { name: "Renamed Org" }) {
			id
		}
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
