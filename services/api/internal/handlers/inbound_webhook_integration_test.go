package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
	"github.com/mdg-labs/escalite/services/api/internal/queue"
	"github.com/mdg-labs/escalite/services/api/internal/ratelimit"
	"github.com/mdg-labs/escalite/services/api/internal/server"
	_ "github.com/mdg-labs/escalite/services/integrations/testplugin"
)

func inboundWebhookTestHandler(t *testing.T, keyLimit int) (http.Handler, *pgxpool.Pool, func()) {
	t.Helper()

	opts := testServerOptions{}
	if keyLimit > 0 {
		opts.InboundWebhook = &server.InboundWebhookOptions{
			KeyLimiter: ratelimit.NewMemoryLimiter(keyLimit, time.Minute),
		}
	}

	return newTestHandlerWithOptions(t, opts)
}

func postInboundWebhook(t *testing.T, handler http.Handler, plugin, token string, body []byte) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/webhook/"+plugin+"/"+token, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func seedIntegrationKey(t *testing.T, pool *pgxpool.Pool, orgID, serviceID uuid.UUID, pluginName string) (keyID uuid.UUID, plaintextToken string) {
	return seedIntegrationKeyWithConfig(t, pool, orgID, serviceID, pluginName, "{}")
}

func seedIntegrationKeyWithConfig(t *testing.T, pool *pgxpool.Pool, orgID, serviceID uuid.UUID, pluginName, config string) (keyID uuid.UUID, plaintextToken string) {
	t.Helper()

	plaintext, hash, prefix, err := auth.NewIntegrationKeyToken()
	require.NoError(t, err)

	keyID = uuid.Must(uuid.NewV7())
	_, err = pool.Exec(context.Background(), `
		INSERT INTO integration_keys (
			id, service_id, organization_id, token, prefix, plugin_name, config
		) VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
	`, keyID, serviceID, orgID, hash, prefix, pluginName, config)
	require.NoError(t, err)

	return keyID, plaintext
}

func TestInboundWebhookInvalidTokenReturnsNotFound(t *testing.T) {
	handler, _, cleanup := inboundWebhookTestHandler(t, 0)
	defer cleanup()

	rec := postInboundWebhook(t, handler, "test-plugin", "invalid-token-value", []byte(`{}`))
	require.Equal(t, http.StatusNotFound, rec.Code)

	var errResp struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	require.Equal(t, handlers.CodeNotFound, errResp.Code)
}

func TestInboundWebhookValidTokenAccepted(t *testing.T) {
	handler, pool, cleanup := inboundWebhookTestHandler(t, 0)
	defer cleanup()

	bootstrapAdmin(t, handler)

	admin, err := db.New(pool).GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	team := seedTeam(t, pool, admin.OrganizationID, "Platform")
	service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")

	_, token := seedIntegrationKey(t, pool, admin.OrganizationID, service.ID, "test-plugin")

	rec := postInboundWebhook(t, handler, "test-plugin", token, []byte(`{"alert":"firing"}`))
	require.Equal(t, http.StatusAccepted, rec.Code, rec.Body.String())

	var resp struct {
		Status string `json:"status"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "accepted", resp.Status)
}

func TestInboundWebhookRateLimited(t *testing.T) {
	handler, pool, cleanup := inboundWebhookTestHandler(t, 2)
	defer cleanup()

	bootstrapAdmin(t, handler)

	admin, err := db.New(pool).GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	team := seedTeam(t, pool, admin.OrganizationID, "Platform")
	service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")

	_, token := seedIntegrationKey(t, pool, admin.OrganizationID, service.ID, "test-plugin")
	body := []byte(`{"alert":"firing"}`)

	require.Equal(t, http.StatusAccepted, postInboundWebhook(t, handler, "test-plugin", token, body).Code)
	require.Equal(t, http.StatusAccepted, postInboundWebhook(t, handler, "test-plugin", token, body).Code)

	limited := postInboundWebhook(t, handler, "test-plugin", token, body)
	require.Equal(t, http.StatusTooManyRequests, limited.Code)

	var errResp struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(limited.Body.Bytes(), &errResp))
	require.Equal(t, handlers.CodeRateLimited, errResp.Code)
}

func TestInboundWebhookPayloadTooLarge(t *testing.T) {
	handler, pool, cleanup := inboundWebhookTestHandler(t, 0)
	defer cleanup()

	bootstrapAdmin(t, handler)

	admin, err := db.New(pool).GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	team := seedTeam(t, pool, admin.OrganizationID, "Platform")
	service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")

	_, token := seedIntegrationKey(t, pool, admin.OrganizationID, service.ID, "test-plugin")

	oversized := []byte(`"` + strings.Repeat("a", 256*1024) + `"`)
	rec := postInboundWebhook(t, handler, "test-plugin", token, oversized)
	require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)

	var errResp struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	require.Equal(t, handlers.CodePayloadTooLarge, errResp.Code)
}

func TestInboundWebhookRequiresJSONContentType(t *testing.T) {
	handler, pool, cleanup := inboundWebhookTestHandler(t, 0)
	defer cleanup()

	bootstrapAdmin(t, handler)

	admin, err := db.New(pool).GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	team := seedTeam(t, pool, admin.OrganizationID, "Platform")
	service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")

	_, token := seedIntegrationKey(t, pool, admin.OrganizationID, service.ID, "test-plugin")

	req := httptest.NewRequest(http.MethodPost, "/webhook/test-plugin/"+token, bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	require.Equal(t, handlers.CodeValidation, errResp.Code)
}

func TestInboundWebhookLogsTokenPrefixOnly(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler, pool, cleanup := inboundWebhookTestHandler(t, 0)
	defer cleanup()

	bootstrapAdmin(t, handler)

	admin, err := db.New(pool).GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	team := seedTeam(t, pool, admin.OrganizationID, "Platform")
	service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")

	_, token := seedIntegrationKey(t, pool, admin.OrganizationID, service.ID, "test-plugin")
	prefix := auth.TokenPrefix(token)

	jobs, err := queue.NewProducer(context.Background(), pool, logger)
	require.NoError(t, err)

	webhookHandler := handlers.NewInboundWebhookHandler(pool, jobs, logger, handlers.InboundWebhookConfig{})
	req := httptest.NewRequest(http.MethodPost, "/webhook/test-plugin/"+token, bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("plugin", "test-plugin")
	routeCtx.URLParams.Add("token", token)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
	rec := httptest.NewRecorder()
	webhookHandler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusAccepted, rec.Code)

	logOutput := buf.String()
	require.Contains(t, logOutput, `"token_prefix":"`+prefix+`"`)
	require.NotContains(t, logOutput, token)
}
