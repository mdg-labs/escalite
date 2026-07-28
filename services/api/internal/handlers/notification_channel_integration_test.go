package handlers_test

import (
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func TestGraphQLNotificationChannels(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		handler, _, cleanup := newTestHandler(t)
		defer cleanup()

		cookie := bootstrapAdmin(t, handler)

		rec := postGraphQL(t, handler, `{
		notificationChannels {
			name
			configSchema
		}
	}`, cookie)
		require.Equal(a, 200, rec.Code, rec.Body.String())

		var resp struct {
			Data struct {
				NotificationChannels []struct {
					Name         string                 `json:"name"`
					ConfigSchema map[string]interface{} `json:"configSchema"`
				} `json:"notificationChannels"`
			} `json:"data"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Len(a, resp.Data.NotificationChannels, 4)

		names := make([]string, 0, len(resp.Data.NotificationChannels))
		for _, channel := range resp.Data.NotificationChannels {
			names = append(names, channel.Name)
			require.NotEmpty(a, channel.ConfigSchema["type"])
		}
		require.Equal(a, []string{"email", "push", "slack-dm", "webhook"}, names)
	})
}

func TestGraphQLSaveUserContactMethodRejectsUnknownChannel(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
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
		require.Equal(a, 200, rec.Code)

		var resp struct {
			Errors []struct {
				Message    string                 `json:"message"`
				Extensions map[string]interface{} `json:"extensions"`
			} `json:"errors"`
		}
		require.NoError(a, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.NotEmpty(a, resp.Errors)
		require.Equal(a, handlers.CodeValidation, resp.Errors[0].Extensions["code"])
		require.Contains(a, resp.Errors[0].Message, `unknown notification channel "pagerduty"`)
	})
}

func TestGraphQLSaveUserContactMethodAcceptsRegisteredChannel(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
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
		require.Equal(a, "email", resp.Data.SaveUserContactMethod.Channel)
	})
}
