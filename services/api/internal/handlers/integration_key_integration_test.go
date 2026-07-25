package handlers_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
	_ "github.com/mdg-labs/escalite/services/integrations/install"
)

func TestGraphQLCreateIntegrationKeyBeszelPreset(t *testing.T) {
	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)

	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	team := seedTeam(t, pool, admin.OrganizationID, "Platform")
	service := seedService(t, pool, admin.OrganizationID, team.ID, "beszel-hosts")

	createRec := postGraphQL(t, handler, `mutation {
		createIntegrationKey(input: {
			serviceId: "`+service.ID.String()+`"
			pluginName: "generic-webhook"
			config: {
				title: "title"
				body: "message"
				dedup_key: "title"
			}
		}) {
			id
			serviceId
			pluginName
			config
			tokenPrefix
			token
		}
	}`, adminCookie)
	require.Equal(t, 200, createRec.Code, createRec.Body.String())

	var createResp struct {
		Data struct {
			CreateIntegrationKey struct {
				ID          string         `json:"id"`
				ServiceID   string         `json:"serviceId"`
				PluginName  string         `json:"pluginName"`
				Config      map[string]any `json:"config"`
				TokenPrefix string         `json:"tokenPrefix"`
				Token       string         `json:"token"`
			} `json:"createIntegrationKey"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &createResp))
	require.Equal(t, service.ID.String(), createResp.Data.CreateIntegrationKey.ServiceID)
	require.Equal(t, "generic-webhook", createResp.Data.CreateIntegrationKey.PluginName)
	require.Equal(t, "title", createResp.Data.CreateIntegrationKey.Config["title"])
	require.Equal(t, "message", createResp.Data.CreateIntegrationKey.Config["body"])
	require.Equal(t, "title", createResp.Data.CreateIntegrationKey.Config["dedup_key"])
	require.NotEmpty(t, createResp.Data.CreateIntegrationKey.TokenPrefix)
	require.NotEmpty(t, createResp.Data.CreateIntegrationKey.Token)

	payload := []byte(`{
		"title": "Disk usage high",
		"message": "Volume /data is 95% full"
	}`)
	rec := postInboundWebhook(t, handler, "generic-webhook", createResp.Data.CreateIntegrationKey.Token, payload)
	require.Equal(t, 202, rec.Code, rec.Body.String())

	alert, err := queries.GetOpenAlertByServiceDedupKey(context.Background(), db.GetOpenAlertByServiceDedupKeyParams{
		ServiceID: service.ID,
		DedupKey:  "Disk usage high",
	})
	require.NoError(t, err)
	require.Equal(t, "triggered", alert.Status)
	require.Equal(t, "Disk usage high", alert.Summary)
	require.Equal(t, "Volume /data is 95% full", alert.Description.String)
}

func TestGraphQLCreateIntegrationKeyRejectsInvalidPlugin(t *testing.T) {
	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)

	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	team := seedTeam(t, pool, admin.OrganizationID, "Platform")
	service := seedService(t, pool, admin.OrganizationID, team.ID, "checkout-api")

	createRec := postGraphQL(t, handler, `mutation {
		createIntegrationKey(input: {
			serviceId: "`+service.ID.String()+`"
			pluginName: "not-a-plugin"
			config: { title: "title", dedup_key: "title" }
		}) {
			id
		}
	}`, adminCookie)
	require.Equal(t, 200, createRec.Code)

	var errResp struct {
		Errors []struct {
			Message string `json:"message"`
			Extensions struct {
				Code string `json:"code"`
			} `json:"extensions"`
		} `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &errResp))
	require.NotEmpty(t, errResp.Errors)
	require.Equal(t, handlers.CodeValidation, errResp.Errors[0].Extensions.Code)
}

func TestGraphQLRevokeIntegrationKeyReturns404OnWebhook(t *testing.T) {
	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)

	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	team := seedTeam(t, pool, admin.OrganizationID, "Platform")
	service := seedService(t, pool, admin.OrganizationID, team.ID, "webhooks")

	createRec := postGraphQL(t, handler, `mutation {
		createIntegrationKey(input: {
			serviceId: "`+service.ID.String()+`"
			pluginName: "test-plugin"
			config: {}
		}) {
			id
			token
		}
	}`, adminCookie)
	require.Equal(t, 200, createRec.Code, createRec.Body.String())

	var createResp struct {
		Data struct {
			CreateIntegrationKey struct {
				ID    string `json:"id"`
				Token string `json:"token"`
			} `json:"createIntegrationKey"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &createResp))
	require.NotEmpty(t, createResp.Data.CreateIntegrationKey.Token)

	token := createResp.Data.CreateIntegrationKey.Token
	keyID := createResp.Data.CreateIntegrationKey.ID

	payload := []byte(`{"summary":"test"}`)
	activeRec := postInboundWebhook(t, handler, "test-plugin", token, payload)
	require.Equal(t, 202, activeRec.Code, activeRec.Body.String())

	revokeRec := postGraphQL(t, handler, `mutation {
		revokeIntegrationKey(id: "`+keyID+`") {
			id
			revokedAt
			tokenPrefix
		}
	}`, adminCookie)
	require.Equal(t, 200, revokeRec.Code, revokeRec.Body.String())

	var revokeResp struct {
		Data struct {
			RevokeIntegrationKey struct {
				ID          string `json:"id"`
				RevokedAt   string `json:"revokedAt"`
				TokenPrefix string `json:"tokenPrefix"`
			} `json:"revokeIntegrationKey"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(revokeRec.Body.Bytes(), &revokeResp))
	require.NotEmpty(t, revokeResp.Data.RevokeIntegrationKey.RevokedAt)
	require.NotEmpty(t, revokeResp.Data.RevokeIntegrationKey.TokenPrefix)

	revokedRec := postInboundWebhook(t, handler, "test-plugin", token, payload)
	require.Equal(t, 404, revokedRec.Code, revokedRec.Body.String())
}

func TestGraphQLIntegrationKeysListsPrefixWithoutToken(t *testing.T) {
	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	adminCookie := bootstrapAdmin(t, handler)

	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	team := seedTeam(t, pool, admin.OrganizationID, "Platform")
	service := seedService(t, pool, admin.OrganizationID, team.ID, "api")

	createRec := postGraphQL(t, handler, `mutation {
		createIntegrationKey(input: {
			serviceId: "`+service.ID.String()+`"
			pluginName: "test-plugin"
			config: {}
		}) {
			id
			tokenPrefix
			token
		}
	}`, adminCookie)
	require.Equal(t, 200, createRec.Code)

	listRec := postGraphQL(t, handler, `query {
		integrationKeys(serviceId: "`+service.ID.String()+`") {
			id
			tokenPrefix
			token
			pluginName
		}
	}`, adminCookie)
	require.Equal(t, 200, listRec.Code, listRec.Body.String())

	var listResp struct {
		Data struct {
			IntegrationKeys []struct {
				ID          string  `json:"id"`
				TokenPrefix string  `json:"tokenPrefix"`
				Token       *string `json:"token"`
				PluginName  string  `json:"pluginName"`
			} `json:"integrationKeys"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(listRec.Body.Bytes(), &listResp))
	require.Len(t, listResp.Data.IntegrationKeys, 1)
	require.NotEmpty(t, listResp.Data.IntegrationKeys[0].TokenPrefix)
	require.Nil(t, listResp.Data.IntegrationKeys[0].Token)
	require.Equal(t, "test-plugin", listResp.Data.IntegrationKeys[0].PluginName)
}
