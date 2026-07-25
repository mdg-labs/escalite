package handlers_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func TestGraphQLNotificationChannels(t *testing.T) {
	handler, _, cleanup := newTestHandler(t)
	defer cleanup()

	cookie := bootstrapAdmin(t, handler)

	rec := postGraphQL(t, handler, `{
		notificationChannels {
			name
			configSchema
		}
	}`, cookie)
	require.Equal(t, 200, rec.Code, rec.Body.String())

	var resp struct {
		Data struct {
			NotificationChannels []struct {
				Name         string                 `json:"name"`
				ConfigSchema map[string]interface{} `json:"configSchema"`
			} `json:"notificationChannels"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Data.NotificationChannels, 4)

	names := make([]string, 0, len(resp.Data.NotificationChannels))
	for _, channel := range resp.Data.NotificationChannels {
		names = append(names, channel.Name)
		require.NotEmpty(t, channel.ConfigSchema["type"])
	}
	require.Equal(t, []string{"email", "push", "slack-dm", "webhook"}, names)
}

func TestGraphQLSaveUserContactMethodRejectsUnknownChannel(t *testing.T) {
	handler, _, cleanup := newTestHandler(t)
	defer cleanup()

	cookie := bootstrapAdmin(t, handler)

	rec := postGraphQL(t, handler, `mutation {
		saveUserContactMethod(input: {
			channel: "pagerduty"
			config: {}
		}) {
			id
		}
	}`, cookie)
	require.Equal(t, 200, rec.Code)

	var resp struct {
		Errors []struct {
			Message    string                 `json:"message"`
			Extensions map[string]interface{} `json:"extensions"`
		} `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Errors)
	require.Equal(t, handlers.CodeValidation, resp.Errors[0].Extensions["code"])
	require.Contains(t, resp.Errors[0].Message, `unknown notification channel "pagerduty"`)
}

func TestGraphQLSaveUserContactMethodAcceptsRegisteredChannel(t *testing.T) {
	handler, _, cleanup := newTestHandler(t)
	defer cleanup()

	cookie := bootstrapAdmin(t, handler)

	rec := postGraphQL(t, handler, `mutation {
		saveUserContactMethod(input: {
			channel: "email"
			config: {}
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
	require.Equal(t, "email", resp.Data.SaveUserContactMethod.Channel)
}
