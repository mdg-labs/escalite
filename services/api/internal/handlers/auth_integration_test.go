package handlers_test

import (
	"bytes"
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func bootstrapAdmin(t *testing.T, handler http.Handler) *http.Cookie {
	t.Helper()

	body := map[string]string{
		"organizationName": "Acme On-Call",
		"email":            "admin@example.com",
		"password":         "correct-horse-battery-staple",
	}
	payload, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	return findSessionCookie(t, rec)
}

func findSessionCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()

	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == auth.SessionCookieName {
			return cookie
		}
	}
	t.Fatal("session cookie not found")
	return nil
}

func TestLoginIntegration(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		_ = bootstrapAdmin(t, handler)

		loginBody := map[string]string{
			"email":    "admin@example.com",
			"password": "correct-horse-battery-staple",
		}
		payload, err := json.Marshal(loginBody)
		require.NoError(a, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		require.Equal(a, http.StatusOK, rec.Code)

		var resp struct {
			User struct {
				ID    string `json:"id"`
				Email string `json:"email"`
				Role  string `json:"role"`
			} `json:"user"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Equal(a, "admin@example.com", resp.User.Email)
		require.Equal(a, "admin", resp.User.Role)
		require.NotEmpty(a, resp.User.ID)

		sessionCookie := findSessionCookie(t, rec)
		require.True(a, sessionCookie.HttpOnly)
		require.Equal(a, http.SameSiteLaxMode, sessionCookie.SameSite)
		require.True(a, sessionCookie.Secure)

		meReq := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		meReq.AddCookie(sessionCookie)
		meRec := httptest.NewRecorder()
		handler.ServeHTTP(meRec, meReq)
		require.Equal(a, http.StatusOK, meRec.Code)

		_ = pool
	})
}

func TestLoginInvalidCredentialsNoEnumeration(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := newTestHandler(t)
		defer cleanup()

		_ = bootstrapAdmin(t, handler)

		cases := []struct {
			name     string
			email    string
			password string
		}{
			{name: "unknown email", email: "nobody@example.com", password: "correct-horse-battery-staple"},
			{name: "wrong password", email: "admin@example.com", password: "wrong-password"},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				payload, err := json.Marshal(map[string]string{
					"email":    tc.email,
					"password": tc.password,
				})
				require.NoError(a, err)

				req := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(payload))
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)

				require.Equal(a, http.StatusUnauthorized, rec.Code)

				var errResp struct {
					Error string `json:"error"`
					Code  string `json:"code"`
				}
				require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &errResp))
				require.Equal(a, handlers.CodeUnauthenticated, errResp.Code)
				require.Equal(a, "invalid credentials", errResp.Error)
			})
		}
	})
}

func TestRevokedSessionReturnsUnauthenticated(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := newTestHandler(t)
		defer cleanup()

		sessionCookie := bootstrapAdmin(t, handler)

		logoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/logout", nil)
		logoutReq.AddCookie(sessionCookie)
		logoutRec := httptest.NewRecorder()
		handler.ServeHTTP(logoutRec, logoutReq)
		require.Equal(a, http.StatusNoContent, logoutRec.Code)

		meReq := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		meReq.AddCookie(sessionCookie)
		meRec := httptest.NewRecorder()
		handler.ServeHTTP(meRec, meReq)

		require.Equal(a, http.StatusUnauthorized, meRec.Code)

		var errResp struct {
			Error string `json:"error"`
			Code  string `json:"code"`
		}
		require.NoError(a, json.Unmarshal(meRec.Body.Bytes(), &errResp))
		require.Equal(a, handlers.CodeUnauthenticated, errResp.Code)
	})
}

func TestUnauthenticatedMeReturnsUnauthenticated(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := newTestHandler(t)
		defer cleanup()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		require.Equal(a, http.StatusUnauthorized, rec.Code)

		var errResp struct {
			Code string `json:"code"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &errResp))
		require.Equal(a, handlers.CodeUnauthenticated, errResp.Code)
	})
}
