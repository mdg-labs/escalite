package emailtoalert_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/integrations"
	_ "github.com/mdg-labs/escalite/services/integrations/emailtoalert"
)

func TestPluginRegistered(t *testing.T) {
	plugin, err := integrations.Get("email-to-alert")
	require.NoError(t, err)
	require.Equal(t, "email-to-alert", plugin.Name())
}

func TestValidateConfigRequiresMappingFields(t *testing.T) {
	plugin, err := integrations.Get("email-to-alert")
	require.NoError(t, err)

	err = plugin.ValidateConfig(json.RawMessage(`{}`))
	require.Error(t, err)

	err = plugin.ValidateConfig(json.RawMessage(`{"title":"subject","dedup_key":"message_id"}`))
	require.NoError(t, err)

	err = plugin.ValidateConfig(json.RawMessage(`{"title":"subject","dedup_key":"message_id","unknown":"x"}`))
	require.Error(t, err)
}

func TestParseAlertWithConfigMapsEmailFields(t *testing.T) {
	plugin, err := integrations.Get("email-to-alert")
	require.NoError(t, err)

	configurable, ok := plugin.(integrations.ConfigurableInboundPlugin)
	require.True(t, ok)

	cfg := json.RawMessage(`{
		"title": "subject",
		"body": "text",
		"dedup_key": "message_id",
		"priority": "subject"
	}`)
	payload := []byte(`{
		"to": "token@inbound.example.com",
		"from": "monitor@example.com",
		"subject": "low: CPU usage high",
		"text": "CPU usage is 95%",
		"message_id": "<abc@mail.example.com>"
	}`)

	alert, err := configurable.ParseAlertWithConfig(payload, http.Header{}, cfg)
	require.NoError(t, err)
	require.Equal(t, integrations.EventTriggered, alert.EventType)
	require.Equal(t, "<abc@mail.example.com>", alert.DedupKey)
	require.Equal(t, "low: CPU usage high", alert.Summary)
	require.Equal(t, "CPU usage is 95%", alert.Description)
	require.Equal(t, "high", alert.Priority)
	require.Equal(t, "email", alert.Source)
}

func TestParseAlertWithConfigMapsResolvedEvent(t *testing.T) {
	plugin, err := integrations.Get("email-to-alert")
	require.NoError(t, err)

	configurable := plugin.(integrations.ConfigurableInboundPlugin)
	cfg := json.RawMessage(`{"title":"subject","dedup_key":"message_id","event_type":"text"}`)
	payload := []byte(`{
		"subject": "Recovered",
		"text": "resolved",
		"message_id": "<abc@mail.example.com>"
	}`)

	alert, err := configurable.ParseAlertWithConfig(payload, http.Header{}, cfg)
	require.NoError(t, err)
	require.Equal(t, integrations.EventResolved, alert.EventType)
}

func TestParseAlertWithConfigMissingMappedField(t *testing.T) {
	plugin, err := integrations.Get("email-to-alert")
	require.NoError(t, err)

	configurable := plugin.(integrations.ConfigurableInboundPlugin)
	cfg := json.RawMessage(`{"title":"subject","dedup_key":"message_id"}`)
	payload := []byte(`{"subject":"Only subject present"}`)

	_, err = configurable.ParseAlertWithConfig(payload, http.Header{}, cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "dedup_key")
}
