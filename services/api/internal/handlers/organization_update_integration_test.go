package handlers_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/audit"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func TestGraphQLUpdateOrganization(t *testing.T) {
	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)

	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	updateRec := postGraphQL(t, handler, `mutation {
		updateOrganization(input: { name: "Acme Corp" }) {
			id
			name
		}
	}`, adminCookie)
	require.Equal(t, 200, updateRec.Code, updateRec.Body.String())

	var updateResp struct {
		Data struct {
			UpdateOrganization struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"updateOrganization"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(updateRec.Body.Bytes(), &updateResp))
	require.Equal(t, admin.OrganizationID.String(), updateResp.Data.UpdateOrganization.ID)
	require.Equal(t, "Acme Corp", updateResp.Data.UpdateOrganization.Name)

	events, err := queries.ListAuditEventsByOrganization(context.Background(), admin.OrganizationID)
	require.NoError(t, err)

	foundUpdate := false
	for _, event := range events {
		if event.Action == audit.ActionOrganizationUpdated {
			foundUpdate = true
			break
		}
	}
	require.True(t, foundUpdate)
}

func TestGraphQLUpdateOrganizationRejectsEmptyName(t *testing.T) {
	handler, _, cleanup := newTestHandler(t)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)

	rec := postGraphQL(t, handler, `mutation {
		updateOrganization(input: { name: "   " }) {
			id
		}
	}`, adminCookie)
	require.Equal(t, 200, rec.Code)

	var resp struct {
		Errors []struct {
			Message    string                 `json:"message"`
			Extensions map[string]interface{} `json:"extensions"`
		} `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Errors)
	require.Equal(t, handlers.CodeValidation, resp.Errors[0].Extensions["code"])
}

func TestGraphQLUpdateOrganizationRequiresAdmin(t *testing.T) {
	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	_ = bootstrapAdmin(t, handler)

	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	_ = seedMemberUser(t, pool, admin.OrganizationID, "member@example.com", "member-password-123")
	memberCookie := loginUser(t, handler, "member@example.com", "member-password-123")

	rec := postGraphQL(t, handler, `mutation {
		updateOrganization(input: { name: "Renamed Org" }) {
			id
		}
	}`, memberCookie)
	require.Equal(t, 200, rec.Code)

	var resp struct {
		Errors []struct {
			Extensions map[string]interface{} `json:"extensions"`
		} `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Errors)
	require.Equal(t, handlers.CodeForbidden, resp.Errors[0].Extensions["code"])
}
