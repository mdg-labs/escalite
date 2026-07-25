package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func TestMobileAuthDeepLinkFlow(t *testing.T) {
	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	sessionCookie := bootstrapAdmin(t, handler)

	codeReq := httptest.NewRequest(http.MethodPost, "/api/v1/mobile/auth/code", nil)
	codeReq.AddCookie(sessionCookie)
	codeRec := httptest.NewRecorder()
	handler.ServeHTTP(codeRec, codeReq)
	require.Equal(t, http.StatusOK, codeRec.Code)

	var codeResp struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(codeRec.Body.Bytes(), &codeResp))
	require.NotEmpty(t, codeResp.Code)

	exchangePayload, err := json.Marshal(map[string]string{"code": codeResp.Code})
	require.NoError(t, err)

	exchangeReq := httptest.NewRequest(http.MethodPost, "/api/v1/mobile/auth/exchange", bytes.NewReader(exchangePayload))
	exchangeReq.Header.Set("Content-Type", "application/json")
	exchangeRec := httptest.NewRecorder()
	handler.ServeHTTP(exchangeRec, exchangeReq)
	require.Equal(t, http.StatusOK, exchangeRec.Code)

	var exchangeResp struct {
		RefreshToken string `json:"refresh_token"`
		User         struct {
			Email string `json:"email"`
			Role  string `json:"role"`
		} `json:"user"`
	}
	require.NoError(t, json.Unmarshal(exchangeRec.Body.Bytes(), &exchangeResp))
	require.NotEmpty(t, exchangeResp.RefreshToken)
	require.Equal(t, "admin@example.com", exchangeResp.User.Email)
	require.Equal(t, "admin", exchangeResp.User.Role)

	refreshPayload, err := json.Marshal(map[string]string{"refresh_token": exchangeResp.RefreshToken})
	require.NoError(t, err)

	refreshReq := httptest.NewRequest(http.MethodPost, "/api/v1/mobile/auth/refresh", bytes.NewReader(refreshPayload))
	refreshReq.Header.Set("Content-Type", "application/json")
	refreshRec := httptest.NewRecorder()
	handler.ServeHTTP(refreshRec, refreshReq)
	require.Equal(t, http.StatusOK, refreshRec.Code)

	var refreshResp struct {
		User struct {
			Email string `json:"email"`
		} `json:"user"`
	}
	require.NoError(t, json.Unmarshal(refreshRec.Body.Bytes(), &refreshResp))
	require.Equal(t, "admin@example.com", refreshResp.User.Email)

	queries := db.New(pool)
	stored, err := queries.GetRefreshTokenByHash(t.Context(), auth.HashRefreshToken(exchangeResp.RefreshToken))
	require.NoError(t, err)

	err = queries.RevokeRefreshToken(t.Context(), db.RevokeRefreshTokenParams{
		ID:             stored.ID,
		OrganizationID: stored.OrganizationID,
	})
	require.NoError(t, err)

	revokedRefreshReq := httptest.NewRequest(http.MethodPost, "/api/v1/mobile/auth/refresh", bytes.NewReader(refreshPayload))
	revokedRefreshReq.Header.Set("Content-Type", "application/json")
	revokedRefreshRec := httptest.NewRecorder()
	handler.ServeHTTP(revokedRefreshRec, revokedRefreshReq)
	require.Equal(t, http.StatusUnauthorized, revokedRefreshRec.Code)

	var apiErr struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(revokedRefreshRec.Body.Bytes(), &apiErr))
	require.Equal(t, handlers.CodeUnauthenticated, apiErr.Code)
}

func TestMobileAuthCodeRequiresSession(t *testing.T) {
	handler, _, cleanup := newTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/mobile/auth/code", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}
