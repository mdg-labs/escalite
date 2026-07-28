package handlers_test

import (
	"context"
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/allure-framework/allure-go/testify/require"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
	"github.com/mdg-labs/escalite/services/api/internal/ratelimit"
	"github.com/mdg-labs/escalite/services/api/internal/server"
)

func heartbeatPingTestHandler(t *testing.T, tokenLimit int) (http.Handler, *pgxpool.Pool, func()) {
	t.Helper()

	opts := testServerOptions{}
	if tokenLimit > 0 {
		opts.HeartbeatPing = &server.HeartbeatPingOptions{
			TokenLimiter: ratelimit.NewMemoryLimiter(tokenLimit, time.Minute),
		}
	}

	return newTestHandlerWithOptions(t, opts)
}

func pingHeartbeat(t *testing.T, handler http.Handler, method, token string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, "/heartbeat/"+token, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func createHeartbeatMonitorViaGraphQL(t *testing.T, handler http.Handler, serviceID string, adminCookie *http.Cookie) (monitorID, token string) {
	t.Helper()

	rec := postGraphQL(t, handler, `mutation {
		createHeartbeatMonitor(input: {
			serviceId: "`+serviceID+`"
			name: "Ping test"
			intervalSeconds: 3600
			graceSeconds: 300
		}) {
			id
			token
		}
	}`, adminCookie)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var resp struct {
		Data struct {
			CreateHeartbeatMonitor struct {
				ID    string `json:"id"`
				Token string `json:"token"`
			} `json:"createHeartbeatMonitor"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	return resp.Data.CreateHeartbeatMonitor.ID, resp.Data.CreateHeartbeatMonitor.Token
}

func TestHeartbeatPingValidTokenUpdatesMonitor(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := heartbeatPingTestHandler(t, 0)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")

		monitorID, token := createHeartbeatMonitorViaGraphQL(t, handler, service.ID.String(), adminCookie)

		getRec := pingHeartbeat(t, handler, http.MethodGet, token)
		require.Equal(a, http.StatusOK, getRec.Code, getRec.Body.String())

		var pingResp struct {
			Status string `json:"status"`
		}
		require.NoError(a, json.Unmarshal(getRec.Body.Bytes(), &pingResp))
		require.Equal(a, "healthy", pingResp.Status)

		monitor, err := queries.GetHeartbeatMonitorByID(context.Background(), db.GetHeartbeatMonitorByIDParams{
			ID:             uuid.MustParse(monitorID),
			OrganizationID: admin.OrganizationID,
		})
		require.NoError(a, err)
		require.Equal(a, "healthy", monitor.Status)
		require.True(a, monitor.LastPingAt.Valid)

		postRec := pingHeartbeat(t, handler, http.MethodPost, token)
		require.Equal(a, http.StatusOK, postRec.Code, postRec.Body.String())
	})
}

func TestHeartbeatPingInvalidTokenReturnsNotFound(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := heartbeatPingTestHandler(t, 0)
		defer cleanup()

		rec := pingHeartbeat(t, handler, http.MethodGet, "totally-invalid-token")
		require.Equal(a, http.StatusNotFound, rec.Code)

		var errResp struct {
			Code string `json:"code"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &errResp))
		require.Equal(a, handlers.CodeNotFound, errResp.Code)
	})
}

func TestHeartbeatPingRateLimited(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := heartbeatPingTestHandler(t, 2)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")

		_, token := createHeartbeatMonitorViaGraphQL(t, handler, service.ID.String(), adminCookie)

		require.Equal(a, http.StatusOK, pingHeartbeat(t, handler, http.MethodGet, token).Code)
		require.Equal(a, http.StatusOK, pingHeartbeat(t, handler, http.MethodGet, token).Code)

		limited := pingHeartbeat(t, handler, http.MethodGet, token)
		require.Equal(a, http.StatusTooManyRequests, limited.Code)

		var errResp struct {
			Code string `json:"code"`
		}
		require.NoError(a, json.Unmarshal(limited.Body.Bytes(), &errResp))
		require.Equal(a, handlers.CodeRateLimited, errResp.Code)
	})
}

func TestHeartbeatPingRestoresHealthyStatus(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := heartbeatPingTestHandler(t, 0)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		team := seedTeam(t, pool, admin.OrganizationID, "Platform")
		service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")

		monitorID, token := createHeartbeatMonitorViaGraphQL(t, handler, service.ID.String(), adminCookie)

		_, err = pool.Exec(context.Background(),
			`UPDATE heartbeat_monitors SET status = 'overdue', last_ping_at = now() - interval '2 hours' WHERE id = $1`,
			uuid.MustParse(monitorID),
		)
		require.NoError(a, err)

		rec := pingHeartbeat(t, handler, http.MethodPost, token)
		require.Equal(a, http.StatusOK, rec.Code)

		monitor, err := queries.GetHeartbeatMonitorByID(context.Background(), db.GetHeartbeatMonitorByIDParams{
			ID:             uuid.MustParse(monitorID),
			OrganizationID: admin.OrganizationID,
		})
		require.NoError(a, err)
		require.Equal(a, "healthy", monitor.Status)
		require.True(a, monitor.LastPingAt.Valid)
		require.True(a, monitor.LastPingAt.Time.After(time.Now().UTC().Add(-time.Minute)))
	})
}

func TestHeartbeatPingLogsTokenPrefixOnly(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		require.Equal(a, "abcdEFGH", auth.TokenPrefix("abcdEFGHijklmnop"))
		require.Equal(a, "short", auth.TokenPrefix("short"))
	})
}

func TestHeartbeatPingEmptyTokenReturnsNotFound(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := heartbeatPingTestHandler(t, 0)
		defer cleanup()

		req := httptest.NewRequest(http.MethodGet, "/heartbeat/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		require.NotEqual(a, http.StatusOK, rec.Code)
	})
}
