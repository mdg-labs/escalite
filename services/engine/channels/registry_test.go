package channels_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/engine/channels"
	_ "github.com/mdg-labs/escalite/services/engine/channelsinstall"
)

func TestRegistryListsDayOneChannels(t *testing.T) {
	names := channels.Names()
	require.Equal(t, []string{"email", "push", "slack-dm", "webhook"}, names)
}

func TestValidateConfigUnknownChannel(t *testing.T) {
	err := channels.ValidateConfig("pagerduty", json.RawMessage(`{}`))
	require.Error(t, err)

	var unknown channels.ErrUnknownChannel
	require.ErrorAs(t, err, &unknown)
	require.Equal(t, "pagerduty", unknown.Name)
}

func TestValidateConfigPushRequiresToken(t *testing.T) {
	err := channels.ValidateConfig("push", json.RawMessage(`{}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "expo_push_token")

	err = channels.ValidateConfig("push", json.RawMessage(`{"expo_push_token":"ExponentPushToken[abc]"}`))
	require.NoError(t, err)
}

func TestValidateConfigWebhookRequiresURL(t *testing.T) {
	err := channels.ValidateConfig("webhook", json.RawMessage(`{"url":"not-a-url"}`))
	require.Error(t, err)

	err = channels.ValidateConfig("webhook", json.RawMessage(`{"url":"https://example.com/hooks/escalite"}`))
	require.NoError(t, err)
}

func TestConfigSchemaReturnsJSONForKnownChannel(t *testing.T) {
	schema, err := channels.ConfigSchema("email")
	require.NoError(t, err)
	require.Contains(t, string(schema), `"type": "object"`)
}

func TestConfigSchemaUnknownChannel(t *testing.T) {
	_, err := channels.ConfigSchema("sms")
	require.Error(t, err)

	var unknown channels.ErrUnknownChannel
	require.ErrorAs(t, err, &unknown)
}
