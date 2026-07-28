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
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func TestMobileAuthDeepLinkFlow(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		sessionCookie := bootstrapAdmin(t, handler)

		codeReq := httptest.NewRequest(http.MethodPost, "/api/v1/mobile/auth/code", nil)
		codeReq.AddCookie(sessionCookie)
		codeRec := httptest.NewRecorder()
		handler.ServeHTTP(codeRec, codeReq)
		require.Equal(a, http.StatusOK, codeRec.Code)

		var codeResp struct {
			Code string `json:"code"`
		}
		require.NoError(a, json.Unmarshal(codeRec.Body.Bytes(), &codeResp))
		require.NotEmpty(a, codeResp.Code)

		exchangePayload, err := json.Marshal(map[string]string{"code": codeResp.Code})
		require.NoError(a, err)

		exchangeReq := httptest.NewRequest(http.MethodPost, "/api/v1/mobile/auth/exchange", bytes.NewReader(exchangePayload))
		exchangeReq.Header.Set("Content-Type", "application/json")
		exchangeRec := httptest.NewRecorder()
		handler.ServeHTTP(exchangeRec, exchangeReq)
		require.Equal(a, http.StatusOK, exchangeRec.Code)

		var exchangeResp struct {
			RefreshToken string `json:"refresh_token"`
			User         struct {
				Email string `json:"email"`
				Role  string `json:"role"`
			} `json:"user"`
		}
		require.NoError(a, json.Unmarshal(exchangeRec.Body.Bytes(), &exchangeResp))
		require.NotEmpty(a, exchangeResp.RefreshToken)
		require.Equal(a, "admin@example.com", exchangeResp.User.Email)
		require.Equal(a, "admin", exchangeResp.User.Role)

		refreshPayload, err := json.Marshal(map[string]string{"refresh_token": exchangeResp.RefreshToken})
		require.NoError(a, err)

		refreshReq := httptest.NewRequest(http.MethodPost, "/api/v1/mobile/auth/refresh", bytes.NewReader(refreshPayload))
		refreshReq.Header.Set("Content-Type", "application/json")
		refreshRec := httptest.NewRecorder()
		handler.ServeHTTP(refreshRec, refreshReq)
		require.Equal(a, http.StatusOK, refreshRec.Code)

		var refreshResp struct {
			User struct {
				Email string `json:"email"`
			} `json:"user"`
		}
		require.NoError(a, json.Unmarshal(refreshRec.Body.Bytes(), &refreshResp))
		require.Equal(a, "admin@example.com", refreshResp.User.Email)

		queries := db.New(pool)
		stored, err := queries.GetRefreshTokenByHash(t.Context(), auth.HashRefreshToken(exchangeResp.RefreshToken))
		require.NoError(a, err)

		err = queries.RevokeRefreshToken(t.Context(), db.RevokeRefreshTokenParams{
			ID:             stored.ID,
			OrganizationID: stored.OrganizationID,
		})
		require.NoError(a, err)

		revokedRefreshReq := httptest.NewRequest(http.MethodPost, "/api/v1/mobile/auth/refresh", bytes.NewReader(refreshPayload))
		revokedRefreshReq.Header.Set("Content-Type", "application/json")
		revokedRefreshRec := httptest.NewRecorder()
		handler.ServeHTTP(revokedRefreshRec, revokedRefreshReq)
		require.Equal(a, http.StatusUnauthorized, revokedRefreshRec.Code)

		var apiErr struct {
			Code string `json:"code"`
		}
		require.NoError(a, json.Unmarshal(revokedRefreshRec.Body.Bytes(), &apiErr))
		require.Equal(a, handlers.CodeUnauthenticated, apiErr.Code)
	})
}

func TestMobileAuthCodeRequiresSession(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := newTestHandler(t)
		defer cleanup()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/mobile/auth/code", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equal(a, http.StatusUnauthorized, rec.Code)
	})
}
