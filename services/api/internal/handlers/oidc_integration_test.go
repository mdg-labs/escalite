package handlers_test

import (
	"context"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/config"
	"github.com/mdg-labs/escalite/services/api/internal/oidc"
	"github.com/mdg-labs/escalite/services/api/internal/oidc/testutil"
	"github.com/mdg-labs/escalite/services/api/internal/server"
)

func TestOIDCRoutesDisabledWithoutConfig(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := newTestHandler(t)
		defer cleanup()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/login", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		require.Equal(a, http.StatusNotFound, rec.Code)
	})
}

func TestOIDCLoginCreatesSessionForExistingUser(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandlerWithOIDC(t, "admin@example.com")
		defer cleanup()

		_ = bootstrapAdmin(t, handler)

		loginReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/login", nil)
		loginRec := httptest.NewRecorder()
		handler.ServeHTTP(loginRec, loginReq)
		require.Equal(a, http.StatusFound, loginRec.Code)

		stateCookie := findOIDCStateCookie(t, loginRec)
		require.NotEmpty(a, stateCookie.Value)

		callbackReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/callback?code=test-auth-code&state="+stateCookie.Value, nil)
		callbackReq.AddCookie(stateCookie)
		callbackRec := httptest.NewRecorder()
		handler.ServeHTTP(callbackRec, callbackReq)

		require.Equal(a, http.StatusFound, callbackRec.Code)
		require.Equal(a, "http://localhost:3000", callbackRec.Header().Get("Location"))

		sessionCookie := findSessionCookie(t, callbackRec)
		require.NotEmpty(a, sessionCookie.Value)

		meReq := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		meReq.AddCookie(sessionCookie)
		meRec := httptest.NewRecorder()
		handler.ServeHTTP(meRec, meReq)
		require.Equal(a, http.StatusOK, meRec.Code)

		_ = pool
	})
}

func TestOIDCLoginCreatesMemberForNewEmail(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := newTestHandlerWithOIDC(t, "oidc-user@example.com")
		defer cleanup()

		_ = bootstrapAdmin(t, handler)

		loginReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/login", nil)
		loginRec := httptest.NewRecorder()
		handler.ServeHTTP(loginRec, loginReq)
		require.Equal(a, http.StatusFound, loginRec.Code)

		stateCookie := findOIDCStateCookie(t, loginRec)
		require.NotEmpty(a, stateCookie.Value)

		callbackReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/callback?code=test-auth-code&state="+stateCookie.Value, nil)
		callbackReq.AddCookie(stateCookie)
		callbackRec := httptest.NewRecorder()
		handler.ServeHTTP(callbackRec, callbackReq)
		require.Equal(a, http.StatusFound, callbackRec.Code)

		sessionCookie := findSessionCookie(t, callbackRec)

		meReq := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		meReq.AddCookie(sessionCookie)
		meRec := httptest.NewRecorder()
		handler.ServeHTTP(meRec, meReq)
		require.Equal(a, http.StatusOK, meRec.Code)
	})
}

func findOIDCStateCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()

	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == "escalite_oidc_state" {
			return cookie
		}
	}
	t.Fatal("oidc state cookie not found")
	return nil
}

func newTestHandlerWithOIDC(t *testing.T, email string) (http.Handler, *pgxpool.Pool, func()) {
	t.Helper()

	_, pool, cleanup := newTestHandler(t)

	const (
		clientID     = "test-client"
		clientSecret = "test-secret"
		redirectURL  = "http://example.com/api/v1/auth/oidc/callback"
	)

	mock := testutil.NewMockServer(t, clientID, clientSecret, redirectURL, email)
	t.Cleanup(mock.Close)

	provider, err := oidc.NewProvider(context.Background(), oidc.Config{
		IssuerURL:    mock.URL,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
	})
	require.NoError(t, err)

	oidcCfg := &config.OIDCConfig{
		IssuerURL:    mock.URL,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		SuccessURL:   "http://localhost:3000",
	}

	return server.New(server.Dependencies{
		Logger: slog.Default(),
		Pool:   pool,
		OIDC:   server.NewOIDCServices(pool, slog.Default(), oidcCfg, provider),
	}), pool, cleanup
}
