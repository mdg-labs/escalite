package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"

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
	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)

	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)
	orgID := admin.OrganizationID

	teamA := seedTeam(t, pool, orgID, "Platform")
	teamB := seedTeam(t, pool, orgID, "Payments")

	recA := getTeam(t, handler, adminCookie, teamA.ID)
	require.Equal(t, http.StatusOK, recA.Code, recA.Body.String())

	var teamResp struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	require.NoError(t, json.Unmarshal(recA.Body.Bytes(), &teamResp))
	require.Equal(t, teamA.ID.String(), teamResp.ID)
	require.Equal(t, "Platform", teamResp.Name)

	recB := getTeam(t, handler, adminCookie, teamB.ID)
	require.Equal(t, http.StatusOK, recB.Code, recB.Body.String())
}

func TestRBACMemberTeamAccess(t *testing.T) {
	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	_ = bootstrapAdmin(t, handler)

	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)
	orgID := admin.OrganizationID

	teamA := seedTeam(t, pool, orgID, "Platform")
	teamB := seedTeam(t, pool, orgID, "Payments")

	member := seedMemberUser(t, pool, orgID, "member@example.com", "member-password-123")
	seedTeamMembership(t, pool, orgID, teamA.ID, member.ID)

	memberCookie := loginUser(t, handler, "member@example.com", "member-password-123")

	allowed := getTeam(t, handler, memberCookie, teamA.ID)
	require.Equal(t, http.StatusOK, allowed.Code, allowed.Body.String())

	denied := getTeam(t, handler, memberCookie, teamB.ID)
	require.Equal(t, http.StatusForbidden, denied.Code, denied.Body.String())

	var errResp struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(denied.Body.Bytes(), &errResp))
	require.Equal(t, handlers.CodeForbidden, errResp.Code)
	require.Equal(t, "access denied", errResp.Error)
}

func TestRBACMemberCannotAccessUnknownTeam(t *testing.T) {
	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	_ = bootstrapAdmin(t, handler)

	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	member := seedMemberUser(t, pool, admin.OrganizationID, "member@example.com", "member-password-123")
	memberCookie := loginUser(t, handler, member.Email, "member-password-123")

	unknownTeamID := uuid.Must(uuid.NewV7())
	rec := getTeam(t, handler, memberCookie, unknownTeamID)
	require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())

	var errResp struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	require.Equal(t, handlers.CodeNotFound, errResp.Code)
}
