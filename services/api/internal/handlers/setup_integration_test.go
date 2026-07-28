package handlers_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/crypto"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/email"
	"github.com/mdg-labs/escalite/services/api/internal/graphql"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
	"github.com/mdg-labs/escalite/services/api/internal/migrate"
	"github.com/mdg-labs/escalite/services/api/internal/queue"
	"github.com/mdg-labs/escalite/services/api/internal/realtime"
	"github.com/mdg-labs/escalite/services/api/internal/server"
	"github.com/mdg-labs/escalite/services/api/internal/testutil"
)

type testServerOptions struct {
	Mail             email.Sender
	PublicURL        string
	PasswordReset    *server.PasswordResetOptions
	HeartbeatPing    *server.HeartbeatPingOptions
	InboundWebhook   *server.InboundWebhookOptions
	InboundEmail     *server.InboundEmailOptions
	SlackInteractive *server.SlackInteractiveOptions
	Secrets          *crypto.Box
}

const testEncryptionKeyHex = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func testSecretsBox(t *testing.T) *crypto.Box {
	t.Helper()
	key, err := hex.DecodeString(testEncryptionKeyHex)
	require.NoError(t, err)
	box, err := crypto.NewBox(key)
	require.NoError(t, err)
	return box
}

func mustDecodeTestEncryptionKey(t *testing.T) []byte {
	t.Helper()
	key, err := hex.DecodeString(testEncryptionKeyHex)
	require.NoError(t, err)
	return key
}

func newTestHandler(t *testing.T) (http.Handler, *pgxpool.Pool, func()) {
	return newTestHandlerWithOptions(t, testServerOptions{})
}

func newTestHandlerWithOptions(t *testing.T, opts testServerOptions) (http.Handler, *pgxpool.Pool, func()) {
	t.Helper()

	databaseURL, cleanup := testutil.StartPostgres(t)

	ctx := context.Background()
	require.NoError(t, migrate.Up(ctx, databaseURL, slog.Default()))

	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)

	jobs, err := queue.NewProducer(ctx, pool, slog.Default())
	require.NoError(t, err)

	secrets := opts.Secrets
	if secrets == nil {
		secrets = testSecretsBox(t)
	}

	realtimeBridge := realtime.NewBridge(databaseURL, slog.Default())
	realtimeBridge.Start(ctx)

	publicURL := opts.PublicURL
	if publicURL == "" {
		publicURL = "http://example.com"
	}

	handler := server.New(server.Dependencies{
		Logger:           slog.Default(),
		Pool:             pool,
		Jobs:             jobs,
		Secrets:          secrets,
		EncryptionKey:    mustDecodeTestEncryptionKey(t),
		Mail:             opts.Mail,
		PublicURL:        publicURL,
		PasswordReset:    opts.PasswordReset,
		HeartbeatPing:    opts.HeartbeatPing,
		InboundWebhook:   opts.InboundWebhook,
		InboundEmail:     opts.InboundEmail,
		SlackInteractive: opts.SlackInteractive,
		SAML:             server.NewSAMLServices(pool, slog.Default(), secrets, publicURL, "http://localhost:3000"),
		GraphQL: graphql.Options{
			PublicURL: publicURL,
		},
		Realtime: realtimeBridge.Hub,
	})

	return handler, pool, func() {
		realtimeBridge.Close()
		pool.Close()
		cleanup()
	}
}

func TestSetupIntegration(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		body := map[string]string{
			"organizationName": "Acme On-Call",
			"email":            "admin@example.com",
			"password":         "correct-horse-battery-staple",
		}
		payload, err := json.Marshal(body)
		require.NoError(a, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/setup", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "setup-integration-test")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		require.Equal(a, http.StatusCreated, rec.Code)

		var resp struct {
			Organization struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"organization"`
			User struct {
				ID    string `json:"id"`
				Email string `json:"email"`
				Role  string `json:"role"`
			} `json:"user"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Equal(a, "Acme On-Call", resp.Organization.Name)
		require.NotEmpty(a, resp.Organization.ID)
		require.Equal(a, "admin@example.com", resp.User.Email)
		require.Equal(a, "admin", resp.User.Role)
		require.NotEmpty(a, resp.User.ID)

		cookies := rec.Result().Cookies()
		var sessionCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == auth.SessionCookieName {
				sessionCookie = cookie
				break
			}
		}
		require.NotNil(a, sessionCookie)
		require.NotEmpty(a, sessionCookie.Value)
		require.True(a, sessionCookie.HttpOnly)
		require.Equal(a, http.SameSiteLaxMode, sessionCookie.SameSite)

		orgID := uuid.MustParse(resp.Organization.ID)
		sessionID := uuid.MustParse(sessionCookie.Value)

		ctx := context.Background()
		queries := db.New(pool)
		session, err := queries.GetSessionByID(ctx, db.GetSessionByIDParams{
			ID:             sessionID,
			OrganizationID: orgID,
		})
		require.NoError(a, err)
		require.Equal(a, uuid.MustParse(resp.User.ID), session.UserID)

		user, err := queries.GetUserByEmail(ctx, db.GetUserByEmailParams{
			OrganizationID: orgID,
			Email:          "admin@example.com",
		})
		require.NoError(a, err)
		require.Equal(a, "admin", user.Role)

		account, err := queries.GetAccountByID(ctx, user.AccountID)
		require.NoError(a, err)
		require.True(a, account.PasswordHash.Valid)
		require.NotEmpty(a, account.PasswordHash.String)

		match, err := auth.VerifyPassword("correct-horse-battery-staple", account.PasswordHash.String)
		require.NoError(a, err)
		require.True(a, match)

		repeatReq := httptest.NewRequest(http.MethodPost, "/api/v1/setup", bytes.NewReader(payload))
		repeatReq.Header.Set("Content-Type", "application/json")
		repeatRec := httptest.NewRecorder()
		handler.ServeHTTP(repeatRec, repeatReq)

		require.Equal(a, http.StatusForbidden, repeatRec.Code)

		var errResp struct {
			Error string `json:"error"`
			Code  string `json:"code"`
		}
		require.NoError(a, json.Unmarshal(repeatRec.Body.Bytes(), &errResp))
		require.Equal(a, "FORBIDDEN", errResp.Code)
	})
}

func TestSetupHandlerValidation(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		_, pool, cleanup := newTestHandler(t)
		defer cleanup()

		setup := handlers.NewSetupHandler(pool, slog.Default())

		req := httptest.NewRequest(http.MethodPost, "/api/v1/setup", bytes.NewReader([]byte(`{"organizationName":"","email":"bad","password":"short"}`)))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		setup.ServeHTTP(rec, req)

		require.Equal(a, http.StatusBadRequest, rec.Code)
		body, err := io.ReadAll(rec.Body)
		require.NoError(a, err)
		require.Contains(a, string(body), "VALIDATION")
	})
}
