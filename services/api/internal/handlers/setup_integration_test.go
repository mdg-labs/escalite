package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/email"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
	"github.com/mdg-labs/escalite/services/api/internal/migrate"
	"github.com/mdg-labs/escalite/services/api/internal/queue"
	"github.com/mdg-labs/escalite/services/api/internal/server"
)

func startPostgres(t *testing.T) (string, func()) {
	t.Helper()

	ctx := context.Background()

	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("escalite"),
		postgres.WithUsername("escalite"),
		postgres.WithPassword("escalite"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)

	databaseURL, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	cleanup := func() {
		require.NoError(t, testcontainers.TerminateContainer(container))
	}

	return databaseURL, cleanup
}

type testServerOptions struct {
	Mail          email.Sender
	PublicURL     string
	PasswordReset *server.PasswordResetOptions
}

func newTestHandler(t *testing.T) (http.Handler, *pgxpool.Pool, func()) {
	return newTestHandlerWithOptions(t, testServerOptions{})
}

func newTestHandlerWithOptions(t *testing.T, opts testServerOptions) (http.Handler, *pgxpool.Pool, func()) {
	t.Helper()

	databaseURL, cleanup := startPostgres(t)

	ctx := context.Background()
	require.NoError(t, migrate.Up(ctx, databaseURL, slog.Default()))

	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)

	jobs, err := queue.NewProducer(ctx, pool, slog.Default())
	require.NoError(t, err)

	handler := server.New(server.Dependencies{
		Logger:        slog.Default(),
		Pool:          pool,
		Jobs:          jobs,
		Mail:          opts.Mail,
		PublicURL:     opts.PublicURL,
		PasswordReset: opts.PasswordReset,
	})

	return handler, pool, func() {
		pool.Close()
		cleanup()
	}
}

func TestSetupIntegration(t *testing.T) {
	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	body := map[string]string{
		"organizationName": "Acme On-Call",
		"email":            "admin@example.com",
		"password":         "correct-horse-battery-staple",
	}
	payload, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "setup-integration-test")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

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
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "Acme On-Call", resp.Organization.Name)
	require.NotEmpty(t, resp.Organization.ID)
	require.Equal(t, "admin@example.com", resp.User.Email)
	require.Equal(t, "admin", resp.User.Role)
	require.NotEmpty(t, resp.User.ID)

	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == auth.SessionCookieName {
			sessionCookie = cookie
			break
		}
	}
	require.NotNil(t, sessionCookie)
	require.NotEmpty(t, sessionCookie.Value)
	require.True(t, sessionCookie.HttpOnly)
	require.Equal(t, http.SameSiteLaxMode, sessionCookie.SameSite)

	orgID := uuid.MustParse(resp.Organization.ID)
	sessionID := uuid.MustParse(sessionCookie.Value)

	ctx := context.Background()
	queries := db.New(pool)
	session, err := queries.GetSessionByID(ctx, db.GetSessionByIDParams{
		ID:             sessionID,
		OrganizationID: orgID,
	})
	require.NoError(t, err)
	require.Equal(t, uuid.MustParse(resp.User.ID), session.UserID)

	user, err := queries.GetUserByEmail(ctx, db.GetUserByEmailParams{
		OrganizationID: orgID,
		Email:          "admin@example.com",
	})
	require.NoError(t, err)
	require.Equal(t, "admin", user.Role)
	require.True(t, user.PasswordHash.Valid)
	require.NotEmpty(t, user.PasswordHash.String)

	match, err := auth.VerifyPassword("correct-horse-battery-staple", user.PasswordHash.String)
	require.NoError(t, err)
	require.True(t, match)

	repeatReq := httptest.NewRequest(http.MethodPost, "/api/v1/setup", bytes.NewReader(payload))
	repeatReq.Header.Set("Content-Type", "application/json")
	repeatRec := httptest.NewRecorder()
	handler.ServeHTTP(repeatRec, repeatReq)

	require.Equal(t, http.StatusForbidden, repeatRec.Code)

	var errResp struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(repeatRec.Body.Bytes(), &errResp))
	require.Equal(t, "FORBIDDEN", errResp.Code)
}

func TestSetupHandlerValidation(t *testing.T) {
	_, pool, cleanup := newTestHandler(t)
	defer cleanup()

	setup := handlers.NewSetupHandler(pool, slog.Default())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup", bytes.NewReader([]byte(`{"organizationName":"","email":"bad","password":"short"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	setup.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	body, err := io.ReadAll(rec.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "VALIDATION")
}
