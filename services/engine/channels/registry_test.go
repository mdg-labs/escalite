package channels_test

import (
	"encoding/json"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/engine/channels"
	_ "github.com/mdg-labs/escalite/services/engine/channelsinstall"
)

func TestRegistryListsDayOneChannels(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		names := channels.Names()
		require.Equal(a, []string{"email", "push", "slack-dm", "webhook"}, names)
	})
}

func TestValidateConfigUnknownChannel(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		err := channels.ValidateConfig("pagerduty", json.RawMessage(`{}`))
		require.Error(a, err)

		var unknown channels.ErrUnknownChannel
		require.ErrorAs(a, err, &unknown)
		require.Equal(a, "pagerduty", unknown.Name)
	})
}

func TestValidateConfigPushRequiresToken(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		err := channels.ValidateConfig("push", json.RawMessage(`{}`))
		require.Error(a, err)
		require.Contains(a, err.Error(), "expo_push_token")

		err = channels.ValidateConfig("push", json.RawMessage(`{"expo_push_token":"ExponentPushToken[abc]"}`))
		require.NoError(a, err)
	})
}

func TestValidateConfigWebhookRequiresURL(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		err := channels.ValidateConfig("webhook", json.RawMessage(`{"url":"not-a-url"}`))
		require.Error(a, err)

		err = channels.ValidateConfig("webhook", json.RawMessage(`{"url":"https://example.com/hooks/escalite"}`))
		require.NoError(a, err)
	})
}

func TestConfigSchemaReturnsJSONForKnownChannel(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		schema, err := channels.ConfigSchema("email")
		require.NoError(a, err)
		require.Contains(a, string(schema), `"type": "object"`)
	})
}

func TestConfigSchemaUnknownChannel(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		_, err := channels.ConfigSchema("sms")
		require.Error(a, err)

		var unknown channels.ErrUnknownChannel
		require.ErrorAs(a, err, &unknown)
	})
}
