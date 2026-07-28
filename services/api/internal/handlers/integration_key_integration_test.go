package handlers_test

import (
	"context"
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
	_ "github.com/mdg-labs/escalite/services/integrations/install"
)

func TestGraphQLCreateIntegrationKeyBeszelPreset(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

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
		require.Equal(a, 200, createRec.Code, createRec.Body.String())

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
		require.NoError(a, json.Unmarshal(createRec.Body.Bytes(), &createResp))
		require.Equal(a, service.ID.String(), createResp.Data.CreateIntegrationKey.ServiceID)
		require.Equal(a, "generic-webhook", createResp.Data.CreateIntegrationKey.PluginName)
		require.Equal(a, "title", createResp.Data.CreateIntegrationKey.Config["title"])
		require.Equal(a, "message", createResp.Data.CreateIntegrationKey.Config["body"])
		require.Equal(a, "title", createResp.Data.CreateIntegrationKey.Config["dedup_key"])
		require.NotEmpty(a, createResp.Data.CreateIntegrationKey.TokenPrefix)
		require.NotEmpty(a, createResp.Data.CreateIntegrationKey.Token)

		payload := []byte(`{
		"title": "Disk usage high",
		"message": "Volume /data is 95% full"
	}`)
		rec := postInboundWebhook(t, handler, "generic-webhook", createResp.Data.CreateIntegrationKey.Token, payload)
		require.Equal(a, 202, rec.Code, rec.Body.String())

		alert, err := queries.GetOpenAlertByServiceDedupKeyForResolve(context.Background(), db.GetOpenAlertByServiceDedupKeyForResolveParams{
			ServiceID: service.ID,
			DedupKey:  "Disk usage high",
		})
		require.NoError(a, err)
		require.Equal(a, "triggered", alert.Status)
		require.Equal(a, "Disk usage high", alert.Summary)
		require.Equal(a, "Volume /data is 95% full", alert.Description.String)
	})
}

func TestGraphQLCreateIntegrationKeyRejectsInvalidPlugin(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

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
		require.Equal(a, 200, createRec.Code)

		var errResp struct {
			Errors []struct {
				Message    string `json:"message"`
				Extensions struct {
					Code string `json:"code"`
				} `json:"extensions"`
			} `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(createRec.Body.Bytes(), &errResp))
		require.NotEmpty(a, errResp.Errors)
		require.Equal(a, handlers.CodeValidation, errResp.Errors[0].Extensions.Code)
	})
}

func TestGraphQLRevokeIntegrationKeyReturns404OnWebhook(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

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
		require.Equal(a, 200, createRec.Code, createRec.Body.String())

		var createResp struct {
			Data struct {
				CreateIntegrationKey struct {
					ID    string `json:"id"`
					Token string `json:"token"`
				} `json:"createIntegrationKey"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(createRec.Body.Bytes(), &createResp))
		require.NotEmpty(a, createResp.Data.CreateIntegrationKey.Token)

		token := createResp.Data.CreateIntegrationKey.Token
		keyID := createResp.Data.CreateIntegrationKey.ID

		payload := []byte(`{"summary":"test"}`)
		activeRec := postInboundWebhook(t, handler, "test-plugin", token, payload)
		require.Equal(a, 202, activeRec.Code, activeRec.Body.String())

		revokeRec := postGraphQL(t, handler, `mutation {
		revokeIntegrationKey(id: "`+keyID+`") {
			id
			revokedAt
			tokenPrefix
		}
	}`, adminCookie)
		require.Equal(a, 200, revokeRec.Code, revokeRec.Body.String())

		var revokeResp struct {
			Data struct {
				RevokeIntegrationKey struct {
					ID          string `json:"id"`
					RevokedAt   string `json:"revokedAt"`
					TokenPrefix string `json:"tokenPrefix"`
				} `json:"revokeIntegrationKey"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(revokeRec.Body.Bytes(), &revokeResp))
		require.NotEmpty(a, revokeResp.Data.RevokeIntegrationKey.RevokedAt)
		require.NotEmpty(a, revokeResp.Data.RevokeIntegrationKey.TokenPrefix)

		revokedRec := postInboundWebhook(t, handler, "test-plugin", token, payload)
		require.Equal(a, 404, revokedRec.Code, revokedRec.Body.String())
	})
}

func TestGraphQLIntegrationKeysListsPrefixWithoutToken(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		adminCookie := bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

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
		require.Equal(a, 200, createRec.Code)

		listRec := postGraphQL(t, handler, `query {
		integrationKeys(serviceId: "`+service.ID.String()+`") {
			id
			tokenPrefix
			token
			pluginName
		}
	}`, adminCookie)
		require.Equal(a, 200, listRec.Code, listRec.Body.String())

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
		require.NoError(a, json.Unmarshal(listRec.Body.Bytes(), &listResp))
		require.Len(a, listResp.Data.IntegrationKeys, 1)
		require.NotEmpty(a, listResp.Data.IntegrationKeys[0].TokenPrefix)
		require.Nil(a, listResp.Data.IntegrationKeys[0].Token)
		require.Equal(a, "test-plugin", listResp.Data.IntegrationKeys[0].PluginName)
	})
}
