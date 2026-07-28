package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/allure-framework/allure-go/testify/require"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	escalitescim "github.com/mdg-labs/escalite/services/api/internal/scim"
)

func TestGraphQLRotateScimToken(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := newTestHandler(t)
		defer cleanup()

		cookie := bootstrapAdmin(t, handler)

		rec := postGraphQL(t, handler, `mutation {
		rotateScimToken {
			token
			scimSettings {
				configured
				tokenHint
				scimBaseUrl
			}
		}
	}`, cookie)
		require.Equal(a, http.StatusOK, rec.Code, rec.Body.String())

		var resp struct {
			Data struct {
				RotateScimToken struct {
					Token        string `json:"token"`
					ScimSettings struct {
						Configured  bool   `json:"configured"`
						TokenHint   string `json:"tokenHint"`
						ScimBaseURL string `json:"scimBaseUrl"`
					} `json:"scimSettings"`
				} `json:"rotateScimToken"`
			} `json:"data"`
			Errors []any `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Empty(a, resp.Errors)
		require.NotEmpty(a, resp.Data.RotateScimToken.Token)
		require.True(a, resp.Data.RotateScimToken.ScimSettings.Configured)
		require.NotEmpty(a, resp.Data.RotateScimToken.ScimSettings.TokenHint)
		require.Contains(a, resp.Data.RotateScimToken.ScimSettings.ScimBaseURL, "/scim/v2")

		settingsRec := postGraphQL(t, handler, `{ scimSettings { configured tokenHint scimBaseUrl } }`, cookie)
		require.Equal(a, http.StatusOK, settingsRec.Code)
		require.NotContains(a, settingsRec.Body.String(), resp.Data.RotateScimToken.Token)
	})
}

func TestScimDeprovisionBlocksAuthenticationWithin60Seconds(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		cookie := bootstrapAdmin(t, handler)
		token := rotateScimToken(t, handler, cookie)

		userID := createScimUser(t, handler, token, "scim-user@example.com", "ext-1")
		sessionCookie := createSessionForUser(t, pool, userID)

		meReq := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		meReq.AddCookie(sessionCookie)
		meRec := httptest.NewRecorder()
		handler.ServeHTTP(meRec, meReq)
		require.Equal(a, http.StatusOK, meRec.Code)

		deleteReq := scimRequest(t, http.MethodDelete, "/scim/v2/Users/"+userID.String(), token, nil)
		deleteRec := httptest.NewRecorder()
		handler.ServeHTTP(deleteRec, deleteReq)
		require.Equal(a, http.StatusNoContent, deleteRec.Code)

		start := time.Now()
		meReq = httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		meReq.AddCookie(sessionCookie)
		meRec = httptest.NewRecorder()
		handler.ServeHTTP(meRec, meReq)
		require.Equal(a, http.StatusUnauthorized, meRec.Code)
		require.Less(a, time.Since(start), 60*time.Second)

		loginBody := map[string]string{
			"email":    "scim-user@example.com",
			"password": "correct-horse-battery-staple",
		}
		payload, err := json.Marshal(loginBody)
		require.NoError(a, err)
		loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(payload))
		loginReq.Header.Set("Content-Type", "application/json")
		loginRec := httptest.NewRecorder()
		handler.ServeHTTP(loginRec, loginReq)
		require.Equal(a, http.StatusUnauthorized, loginRec.Code)
	})
}

func TestScimGroupMapsMembersToTeam(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		cookie := bootstrapAdmin(t, handler)
		token := rotateScimToken(t, handler, cookie)

		userID := createScimUser(t, handler, token, "group-user@example.com", "ext-group-user")

		groupBody := map[string]any{
			"schemas":     []string{escalitescim.SchemaGroup},
			"externalId":  "eng-oncall",
			"displayName": "Engineering On-Call",
			"members": []map[string]string{
				{"value": userID.String()},
			},
		}
		groupPayload, err := json.Marshal(groupBody)
		require.NoError(a, err)

		createGroupReq := scimRequest(t, http.MethodPost, "/scim/v2/Groups", token, groupPayload)
		createGroupRec := httptest.NewRecorder()
		handler.ServeHTTP(createGroupRec, createGroupReq)
		require.Equal(a, http.StatusCreated, createGroupRec.Code)

		queries := db.New(pool)
		org, err := queries.GetFirstOrganization(context.Background())
		require.NoError(a, err)

		team, err := queries.GetTeamByName(context.Background(), db.GetTeamByNameParams{
			OrganizationID: org.ID,
			Name:           "Engineering On-Call",
		})
		require.NoError(a, err)

		hasMembership, err := queries.HasTeamMembership(context.Background(), db.HasTeamMembershipParams{
			TeamID:         team.ID,
			UserID:         uuid.MustParse(userID.String()),
			OrganizationID: org.ID,
		})
		require.NoError(a, err)
		require.True(a, hasMembership)
	})
}

func rotateScimToken(t *testing.T, handler http.Handler, cookie *http.Cookie) string {
	t.Helper()

	rec := postGraphQL(t, handler, `mutation { rotateScimToken { token } }`, cookie)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var resp struct {
		Data struct {
			RotateScimToken struct {
				Token string `json:"token"`
			} `json:"rotateScimToken"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Data.RotateScimToken.Token)
	return resp.Data.RotateScimToken.Token
}

func createScimUser(t *testing.T, handler http.Handler, token, email, externalID string) uuid.UUID {
	t.Helper()

	body := map[string]any{
		"schemas":    []string{escalitescim.SchemaUser},
		"userName":   email,
		"externalId": externalID,
		"active":     true,
		"emails": []map[string]any{
			{"value": email, "primary": true},
		},
	}
	payload, err := json.Marshal(body)
	require.NoError(t, err)

	req := scimRequest(t, http.MethodPost, "/scim/v2/Users", token, payload)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	var user escalitescim.User
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &user))
	return uuid.MustParse(user.ID)
}

func createSessionForUser(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID) *http.Cookie {
	t.Helper()

	queries := db.New(pool)
	org, err := queries.GetFirstOrganization(context.Background())
	require.NoError(t, err)

	user, err := queries.GetUserByID(context.Background(), db.GetUserByIDParams{
		ID:             userID,
		OrganizationID: org.ID,
	})
	require.NoError(t, err)

	sessionID := uuid.Must(uuid.NewV7())
	expiresAt := time.Now().UTC().Add(auth.DefaultSessionTTL)
	_, err = queries.CreateSession(context.Background(), db.CreateSessionParams{
		ID:             sessionID,
		UserID:         user.ID,
		OrganizationID: user.OrganizationID,
		ExpiresAt:      pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	require.NoError(t, err)

	return &http.Cookie{
		Name:  auth.SessionCookieName,
		Value: sessionID.String(),
	}
}

func scimRequest(t *testing.T, method, path, token string, body []byte) *http.Request {
	t.Helper()

	var reader *bytes.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	} else {
		reader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", escalitescim.ContentType)
	}
	return req
}
