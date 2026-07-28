package handlers_test

import (
	"context"
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mdg-labs/escalite/services/api/internal/authz"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func seedSecondOrganizationWithMembership(
	t *testing.T,
	pool db.DBTX,
	accountID uuid.UUID,
	email string,
	orgName string,
	role string,
) (db.Organization, db.User) {
	t.Helper()

	queries := db.New(pool)
	ctx := context.Background()

	org, err := queries.CreateOrganization(ctx, db.CreateOrganizationParams{
		ID:   uuid.Must(uuid.NewV7()),
		Name: orgName,
	})
	require.NoError(t, err)

	user, err := queries.CreateUser(ctx, db.CreateUserParams{
		ID:             uuid.Must(uuid.NewV7()),
		AccountID:      accountID,
		OrganizationID: org.ID,
		Email:          email,
		Role:           role,
	})
	require.NoError(t, err)

	return org, user
}

func TestMultiOrgMemberCannotSwitchToUnaffiliatedOrg(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		otherOrg, err := queries.CreateOrganization(context.Background(), db.CreateOrganizationParams{
			ID:   uuid.Must(uuid.NewV7()),
			Name: "Org B",
		})
		require.NoError(a, err)

		strangerAccount, err := queries.CreateAccount(context.Background(), db.CreateAccountParams{
			ID:           uuid.Must(uuid.NewV7()),
			Email:        "stranger@example.com",
			PasswordHash: pgtype.Text{},
		})
		require.NoError(a, err)

		_, err = queries.CreateUser(context.Background(), db.CreateUserParams{
			ID:             uuid.Must(uuid.NewV7()),
			AccountID:      strangerAccount.ID,
			OrganizationID: otherOrg.ID,
			Email:          strangerAccount.Email,
			Role:           authz.RoleMember,
		})
		require.NoError(a, err)

		rec := postGraphQL(t, handler, `mutation {
		switchOrganization(organizationId: "`+otherOrg.ID.String()+`") {
			user { id }
		}
	}`, adminCookie)
		require.Equal(a, 200, rec.Code)

		var resp struct {
			Errors []struct {
				Extensions map[string]interface{} `json:"extensions"`
			} `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.NotEmpty(a, resp.Errors)
		require.Equal(a, handlers.CodeForbidden, resp.Errors[0].Extensions["code"])
		_ = admin
	})
}

func TestMultiOrgAdminRoleIsPerOrganization(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		orgB, _ := seedSecondOrganizationWithMembership(
			t, pool, admin.AccountID, admin.Email, "Org B", authz.RoleMember,
		)

		switchRec := postGraphQL(t, handler, `mutation {
		switchOrganization(organizationId: "`+orgB.ID.String()+`") {
			user { role organizationId }
		}
	}`, adminCookie)
		require.Equal(a, 200, switchRec.Code)

		var switchResp struct {
			Data struct {
				SwitchOrganization struct {
					User struct {
						Role           string `json:"role"`
						OrganizationID string `json:"organizationId"`
					} `json:"user"`
				} `json:"switchOrganization"`
			} `json:"data"`
			Errors []struct {
				Extensions map[string]interface{} `json:"extensions"`
			} `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(switchRec.Body.Bytes(), &switchResp))
		require.Empty(a, switchResp.Errors)
		require.Equal(a, "MEMBER", switchResp.Data.SwitchOrganization.User.Role)
		require.Equal(a, orgB.ID.String(), switchResp.Data.SwitchOrganization.User.OrganizationID)

		teamsRec := postGraphQL(t, handler, `{ teams { id name } }`, adminCookie)
		require.Equal(a, 200, teamsRec.Code)

		var teamsResp struct {
			Errors []struct {
				Message    string                 `json:"message"`
				Extensions map[string]interface{} `json:"extensions"`
			} `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(teamsRec.Body.Bytes(), &teamsResp))
		require.NotEmpty(a, teamsResp.Errors)
		require.Equal(a, handlers.CodeForbidden, teamsResp.Errors[0].Extensions["code"])
	})
}

func TestMyOrganizationsListsMembershipsOnly(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		seedSecondOrganizationWithMembership(
			t, pool, admin.AccountID, admin.Email, "Org B", authz.RoleMember,
		)

		otherAccount, err := queries.CreateAccount(context.Background(), db.CreateAccountParams{
			ID:           uuid.Must(uuid.NewV7()),
			Email:        "other@example.com",
			PasswordHash: pgtype.Text{},
		})
		require.NoError(a, err)
		otherOrg, err := queries.CreateOrganization(context.Background(), db.CreateOrganizationParams{
			ID:   uuid.Must(uuid.NewV7()),
			Name: "Other Org",
		})
		require.NoError(a, err)
		_, err = queries.CreateUser(context.Background(), db.CreateUserParams{
			ID:             uuid.Must(uuid.NewV7()),
			AccountID:      otherAccount.ID,
			OrganizationID: otherOrg.ID,
			Email:          otherAccount.Email,
			Role:           authz.RoleAdmin,
		})
		require.NoError(a, err)

		rec := postGraphQL(t, handler, `{
		myOrganizations {
			organization { name }
			role
		}
	}`, adminCookie)
		require.Equal(a, 200, rec.Code)

		var resp struct {
			Data struct {
				MyOrganizations []struct {
					Organization struct {
						Name string `json:"name"`
					} `json:"organization"`
					Role string `json:"role"`
				} `json:"myOrganizations"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Len(a, resp.Data.MyOrganizations, 2)

		names := []string{
			resp.Data.MyOrganizations[0].Organization.Name,
			resp.Data.MyOrganizations[1].Organization.Name,
		}
		require.Contains(a, names, "Acme On-Call")
		require.Contains(a, names, "Org B")
		require.NotContains(a, names, "Other Org")
	})
}
