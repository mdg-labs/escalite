package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/authz"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func loginUser(t *testing.T, handler http.Handler, email, password string) *http.Cookie {
	t.Helper()

	payload, err := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	return findSessionCookie(t, rec)
}

func seedMemberUser(t *testing.T, pool db.DBTX, orgID uuid.UUID, email, password string) db.User {
	t.Helper()

	passwordHash, err := auth.HashPassword(password)
	require.NoError(t, err)

	queries := db.New(pool)
	account, err := queries.CreateAccount(context.Background(), db.CreateAccountParams{
		ID:           uuid.Must(uuid.NewV7()),
		Email:        email,
		PasswordHash: pgtype.Text{String: passwordHash, Valid: true},
	})
	require.NoError(t, err)

	user, err := queries.CreateUser(context.Background(), db.CreateUserParams{
		ID:             uuid.Must(uuid.NewV7()),
		AccountID:      account.ID,
		OrganizationID: orgID,
		Email:          email,
		Role:           authz.RoleMember,
	})
	require.NoError(t, err)
	return user
}

func seedTeam(t *testing.T, pool db.DBTX, orgID uuid.UUID, name string) db.Team {
	t.Helper()

	queries := db.New(pool)
	team, err := queries.CreateTeam(context.Background(), db.CreateTeamParams{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: orgID,
		Name:           name,
	})
	require.NoError(t, err)
	return team
}

func seedTeamMembership(t *testing.T, pool db.DBTX, orgID, teamID, userID uuid.UUID) {
	t.Helper()

	queries := db.New(pool)
	_, err := queries.CreateTeamMembership(context.Background(), db.CreateTeamMembershipParams{
		ID:             uuid.Must(uuid.NewV7()),
		TeamID:         teamID,
		UserID:         userID,
		OrganizationID: orgID,
	})
	require.NoError(t, err)
}

func getTeam(t *testing.T, handler http.Handler, cookie *http.Cookie, teamID uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+teamID.String(), nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestRBACAdminCanAccessAllTeams(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)
		orgID := admin.OrganizationID

		teamA := seedTeam(t, pool, orgID, "Platform")
		teamB := seedTeam(t, pool, orgID, "Payments")

		recA := getTeam(t, handler, adminCookie, teamA.ID)
		require.Equal(a, http.StatusOK, recA.Code, recA.Body.String())

		var teamResp struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		require.NoError(a, json.Unmarshal(recA.Body.Bytes(), &teamResp))
		require.Equal(a, teamA.ID.String(), teamResp.ID)
		require.Equal(a, "Platform", teamResp.Name)

		recB := getTeam(t, handler, adminCookie, teamB.ID)
		require.Equal(a, http.StatusOK, recB.Code, recB.Body.String())
	})
}

func TestRBACMemberTeamAccess(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		_ = bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)
		orgID := admin.OrganizationID

		teamA := seedTeam(t, pool, orgID, "Platform")
		teamB := seedTeam(t, pool, orgID, "Payments")

		member := seedMemberUser(t, pool, orgID, "member@example.com", "member-password-123")
		seedTeamMembership(t, pool, orgID, teamA.ID, member.ID)

		memberCookie := loginUser(t, handler, "member@example.com", "member-password-123")

		allowed := getTeam(t, handler, memberCookie, teamA.ID)
		require.Equal(a, http.StatusOK, allowed.Code, allowed.Body.String())

		denied := getTeam(t, handler, memberCookie, teamB.ID)
		require.Equal(a, http.StatusForbidden, denied.Code, denied.Body.String())

		var errResp struct {
			Error string `json:"error"`
			Code  string `json:"code"`
		}
		require.NoError(a, json.Unmarshal(denied.Body.Bytes(), &errResp))
		require.Equal(a, handlers.CodeForbidden, errResp.Code)
		require.Equal(a, "access denied", errResp.Error)
	})
}

func TestRBACMemberCannotAccessUnknownTeam(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		_ = bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		member := seedMemberUser(t, pool, admin.OrganizationID, "member@example.com", "member-password-123")
		memberCookie := loginUser(t, handler, member.Email, "member-password-123")

		unknownTeamID := uuid.Must(uuid.NewV7())
		rec := getTeam(t, handler, memberCookie, unknownTeamID)
		require.Equal(a, http.StatusNotFound, rec.Code, rec.Body.String())

		var errResp struct {
			Code string `json:"code"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &errResp))
		require.Equal(a, handlers.CodeNotFound, errResp.Code)
	})
}

func TestRBACAdminCanAddAndRemoveTeamMember(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)
		orgID := admin.OrganizationID

		team := seedTeam(t, pool, orgID, "Platform")
		member := seedMemberUser(t, pool, orgID, "member@example.com", "member-password-123")

		addRec := postGraphQL(t, handler, `mutation {
		addTeamMember(teamId: "`+team.ID.String()+`", userId: "`+member.ID.String()+`") {
			id
			teamId
			userId
		}
	}`, adminCookie)
		require.Equal(a, http.StatusOK, addRec.Code, addRec.Body.String())

		var addResp struct {
			Data struct {
				AddTeamMember struct {
					ID     string `json:"id"`
					TeamID string `json:"teamId"`
					UserID string `json:"userId"`
				} `json:"addTeamMember"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(addRec.Body.Bytes(), &addResp))
		require.Equal(a, team.ID.String(), addResp.Data.AddTeamMember.TeamID)
		require.Equal(a, member.ID.String(), addResp.Data.AddTeamMember.UserID)

		memberCookie := loginUser(t, handler, member.Email, "member-password-123")
		allowed := getTeam(t, handler, memberCookie, team.ID)
		require.Equal(a, http.StatusOK, allowed.Code, allowed.Body.String())

		removeRec := postGraphQL(t, handler, `mutation {
		removeTeamMember(teamId: "`+team.ID.String()+`", userId: "`+member.ID.String()+`")
	}`, adminCookie)
		require.Equal(a, http.StatusOK, removeRec.Code, removeRec.Body.String())

		var removeResp struct {
			Data struct {
				RemoveTeamMember bool `json:"removeTeamMember"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(removeRec.Body.Bytes(), &removeResp))
		require.True(a, removeResp.Data.RemoveTeamMember)

		denied := getTeam(t, handler, memberCookie, team.ID)
		require.Equal(a, http.StatusForbidden, denied.Code, denied.Body.String())
	})
}

func TestRBACRemoveTeamMemberRejectsLastOrgAdmin(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		seedTeamMembership(t, pool, admin.OrganizationID, team.ID, admin.ID)

		removeRec := postGraphQL(t, handler, `mutation {
		removeTeamMember(teamId: "`+team.ID.String()+`", userId: "`+admin.ID.String()+`")
	}`, adminCookie)
		require.Equal(a, http.StatusOK, removeRec.Code, removeRec.Body.String())

		var resp struct {
			Errors []struct {
				Message    string                 `json:"message"`
				Extensions map[string]interface{} `json:"extensions"`
			} `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(removeRec.Body.Bytes(), &resp))
		require.NotEmpty(a, resp.Errors)
		require.Equal(a, handlers.CodeValidation, resp.Errors[0].Extensions["code"])
		require.Equal(a, "cannot remove last admin from org", resp.Errors[0].Message)
	})
}

func TestRBACRemoveTeamMemberAllowsWhenSecondAdminExists(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		secondAdminAccount, err := queries.CreateAccount(context.Background(), db.CreateAccountParams{
			ID:           uuid.Must(uuid.NewV7()),
			Email:        "admin2@example.com",
			PasswordHash: pgtype.Text{},
		})
		require.NoError(a, err)

		secondAdmin, err := queries.CreateUser(context.Background(), db.CreateUserParams{
			ID:             uuid.Must(uuid.NewV7()),
			AccountID:      secondAdminAccount.ID,
			OrganizationID: admin.OrganizationID,
			Email:          secondAdminAccount.Email,
			Role:           authz.RoleAdmin,
		})
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		seedTeamMembership(t, pool, admin.OrganizationID, team.ID, admin.ID)

		removeRec := postGraphQL(t, handler, `mutation {
		removeTeamMember(teamId: "`+team.ID.String()+`", userId: "`+admin.ID.String()+`")
	}`, adminCookie)
		require.Equal(a, http.StatusOK, removeRec.Code, removeRec.Body.String())

		var removeResp struct {
			Data struct {
				RemoveTeamMember bool `json:"removeTeamMember"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(removeRec.Body.Bytes(), &removeResp))
		require.True(a, removeResp.Data.RemoveTeamMember)
		_ = secondAdmin
	})
}

func TestRBACMemberCannotManageTeamMembership(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		_ = bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		member := seedMemberUser(t, pool, admin.OrganizationID, "member@example.com", "member-password-123")
		otherMember := seedMemberUser(t, pool, admin.OrganizationID, "member2@example.com", "member-password-123")
		memberCookie := loginUser(t, handler, member.Email, "member-password-123")

		addRec := postGraphQL(t, handler, `mutation {
		addTeamMember(teamId: "`+team.ID.String()+`", userId: "`+otherMember.ID.String()+`") {
			id
		}
	}`, memberCookie)
		require.Equal(a, http.StatusOK, addRec.Code)

		var addResp struct {
			Errors []struct {
				Extensions map[string]interface{} `json:"extensions"`
			} `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(addRec.Body.Bytes(), &addResp))
		require.NotEmpty(a, addResp.Errors)
		require.Equal(a, handlers.CodeForbidden, addResp.Errors[0].Extensions["code"])
		_ = otherMember
	})
}

func TestRBACOrganizationUsersAdminSeesAll(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		memberA := seedMemberUser(t, pool, admin.OrganizationID, "member-a@example.com", "member-password-123")
		memberB := seedMemberUser(t, pool, admin.OrganizationID, "member-b@example.com", "member-password-123")
		seedTeamMembership(t, pool, admin.OrganizationID, team.ID, memberA.ID)

		rec := postGraphQL(t, handler, `{
		organizationUsers {
			id
			email
			role
			teamMemberships { teamId userId }
		}
	}`, adminCookie)
		require.Equal(a, http.StatusOK, rec.Code, rec.Body.String())

		var resp struct {
			Data struct {
				OrganizationUsers []struct {
					ID              string `json:"id"`
					Email           string `json:"email"`
					Role            string `json:"role"`
					TeamMemberships []struct {
						TeamID string `json:"teamId"`
						UserID string `json:"userId"`
					} `json:"teamMemberships"`
				} `json:"organizationUsers"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))

		emails := make([]string, 0, len(resp.Data.OrganizationUsers))
		for _, user := range resp.Data.OrganizationUsers {
			emails = append(emails, user.Email)
		}
		require.Contains(a, emails, "admin@example.com")
		require.Contains(a, emails, memberA.Email)
		require.Contains(a, emails, memberB.Email)

		var memberAEntry *struct {
			ID              string `json:"id"`
			Email           string `json:"email"`
			Role            string `json:"role"`
			TeamMemberships []struct {
				TeamID string `json:"teamId"`
				UserID string `json:"userId"`
			} `json:"teamMemberships"`
		}
		for i := range resp.Data.OrganizationUsers {
			if resp.Data.OrganizationUsers[i].Email == memberA.Email {
				memberAEntry = &resp.Data.OrganizationUsers[i]
				break
			}
		}
		require.NotNil(a, memberAEntry)
		require.Equal(a, "MEMBER", memberAEntry.Role)
		require.Len(a, memberAEntry.TeamMemberships, 1)
		require.Equal(a, team.ID.String(), memberAEntry.TeamMemberships[0].TeamID)
		require.Equal(a, memberA.ID.String(), memberAEntry.TeamMemberships[0].UserID)
	})
}

func TestRBACOrganizationUsersMemberSeesSharedTeamsOnly(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		_ = bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		teamShared := seedTeam(t, pool, admin.OrganizationID, "Platform")
		teamOther := seedTeam(t, pool, admin.OrganizationID, "Payments")

		memberA := seedMemberUser(t, pool, admin.OrganizationID, "member-a@example.com", "member-password-123")
		memberB := seedMemberUser(t, pool, admin.OrganizationID, "member-b@example.com", "member-password-123")
		memberC := seedMemberUser(t, pool, admin.OrganizationID, "member-c@example.com", "member-password-123")

		seedTeamMembership(t, pool, admin.OrganizationID, teamShared.ID, memberA.ID)
		seedTeamMembership(t, pool, admin.OrganizationID, teamShared.ID, memberB.ID)
		seedTeamMembership(t, pool, admin.OrganizationID, teamOther.ID, memberC.ID)

		memberACookie := loginUser(t, handler, memberA.Email, "member-password-123")

		rec := postGraphQL(t, handler, `{
		organizationUsers {
			email
		}
	}`, memberACookie)
		require.Equal(a, http.StatusOK, rec.Code, rec.Body.String())

		var resp struct {
			Data struct {
				OrganizationUsers []struct {
					Email string `json:"email"`
				} `json:"organizationUsers"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))

		emails := make([]string, 0, len(resp.Data.OrganizationUsers))
		for _, user := range resp.Data.OrganizationUsers {
			emails = append(emails, user.Email)
		}
		require.Contains(a, emails, memberA.Email)
		require.Contains(a, emails, memberB.Email)
		require.NotContains(a, emails, memberC.Email)
		require.NotContains(a, emails, admin.Email)
	})
}
