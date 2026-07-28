package handlers_test

import (
	"context"
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func TestGraphQLSaveSlackSettingsReturnsHintOnly(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := newTestHandler(t)
		defer cleanup()

		cookie := bootstrapAdmin(t, handler)

		const botToken = "xoxb-1234567890abcdefghijklmnop"
		rec := postGraphQL(t, handler, `mutation {
		saveSlackSettings(input: { botToken: "`+botToken+`" }) {
			configured
			tokenHint
		}
	}`, cookie)
		require.Equal(a, 200, rec.Code, rec.Body.String())

		var resp struct {
			Data struct {
				SaveSlackSettings struct {
					Configured bool   `json:"configured"`
					TokenHint  string `json:"tokenHint"`
				} `json:"saveSlackSettings"`
			} `json:"data"`
			Errors []any `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Empty(a, resp.Errors)
		require.True(a, resp.Data.SaveSlackSettings.Configured)
		require.Equal(a, "mnop", resp.Data.SaveSlackSettings.TokenHint)
		require.NotContains(a, rec.Body.String(), botToken)
	})
}

func TestGraphQLSlackSettingsNeverReturnsFullToken(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := newTestHandler(t)
		defer cleanup()

		cookie := bootstrapAdmin(t, handler)

		const botToken = "xoxb-1234567890abcdefghijklmnop"
		saveRec := postGraphQL(t, handler, `mutation {
		saveSlackSettings(input: { botToken: "`+botToken+`" }) {
			configured
			tokenHint
		}
	}`, cookie)
		require.Equal(a, 200, saveRec.Code, saveRec.Body.String())

		rec := postGraphQL(t, handler, `{
		slackSettings {
			configured
			tokenHint
		}
	}`, cookie)
		require.Equal(a, 200, rec.Code, rec.Body.String())

		var resp struct {
			Data struct {
				SlackSettings struct {
					Configured bool   `json:"configured"`
					TokenHint  string `json:"tokenHint"`
				} `json:"slackSettings"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.True(a, resp.Data.SlackSettings.Configured)
		require.Equal(a, "mnop", resp.Data.SlackSettings.TokenHint)
		require.NotContains(a, rec.Body.String(), botToken)
	})
}

func TestGraphQLSaveSlackSettingsRequiresAdmin(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, pool, cleanup := newTestHandler(t)
		defer cleanup()

		_ = bootstrapAdmin(t, handler)

		queries := db.New(pool)
		admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
		require.NoError(a, err)

		_ = seedMemberUser(t, pool, admin.OrganizationID, "member@example.com", "member-password-123")
		memberCookie := loginUser(t, handler, "member@example.com", "member-password-123")

		rec := postGraphQL(t, handler, `mutation {
		saveSlackSettings(input: { botToken: "xoxb-test-token" }) {
			configured
		}
	}`, memberCookie)
		require.Equal(a, 200, rec.Code)

		var resp struct {
			Errors []struct {
				Extensions map[string]interface{} `json:"extensions"`
			} `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.NotEmpty(a, resp.Errors)
		require.Equal(a, handlers.CodeForbidden, resp.Errors[0].Extensions["code"])
	})
}

func TestGraphQLSaveUserContactMethodPersistsSlackDMConfig(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := newTestHandler(t)
		defer cleanup()

		cookie := bootstrapAdmin(t, handler)

		rec := postGraphQL(t, handler, `mutation {
		saveUserContactMethod(input: {
			channel: "slack-dm"
			config: { slack_user_id: "U12345678" }
		}) {
			channel
			config
		}
	}`, cookie)
		require.Equal(a, 200, rec.Code, rec.Body.String())

		var resp struct {
			Data struct {
				SaveUserContactMethod struct {
					Channel string                 `json:"channel"`
					Config  map[string]interface{} `json:"config"`
				} `json:"saveUserContactMethod"`
			} `json:"data"`
			Errors []any `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Empty(a, resp.Errors)
		require.Equal(a, "slack-dm", resp.Data.SaveUserContactMethod.Channel)
		require.Equal(a, "U12345678", resp.Data.SaveUserContactMethod.Config["slack_user_id"])
	})
}
