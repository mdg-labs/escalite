package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/email"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
	"github.com/mdg-labs/escalite/services/api/internal/ratelimit"
	"github.com/mdg-labs/escalite/services/api/internal/server"
)

func passwordResetTestHandler(t *testing.T, emailLimit, ipLimit int) (http.Handler, *pgxpool.Pool, *email.RecordingSender, func()) {
	t.Helper()

	mail := &email.RecordingSender{}
	handler, pool, cleanup := newTestHandlerWithOptions(t, testServerOptions{
		Mail:      mail,
		PublicURL: "http://localhost:5173",
		PasswordReset: &server.PasswordResetOptions{
			EmailLimiter: ratelimit.NewMemoryLimiter(emailLimit, time.Hour),
			IPLimiter:    ratelimit.NewMemoryLimiter(ipLimit, time.Hour),
		},
	})

	return handler, pool, mail, cleanup
}

func postPasswordResetRequest(t *testing.T, handler http.Handler, emailAddr string) *httptest.ResponseRecorder {
	t.Helper()

	payload, err := json.Marshal(map[string]string{"email": emailAddr})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/password-reset/request", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.10:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func extractResetToken(t *testing.T, body string) string {
	t.Helper()

	const marker = "token="
	if idx := strings.Index(body, marker); idx >= 0 {
		token := strings.TrimSpace(body[idx+len(marker):])
		if end := strings.IndexAny(token, "\r\n "); end >= 0 {
			token = token[:end]
		}
		require.NotEmpty(t, token)
		return token
	}

	token := strings.TrimSpace(body)
	require.NotEmpty(t, token)
	return token
}

func TestPasswordResetRequestSendsEmailOnlyForExistingUser(t *testing.T) {
	handler, _, mail, cleanup := passwordResetTestHandler(t, 10, 10)
	defer cleanup()

	_ = bootstrapAdmin(t, handler)

	existing := postPasswordResetRequest(t, handler, "admin@example.com")
	require.Equal(t, http.StatusAccepted, existing.Code)
	require.Len(t, mail.Messages, 1)
	require.Equal(t, "admin@example.com", mail.Messages[0].To)

	unknown := postPasswordResetRequest(t, handler, "nobody@example.com")
	require.Equal(t, http.StatusAccepted, unknown.Code)
	require.Len(t, mail.Messages, 1)

	var existingResp struct {
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(existing.Body.Bytes(), &existingResp))

	var unknownResp struct {
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(unknown.Body.Bytes(), &unknownResp))
	require.Equal(t, existingResp.Message, unknownResp.Message)
}

func TestPasswordResetConfirmExpiredTokenRejected(t *testing.T) {
	handler, pool, _, cleanup := passwordResetTestHandler(t, 10, 10)
	defer cleanup()

	_ = bootstrapAdmin(t, handler)

	plaintext, tokenHash, err := auth.NewPasswordResetToken()
	require.NoError(t, err)

	ctx := context.Background()
	queries := db.New(pool)
	user, err := queries.GetUserByEmailForAuth(ctx, "admin@example.com")
	require.NoError(t, err)

	_, err = queries.CreatePasswordResetToken(ctx, db.CreatePasswordResetTokenParams{
		ID:             uuid.Must(uuid.NewV7()),
		UserID:         user.ID,
		OrganizationID: user.OrganizationID,
		TokenHash:      tokenHash,
		ExpiresAt:      pgtype.Timestamptz{Time: time.Now().UTC().Add(-time.Minute), Valid: true},
	})
	require.NoError(t, err)

	payload, err := json.Marshal(map[string]string{
		"token":    plaintext,
		"password": "new-password-12345",
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/password-reset/confirm", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	require.Equal(t, handlers.CodeValidation, errResp.Code)
}

func TestPasswordResetRequestRateLimited(t *testing.T) {
	handler, _, _, cleanup := passwordResetTestHandler(t, 2, 100)
	defer cleanup()

	_ = bootstrapAdmin(t, handler)

	require.Equal(t, http.StatusAccepted, postPasswordResetRequest(t, handler, "admin@example.com").Code)
	require.Equal(t, http.StatusAccepted, postPasswordResetRequest(t, handler, "admin@example.com").Code)

	limited := postPasswordResetRequest(t, handler, "admin@example.com")
	require.Equal(t, http.StatusTooManyRequests, limited.Code)

	var errResp struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(limited.Body.Bytes(), &errResp))
	require.Equal(t, handlers.CodeRateLimited, errResp.Code)
}

func TestPasswordResetConfirmUpdatesPassword(t *testing.T) {
	handler, pool, mail, cleanup := passwordResetTestHandler(t, 10, 10)
	defer cleanup()

	_ = bootstrapAdmin(t, handler)

	rec := postPasswordResetRequest(t, handler, "admin@example.com")
	require.Equal(t, http.StatusAccepted, rec.Code)
	require.Len(t, mail.Messages, 1)

	token := extractResetToken(t, mail.Messages[0].Body)
	newPassword := "brand-new-password-99"

	confirmPayload, err := json.Marshal(map[string]string{
		"token":    token,
		"password": newPassword,
	})
	require.NoError(t, err)

	confirmReq := httptest.NewRequest(http.MethodPost, "/api/v1/password-reset/confirm", bytes.NewReader(confirmPayload))
	confirmReq.Header.Set("Content-Type", "application/json")
	confirmRec := httptest.NewRecorder()
	handler.ServeHTTP(confirmRec, confirmReq)
	require.Equal(t, http.StatusNoContent, confirmRec.Code)

	ctx := context.Background()
	user, err := db.New(pool).GetUserByEmailForAuth(ctx, "admin@example.com")
	require.NoError(t, err)
	match, err := auth.VerifyPassword(newPassword, user.PasswordHash.String)
	require.NoError(t, err)
	require.True(t, match)

	oldLoginPayload, err := json.Marshal(map[string]string{
		"email":    "admin@example.com",
		"password": "correct-horse-battery-staple",
	})
	require.NoError(t, err)
	oldLoginReq := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(oldLoginPayload))
	oldLoginReq.Header.Set("Content-Type", "application/json")
	oldLoginRec := httptest.NewRecorder()
	handler.ServeHTTP(oldLoginRec, oldLoginReq)
	require.Equal(t, http.StatusUnauthorized, oldLoginRec.Code)

	newLoginPayload, err := json.Marshal(map[string]string{
		"email":    "admin@example.com",
		"password": newPassword,
	})
	require.NoError(t, err)
	newLoginReq := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(newLoginPayload))
	newLoginReq.Header.Set("Content-Type", "application/json")
	newLoginRec := httptest.NewRecorder()
	handler.ServeHTTP(newLoginRec, newLoginReq)
	require.Equal(t, http.StatusOK, newLoginRec.Code)
}
