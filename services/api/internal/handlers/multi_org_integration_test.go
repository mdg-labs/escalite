package handlers_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"

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
	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)

	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	otherOrg, err := queries.CreateOrganization(context.Background(), db.CreateOrganizationParams{
		ID:   uuid.Must(uuid.NewV7()),
		Name: "Org B",
	})
	require.NoError(t, err)

	strangerAccount, err := queries.CreateAccount(context.Background(), db.CreateAccountParams{
		ID:           uuid.Must(uuid.NewV7()),
		Email:        "stranger@example.com",
		PasswordHash: pgtype.Text{},
	})
	require.NoError(t, err)

	_, err = queries.CreateUser(context.Background(), db.CreateUserParams{
		ID:             uuid.Must(uuid.NewV7()),
		AccountID:      strangerAccount.ID,
		OrganizationID: otherOrg.ID,
		Email:          strangerAccount.Email,
		Role:           authz.RoleMember,
	})
	require.NoError(t, err)

	rec := postGraphQL(t, handler, `mutation {
		switchOrganization(organizationId: "`+otherOrg.ID.String()+`") {
			user { id }
		}
	}`, adminCookie)
	require.Equal(t, 200, rec.Code)

	var resp struct {
		Errors []struct {
			Extensions map[string]interface{} `json:"extensions"`
		} `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Errors)
	require.Equal(t, handlers.CodeForbidden, resp.Errors[0].Extensions["code"])
	_ = admin
}

func TestMultiOrgAdminRoleIsPerOrganization(t *testing.T) {
	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)

	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	orgB, _ := seedSecondOrganizationWithMembership(
		t, pool, admin.AccountID, admin.Email, "Org B", authz.RoleMember,
	)

	switchRec := postGraphQL(t, handler, `mutation {
		switchOrganization(organizationId: "`+orgB.ID.String()+`") {
			user { role organizationId }
		}
	}`, adminCookie)
	require.Equal(t, 200, switchRec.Code)

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
	require.NoError(t, json.Unmarshal(switchRec.Body.Bytes(), &switchResp))
	require.Empty(t, switchResp.Errors)
	require.Equal(t, "MEMBER", switchResp.Data.SwitchOrganization.User.Role)
	require.Equal(t, orgB.ID.String(), switchResp.Data.SwitchOrganization.User.OrganizationID)

	teamsRec := postGraphQL(t, handler, `{ teams { id name } }`, adminCookie)
	require.Equal(t, 200, teamsRec.Code)

	var teamsResp struct {
		Errors []struct {
			Message    string                 `json:"message"`
			Extensions map[string]interface{} `json:"extensions"`
		} `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(teamsRec.Body.Bytes(), &teamsResp))
	require.NotEmpty(t, teamsResp.Errors)
	require.Equal(t, handlers.CodeForbidden, teamsResp.Errors[0].Extensions["code"])
}

func TestMyOrganizationsListsMembershipsOnly(t *testing.T) {
	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)

	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	seedSecondOrganizationWithMembership(
		t, pool, admin.AccountID, admin.Email, "Org B", authz.RoleMember,
	)

	otherAccount, err := queries.CreateAccount(context.Background(), db.CreateAccountParams{
		ID:           uuid.Must(uuid.NewV7()),
		Email:        "other@example.com",
		PasswordHash: pgtype.Text{},
	})
	require.NoError(t, err)
	otherOrg, err := queries.CreateOrganization(context.Background(), db.CreateOrganizationParams{
		ID:   uuid.Must(uuid.NewV7()),
		Name: "Other Org",
	})
	require.NoError(t, err)
	_, err = queries.CreateUser(context.Background(), db.CreateUserParams{
		ID:             uuid.Must(uuid.NewV7()),
		AccountID:      otherAccount.ID,
		OrganizationID: otherOrg.ID,
		Email:          otherAccount.Email,
		Role:           authz.RoleAdmin,
	})
	require.NoError(t, err)

	rec := postGraphQL(t, handler, `{
		myOrganizations {
			organization { name }
			role
		}
	}`, adminCookie)
	require.Equal(t, 200, rec.Code)

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
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Data.MyOrganizations, 2)

	names := []string{
		resp.Data.MyOrganizations[0].Organization.Name,
		resp.Data.MyOrganizations[1].Organization.Name,
	}
	require.Contains(t, names, "Acme On-Call")
	require.Contains(t, names, "Org B")
	require.NotContains(t, names, "Other Org")
}
