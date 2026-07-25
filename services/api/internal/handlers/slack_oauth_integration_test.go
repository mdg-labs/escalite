package handlers_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/crypto"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
	"github.com/mdg-labs/escalite/services/api/internal/server"
)

// fakeSlackOAuthProvider is a test double for handlers.SlackOAuthProvider that avoids
// calling the real Slack API.
type fakeSlackOAuthProvider struct {
	authCodeURL string
	result      handlers.SlackInstallResult
	err         error
	codesSeen   []string
}

func (f *fakeSlackOAuthProvider) AuthCodeURL(state string) string {
	return f.authCodeURL + "?state=" + state
}

func (f *fakeSlackOAuthProvider) Exchange(_ context.Context, code string) (handlers.SlackInstallResult, error) {
	f.codesSeen = append(f.codesSeen, code)
	if f.err != nil {
		return handlers.SlackInstallResult{}, f.err
	}
	return f.result, nil
}

func newTestHandlerWithSlackOAuth(t *testing.T, provider handlers.SlackOAuthProvider) (http.Handler, *pgxpool.Pool, func()) {
	t.Helper()

	_, pool, cleanup := newTestHandler(t)

	secrets := testSecretsBox(t)
	slackHandler := handlers.NewSlackOAuthHandler(pool, slog.Default(), provider, secrets, "http://localhost:3000/settings")

	handler := server.New(server.Dependencies{
		Logger:  slog.Default(),
		Pool:    pool,
		Secrets: secrets,
		SlackOAuth: &server.SlackOAuthServices{
			Handler:  slackHandler,
			Install:  slackHandler.Install,
			Callback: slackHandler.Callback,
		},
	})

	return handler, pool, cleanup
}

func findSlackOAuthStateCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()

	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == "escalite_slack_oauth_state" {
			return cookie
		}
	}
	t.Fatal("slack oauth state cookie not found")
	return nil
}

func TestSlackOAuthRoutesDisabledWithoutConfig(t *testing.T) {
	handler, _, cleanup := newTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/slack/install", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestSlackOAuthInstallRequiresAuthentication(t *testing.T) {
	provider := &fakeSlackOAuthProvider{authCodeURL: "https://slack.com/oauth/v2/authorize"}
	handler, _, cleanup := newTestHandlerWithSlackOAuth(t, provider)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/slack/install", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestSlackOAuthInstallRequiresAdmin(t *testing.T) {
	provider := &fakeSlackOAuthProvider{authCodeURL: "https://slack.com/oauth/v2/authorize"}
	handler, pool, cleanup := newTestHandlerWithSlackOAuth(t, provider)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)

	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	_ = seedMemberUser(t, pool, admin.OrganizationID, "member@example.com", "member-password-123")
	memberCookie := loginUser(t, handler, "member@example.com", "member-password-123")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/slack/install", nil)
	req.AddCookie(memberCookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)

	adminReq := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/slack/install", nil)
	adminReq.AddCookie(adminCookie)
	adminRec := httptest.NewRecorder()
	handler.ServeHTTP(adminRec, adminReq)

	require.Equal(t, http.StatusFound, adminRec.Code)
	require.Contains(t, adminRec.Header().Get("Location"), provider.authCodeURL)
	require.NotNil(t, findSlackOAuthStateCookie(t, adminRec))
}

func TestSlackOAuthCallbackInvalidStateRejected(t *testing.T) {
	provider := &fakeSlackOAuthProvider{authCodeURL: "https://slack.com/oauth/v2/authorize"}
	handler, _, cleanup := newTestHandlerWithSlackOAuth(t, provider)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/slack/callback?code=test-code&state=wrong-state", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestSlackOAuthInstallStoresEncryptedTokenWithKeyID(t *testing.T) {
	provider := &fakeSlackOAuthProvider{
		authCodeURL: "https://slack.com/oauth/v2/authorize",
		result: handlers.SlackInstallResult{
			BotToken:      "xoxb-install-token-abcd",
			WorkspaceID:   "T00000001",
			WorkspaceName: "Acme Workspace",
			BotUserID:     "U00000001",
			Scope:         "chat:write,channels:read",
		},
	}
	handler, pool, cleanup := newTestHandlerWithSlackOAuth(t, provider)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)

	installReq := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/slack/install", nil)
	installReq.AddCookie(adminCookie)
	installRec := httptest.NewRecorder()
	handler.ServeHTTP(installRec, installReq)
	require.Equal(t, http.StatusFound, installRec.Code)

	stateCookie := findSlackOAuthStateCookie(t, installRec)
	require.NotEmpty(t, stateCookie.Value)

	callbackReq := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/slack/callback?code=test-auth-code&state="+stateCookie.Value, nil)
	callbackReq.AddCookie(adminCookie)
	callbackReq.AddCookie(stateCookie)
	callbackRec := httptest.NewRecorder()
	handler.ServeHTTP(callbackRec, callbackReq)

	require.Equal(t, http.StatusFound, callbackRec.Code)
	require.Equal(t, "http://localhost:3000/settings?slack=connected", callbackRec.Header().Get("Location"))
	require.Equal(t, []string{"test-auth-code"}, provider.codesSeen)

	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	settings, err := queries.GetOrganizationSlackSettings(context.Background(), admin.OrganizationID)
	require.NoError(t, err)
	require.Equal(t, "abcd", settings.TokenHint)
	require.Equal(t, crypto.DefaultKeyID, settings.EncryptionKeyID)
	require.True(t, settings.WorkspaceID.Valid)
	require.Equal(t, "T00000001", settings.WorkspaceID.String)
	require.True(t, settings.WorkspaceName.Valid)
	require.Equal(t, "Acme Workspace", settings.WorkspaceName.String)
	require.NotContains(t, string(settings.BotTokenCiphertext), "xoxb-install-token-abcd")

	box := testSecretsBox(t)
	plaintext, err := box.Decrypt(crypto.Encrypted{KeyID: settings.EncryptionKeyID, Ciphertext: settings.BotTokenCiphertext})
	require.NoError(t, err)
	require.Equal(t, "xoxb-install-token-abcd", string(plaintext))
}

func TestSlackOAuthReinstallUpdatesTokenWithoutDuplicateWorkspaceRow(t *testing.T) {
	provider := &fakeSlackOAuthProvider{
		authCodeURL: "https://slack.com/oauth/v2/authorize",
		result: handlers.SlackInstallResult{
			BotToken:      "xoxb-first-token-1111",
			WorkspaceID:   "T00000001",
			WorkspaceName: "Acme Workspace",
			BotUserID:     "U00000001",
			Scope:         "chat:write",
		},
	}
	handler, pool, cleanup := newTestHandlerWithSlackOAuth(t, provider)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)

	runInstall := func() {
		installReq := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/slack/install", nil)
		installReq.AddCookie(adminCookie)
		installRec := httptest.NewRecorder()
		handler.ServeHTTP(installRec, installReq)
		require.Equal(t, http.StatusFound, installRec.Code)

		stateCookie := findSlackOAuthStateCookie(t, installRec)

		callbackReq := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/slack/callback?code=code&state="+stateCookie.Value, nil)
		callbackReq.AddCookie(adminCookie)
		callbackReq.AddCookie(stateCookie)
		callbackRec := httptest.NewRecorder()
		handler.ServeHTTP(callbackRec, callbackReq)
		require.Equal(t, http.StatusFound, callbackRec.Code, callbackRec.Body.String())
	}

	runInstall()

	// Reinstall (e.g. re-authorizing the same Slack workspace) with a rotated token.
	provider.result = handlers.SlackInstallResult{
		BotToken:      "xoxb-second-token-2222",
		WorkspaceID:   "T00000001",
		WorkspaceName: "Acme Workspace Renamed",
		BotUserID:     "U00000001",
		Scope:         "chat:write,channels:read",
	}
	runInstall()

	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	var rowCount int
	require.NoError(t, pool.QueryRow(
		context.Background(),
		"SELECT count(*) FROM organization_slack_settings WHERE organization_id = $1",
		admin.OrganizationID,
	).Scan(&rowCount))
	require.Equal(t, 1, rowCount, "reinstall must update the existing row, not insert a duplicate")

	settings, err := queries.GetOrganizationSlackSettings(context.Background(), admin.OrganizationID)
	require.NoError(t, err)
	require.Equal(t, "2222", settings.TokenHint)
	require.Equal(t, "Acme Workspace Renamed", settings.WorkspaceName.String)

	box := testSecretsBox(t)
	plaintext, err := box.Decrypt(crypto.Encrypted{KeyID: settings.EncryptionKeyID, Ciphertext: settings.BotTokenCiphertext})
	require.NoError(t, err)
	require.Equal(t, "xoxb-second-token-2222", string(plaintext))
}

func TestSlackOAuthExchangeFailureRedirectsWithErrorStatus(t *testing.T) {
	provider := &fakeSlackOAuthProvider{
		authCodeURL: "https://slack.com/oauth/v2/authorize",
		err:         errors.New("invalid_code"),
	}
	handler, _, cleanup := newTestHandlerWithSlackOAuth(t, provider)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)

	installReq := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/slack/install", nil)
	installReq.AddCookie(adminCookie)
	installRec := httptest.NewRecorder()
	handler.ServeHTTP(installRec, installReq)
	require.Equal(t, http.StatusFound, installRec.Code)

	stateCookie := findSlackOAuthStateCookie(t, installRec)

	callbackReq := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/slack/callback?code=bad-code&state="+stateCookie.Value, nil)
	callbackReq.AddCookie(adminCookie)
	callbackReq.AddCookie(stateCookie)
	callbackRec := httptest.NewRecorder()
	handler.ServeHTTP(callbackRec, callbackReq)

	require.Equal(t, http.StatusFound, callbackRec.Code)
	require.Equal(t, "http://localhost:3000/settings?slack=error", callbackRec.Header().Get("Location"))
}

func TestSlackOAuthProviderAuthCodeURL(t *testing.T) {
	provider := handlers.NewSlackOAuthClient("client-id", "client-secret", "http://localhost/callback", []string{"chat:write", "channels:read"})
	require.Contains(t, provider.AuthCodeURL("state-123"), "state=state-123")
	require.Contains(t, provider.AuthCodeURL("state-123"), "client_id=client-id")
	require.Contains(t, provider.AuthCodeURL("state-123"), "scope=chat%3Awrite%2Cchannels%3Aread")
}
