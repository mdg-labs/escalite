package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"
	"github.com/google/uuid"

	"github.com/mdg-labs/escalite/services/api/internal/audit"
	"github.com/mdg-labs/escalite/services/api/internal/db"
)

func TestAuditLoginSuccess(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		sessionCookie := bootstrapAdmin(t, handler)

		ctx := context.Background()
		queries := db.New(pool)
		org, err := queries.GetFirstOrganization(ctx)
		require.NoError(a, err)

		events, err := queries.ListAuditEventsByOrganization(ctx, org.ID)
		require.NoError(a, err)
		require.Len(a, events, 1)
		require.Equal(a, audit.ActionLogin, events[0].Action)
		require.True(a, events[0].ActorID.Valid)

		loginBody := map[string]string{
			"email":    "admin@example.com",
			"password": "correct-horse-battery-staple",
		}
		payload, err := json.Marshal(loginBody)
		require.NoError(a, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "audit-login-test/1.0")
		req.RemoteAddr = "198.51.100.42:54321"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equal(a, http.StatusOK, rec.Code)

		events, err = queries.ListAuditEventsByOrganization(ctx, org.ID)
		require.NoError(a, err)
		require.Len(a, events, 2)
		require.Equal(a, audit.ActionLogin, events[1].Action)

		var meta map[string]string
		require.NoError(a, json.Unmarshal(events[1].Metadata, &meta))
		require.Equal(a, "198.51.100.42", meta["ip"])
		require.Equal(a, "audit-login-test/1.0", meta["user_agent"])

		_ = sessionCookie
	})
}

func TestAuditLoginFailedIncludesIPAndUserAgent(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		_ = bootstrapAdmin(t, handler)

		ctx := context.Background()
		queries := db.New(pool)
		org, err := queries.GetFirstOrganization(ctx)
		require.NoError(a, err)

		payload, err := json.Marshal(map[string]string{
			"email":    "admin@example.com",
			"password": "wrong-password",
		})
		require.NoError(a, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "audit-failed-login-test/2.0")
		req.Header.Set("X-Forwarded-For", "203.0.113.10, 198.51.100.1")
		req.RemoteAddr = "127.0.0.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equal(a, http.StatusUnauthorized, rec.Code)

		events, err := queries.ListAuditEventsByOrganization(ctx, org.ID)
		require.NoError(a, err)

		var failedEvent *db.AuditEvent
		for i := range events {
			if events[i].Action == audit.ActionLoginFailed {
				failedEvent = &events[i]
				break
			}
		}
		require.NotNil(a, failedEvent)

		var meta map[string]string
		require.NoError(a, json.Unmarshal(failedEvent.Metadata, &meta))
		require.Equal(a, "203.0.113.10", meta["ip"])
		require.Equal(a, "audit-failed-login-test/2.0", meta["user_agent"])
		require.Equal(a, "admin@example.com", meta["email"])
		require.True(a, failedEvent.TargetID.Valid)
	})
}

func TestAuditLoginFailedUnknownEmailUsesOrganization(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		_ = bootstrapAdmin(t, handler)

		ctx := context.Background()
		queries := db.New(pool)
		org, err := queries.GetFirstOrganization(ctx)
		require.NoError(a, err)

		payload, err := json.Marshal(map[string]string{
			"email":    "nobody@example.com",
			"password": "any-password",
		})
		require.NoError(a, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "audit-unknown-email-test/1.0")
		req.RemoteAddr = "198.51.100.99:8080"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equal(a, http.StatusUnauthorized, rec.Code)

		events, err := queries.ListAuditEventsByOrganization(ctx, org.ID)
		require.NoError(a, err)

		var failedEvent *db.AuditEvent
		for i := range events {
			if events[i].Action == audit.ActionLoginFailed {
				failedEvent = &events[i]
				break
			}
		}
		require.NotNil(a, failedEvent)
		require.False(a, failedEvent.TargetID.Valid)

		var meta map[string]string
		require.NoError(a, json.Unmarshal(failedEvent.Metadata, &meta))
		require.Equal(a, "198.51.100.99", meta["ip"])
		require.Equal(a, "audit-unknown-email-test/1.0", meta["user_agent"])
		require.Equal(a, "nobody@example.com", meta["email"])
	})
}

func TestAuditLogout(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		sessionCookie := bootstrapAdmin(t, handler)

		ctx := context.Background()
		queries := db.New(pool)
		org, err := queries.GetFirstOrganization(ctx)
		require.NoError(a, err)

		logoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/logout", nil)
		logoutReq.AddCookie(sessionCookie)
		logoutReq.Header.Set("User-Agent", "audit-logout-test/1.0")
		logoutReq.RemoteAddr = "198.51.100.50:9999"
		logoutRec := httptest.NewRecorder()
		handler.ServeHTTP(logoutRec, logoutReq)
		require.Equal(a, http.StatusNoContent, logoutRec.Code)

		events, err := queries.ListAuditEventsByOrganization(ctx, org.ID)
		require.NoError(a, err)

		var logoutEvent *db.AuditEvent
		for i := range events {
			if events[i].Action == audit.ActionLogout {
				logoutEvent = &events[i]
				break
			}
		}
		require.NotNil(a, logoutEvent)
		require.True(a, logoutEvent.ActorID.Valid)

		var meta map[string]string
		require.NoError(a, json.Unmarshal(logoutEvent.Metadata, &meta))
		require.Equal(a, "198.51.100.50", meta["ip"])
		require.Equal(a, "audit-logout-test/1.0", meta["user_agent"])
	})
}

func TestAuditStubHooksAppendOnly(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		_ = bootstrapAdmin(t, handler)

		ctx := context.Background()
		queries := db.New(pool)
		org, err := queries.GetFirstOrganization(ctx)
		require.NoError(a, err)

		user, err := queries.GetUserByEmail(ctx, db.GetUserByEmailParams{
			OrganizationID: org.ID,
			Email:          "admin@example.com",
		})
		require.NoError(a, err)

		recorder := audit.NewRecorder(nil)
		keyID := uuid.Must(uuid.NewV7())

		recorder.RoleChanged(ctx, queries, org.ID, user.ID, user.ID, "member", "admin")
		recorder.IntegrationKeyCreated(ctx, queries, org.ID, user.ID, keyID)
		recorder.IntegrationKeyRevoked(ctx, queries, org.ID, user.ID, keyID)

		events, err := queries.ListAuditEventsByOrganization(ctx, org.ID)
		require.NoError(a, err)

		actions := map[string]bool{}
		for _, event := range events {
			actions[event.Action] = true
		}

		require.True(a, actions[audit.ActionRoleChanged])
		require.True(a, actions[audit.ActionIntegrationKeyCreated])
		require.True(a, actions[audit.ActionIntegrationKeyRevoked])

		_ = handler
	})
}
