package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/audit"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/email"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
	"github.com/mdg-labs/escalite/services/api/internal/server"
)

func userInviteGraphQLTestHandler(t *testing.T) (http.Handler, *email.RecordingSender, func()) {
	t.Helper()

	mail := &email.RecordingSender{}
	handler, _, cleanup := newTestHandlerWithOptions(t, testServerOptions{
		Mail:      mail,
		PublicURL: "http://localhost:5173",
		PasswordReset: &server.PasswordResetOptions{
			EmailLimiter: nil,
			IPLimiter:    nil,
		},
	})

	return handler, mail, cleanup
}

func TestGraphQLInviteUserAndLoginPath(t *testing.T) {
	handler, mail, cleanup := userInviteGraphQLTestHandler(t)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)

	inviteRec := postGraphQL(t, handler, `
		mutation {
			inviteUser(input: { email: "new.member@example.com", role: MEMBER }) {
				id
				email
				role
				teamMemberships { id }
			}
		}
	`, adminCookie)
	require.Equal(t, http.StatusOK, inviteRec.Code, inviteRec.Body.String())

	var inviteResp struct {
		Data struct {
			InviteUser struct {
				ID              string `json:"id"`
				Email           string `json:"email"`
				Role            string `json:"role"`
				TeamMemberships []any  `json:"teamMemberships"`
			} `json:"inviteUser"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(inviteRec.Body.Bytes(), &inviteResp))
	require.Empty(t, inviteResp.Errors)
	require.Equal(t, "new.member@example.com", inviteResp.Data.InviteUser.Email)
	require.Equal(t, "MEMBER", inviteResp.Data.InviteUser.Role)
	require.NotEmpty(t, inviteResp.Data.InviteUser.ID)
	require.Empty(t, inviteResp.Data.InviteUser.TeamMemberships)
	require.NotContains(t, inviteRec.Body.String(), "token=")

	require.Len(t, mail.Messages, 1)
	token := extractResetToken(t, mail.Messages[0].Body)
	newPassword := "brand-new-password-99"

	confirmRec := postGraphQL(t, handler, `
		mutation {
			resetPassword(input: { token: "`+token+`", password: "`+newPassword+`" })
		}
	`, nil)
	require.Equal(t, http.StatusOK, confirmRec.Code)

	var confirmResp struct {
		Data struct {
			ResetPassword bool `json:"resetPassword"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(confirmRec.Body.Bytes(), &confirmResp))
	require.True(t, confirmResp.Data.ResetPassword)

	loginRec := postGraphQL(t, handler, `
		mutation {
			login(input: { email: "new.member@example.com", password: "`+newPassword+`" }) {
				user { id email role }
			}
		}
	`, nil)
	require.Equal(t, http.StatusOK, loginRec.Code, loginRec.Body.String())

	var loginResp struct {
		Data struct {
			Login struct {
				User struct {
					ID    string `json:"id"`
					Email string `json:"email"`
					Role  string `json:"role"`
				} `json:"user"`
			} `json:"login"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(loginRec.Body.Bytes(), &loginResp))
	require.Equal(t, inviteResp.Data.InviteUser.ID, loginResp.Data.Login.User.ID)
	require.Equal(t, "new.member@example.com", loginResp.Data.Login.User.Email)
	require.Equal(t, "MEMBER", loginResp.Data.Login.User.Role)
}

func TestGraphQLInviteUserRejectsDuplicate(t *testing.T) {
	handler, _, cleanup := userInviteGraphQLTestHandler(t)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)

	first := postGraphQL(t, handler, `
		mutation {
			inviteUser(input: { email: "dup@example.com", role: MEMBER }) {
				email
			}
		}
	`, adminCookie)
	require.Equal(t, http.StatusOK, first.Code)

	second := postGraphQL(t, handler, `
		mutation {
			inviteUser(input: { email: "dup@example.com", role: MEMBER }) {
				email
			}
		}
	`, adminCookie)
	require.Equal(t, http.StatusOK, second.Code)
	require.Contains(t, second.Body.String(), "user already exists")
}

func TestGraphQLInviteUserMemberForbidden(t *testing.T) {
	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	_ = bootstrapAdmin(t, handler)
	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	member := seedMemberUser(t, pool, admin.OrganizationID, "member@example.com", "member-password-123")
	_ = member
	memberCookie := loginUser(t, handler, "member@example.com", "member-password-123")

	rec := postGraphQL(t, handler, `
		mutation {
			inviteUser(input: { email: "blocked@example.com", role: MEMBER }) {
				email
			}
		}
	`, memberCookie)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Errors []struct {
			Extensions map[string]interface{} `json:"extensions"`
		} `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Errors)
	require.Equal(t, handlers.CodeForbidden, resp.Errors[0].Extensions["code"])
}

func TestGraphQLUpdateUserRolePromotesMember(t *testing.T) {
	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)
	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	member := seedMemberUser(t, pool, admin.OrganizationID, "member@example.com", "member-password-123")

	updated := postGraphQL(t, handler, `
		mutation {
			updateUserRole(input: { userId: "`+member.ID.String()+`", role: ADMIN }) {
				email
				role
			}
		}
	`, adminCookie)
	require.Equal(t, http.StatusOK, updated.Code, updated.Body.String())

	var updatedResp struct {
		Data struct {
			UpdateUserRole struct {
				Email string `json:"email"`
				Role  string `json:"role"`
			} `json:"updateUserRole"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(updated.Body.Bytes(), &updatedResp))
	require.Empty(t, updatedResp.Errors)
	require.Equal(t, "member@example.com", updatedResp.Data.UpdateUserRole.Email)
	require.Equal(t, "ADMIN", updatedResp.Data.UpdateUserRole.Role)

	events, err := queries.ListAuditEventsByOrganization(context.Background(), admin.OrganizationID)
	require.NoError(t, err)

	foundRoleChange := false
	for _, event := range events {
		if event.Action == audit.ActionRoleChanged {
			foundRoleChange = true
			break
		}
	}
	require.True(t, foundRoleChange)
}

func TestGraphQLUpdateUserRoleCannotDemoteLastAdmin(t *testing.T) {
	handler, _, cleanup := userInviteGraphQLTestHandler(t)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)

	meRec := postGraphQL(t, handler, `{
		me { id role }
	}`, adminCookie)
	require.Equal(t, http.StatusOK, meRec.Code)

	var meResp struct {
		Data struct {
			Me struct {
				ID   string `json:"id"`
				Role string `json:"role"`
			} `json:"me"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(meRec.Body.Bytes(), &meResp))
	require.Equal(t, "ADMIN", meResp.Data.Me.Role)

	demote := postGraphQL(t, handler, `
		mutation {
			updateUserRole(input: { userId: "`+meResp.Data.Me.ID+`", role: MEMBER }) {
				role
			}
		}
	`, adminCookie)
	require.Equal(t, http.StatusOK, demote.Code)
	require.Contains(t, demote.Body.String(), "cannot demote last admin")
}

func TestGraphQLUpdateUserRoleMemberForbidden(t *testing.T) {
	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	_ = bootstrapAdmin(t, handler)
	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	member := seedMemberUser(t, pool, admin.OrganizationID, "member@example.com", "member-password-123")
	memberCookie := loginUser(t, handler, "member@example.com", "member-password-123")

	rec := postGraphQL(t, handler, `
		mutation {
			updateUserRole(input: { userId: "`+member.ID.String()+`", role: ADMIN }) {
				role
			}
		}
	`, memberCookie)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Errors []struct {
			Extensions map[string]interface{} `json:"extensions"`
		} `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Errors)
	require.Equal(t, handlers.CodeForbidden, resp.Errors[0].Extensions["code"])
}
