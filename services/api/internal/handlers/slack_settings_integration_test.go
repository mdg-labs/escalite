package handlers_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func TestGraphQLSaveSlackSettingsReturnsHintOnly(t *testing.T) {
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
	require.Equal(t, 200, rec.Code, rec.Body.String())

	var resp struct {
		Data struct {
			SaveSlackSettings struct {
				Configured bool   `json:"configured"`
				TokenHint  string `json:"tokenHint"`
			} `json:"saveSlackSettings"`
		} `json:"data"`
		Errors []any `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Empty(t, resp.Errors)
	require.True(t, resp.Data.SaveSlackSettings.Configured)
	require.Equal(t, "mnop", resp.Data.SaveSlackSettings.TokenHint)
	require.NotContains(t, rec.Body.String(), botToken)
}

func TestGraphQLSlackSettingsNeverReturnsFullToken(t *testing.T) {
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
	require.Equal(t, 200, saveRec.Code, saveRec.Body.String())

	rec := postGraphQL(t, handler, `{
		slackSettings {
			configured
			tokenHint
		}
	}`, cookie)
	require.Equal(t, 200, rec.Code, rec.Body.String())

	var resp struct {
		Data struct {
			SlackSettings struct {
				Configured bool   `json:"configured"`
				TokenHint  string `json:"tokenHint"`
			} `json:"slackSettings"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.True(t, resp.Data.SlackSettings.Configured)
	require.Equal(t, "mnop", resp.Data.SlackSettings.TokenHint)
	require.NotContains(t, rec.Body.String(), botToken)
}

func TestGraphQLSaveSlackSettingsRequiresAdmin(t *testing.T) {
	handler, pool, cleanup := newTestHandler(t)
	defer cleanup()

	_ = bootstrapAdmin(t, handler)

	queries := db.New(pool)
	admin, err := queries.GetUserByEmailForAuth(context.Background(), "admin@example.com")
	require.NoError(t, err)

	_ = seedMemberUser(t, pool, admin.OrganizationID, "member@example.com", "member-password-123")
	memberCookie := loginUser(t, handler, "member@example.com", "member-password-123")

	rec := postGraphQL(t, handler, `mutation {
		saveSlackSettings(input: { botToken: "xoxb-test-token" }) {
			configured
		}
	}`, memberCookie)
	require.Equal(t, 200, rec.Code)

	var resp struct {
		Errors []struct {
			Extensions map[string]interface{} `json:"extensions"`
		} `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Errors)
	require.Equal(t, handlers.CodeForbidden, resp.Errors[0].Extensions["code"])
}

func TestGraphQLSaveUserContactMethodPersistsSlackDMConfig(t *testing.T) {
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
	require.Equal(t, 200, rec.Code, rec.Body.String())

	var resp struct {
		Data struct {
			SaveUserContactMethod struct {
				Channel string                 `json:"channel"`
				Config  map[string]interface{} `json:"config"`
			} `json:"saveUserContactMethod"`
		} `json:"data"`
		Errors []any `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Empty(t, resp.Errors)
	require.Equal(t, "slack-dm", resp.Data.SaveUserContactMethod.Channel)
	require.Equal(t, "U12345678", resp.Data.SaveUserContactMethod.Config["slack_user_id"])
}
