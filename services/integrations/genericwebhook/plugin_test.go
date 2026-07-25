package genericwebhook_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/integrations"
	_ "github.com/mdg-labs/escalite/services/integrations/genericwebhook"
)

func TestPluginRegistered(t *testing.T) {
	plugin, err := integrations.Get("generic-webhook")
	require.NoError(t, err)
	require.Equal(t, "generic-webhook", plugin.Name())
}

func TestValidateConfigRequiresMappingPaths(t *testing.T) {
	plugin, err := integrations.Get("generic-webhook")
	require.NoError(t, err)

	err = plugin.ValidateConfig(json.RawMessage(`{}`))
	require.Error(t, err)

	err = plugin.ValidateConfig(json.RawMessage(`{"title":"title","dedup_key":"id"}`))
	require.NoError(t, err)

	err = plugin.ValidateConfig(json.RawMessage(`{"title":"title","dedup_key":"id","unknown":"x"}`))
	require.Error(t, err)
}

func TestParseAlertWithConfigMapsFields(t *testing.T) {
	plugin, err := integrations.Get("generic-webhook")
	require.NoError(t, err)

	configurable, ok := plugin.(integrations.ConfigurableInboundPlugin)
	require.True(t, ok)

	cfg := json.RawMessage(`{
		"title": "title",
		"body": "message",
		"dedup_key": "id",
		"priority": "severity",
		"event_type": "status"
	}`)
	payload := []byte(`{
		"title": "Disk usage high",
		"message": "Volume /data is 95% full",
		"id": "host-1-disk",
		"severity": "low",
		"status": "firing"
	}`)

	alert, err := configurable.ParseAlertWithConfig(payload, http.Header{}, cfg)
	require.NoError(t, err)
	require.Equal(t, integrations.EventTriggered, alert.EventType)
	require.Equal(t, "host-1-disk", alert.DedupKey)
	require.Equal(t, "Disk usage high", alert.Summary)
	require.Equal(t, "Volume /data is 95% full", alert.Description)
	require.Equal(t, "low", alert.Priority)
	require.Equal(t, "webhook:generic-webhook", alert.Source)
}

func TestParseAlertWithConfigResolvesNestedPaths(t *testing.T) {
	plugin, err := integrations.Get("generic-webhook")
	require.NoError(t, err)

	configurable := plugin.(integrations.ConfigurableInboundPlugin)
	cfg := json.RawMessage(`{"title":"alert.summary","dedup_key":"alert.id"}`)
	payload := []byte(`{"alert":{"summary":"CPU high","id":"cpu-42"}}`)

	alert, err := configurable.ParseAlertWithConfig(payload, http.Header{}, cfg)
	require.NoError(t, err)
	require.Equal(t, "CPU high", alert.Summary)
	require.Equal(t, "cpu-42", alert.DedupKey)
}

func TestParseAlertWithConfigMissingMappedField(t *testing.T) {
	plugin, err := integrations.Get("generic-webhook")
	require.NoError(t, err)

	configurable := plugin.(integrations.ConfigurableInboundPlugin)
	cfg := json.RawMessage(`{"title":"title","dedup_key":"id"}`)
	payload := []byte(`{"title":"Only title present"}`)

	_, err = configurable.ParseAlertWithConfig(payload, http.Header{}, cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "dedup_key")
}

func TestParseAlertWithConfigRejectsInvalidJSON(t *testing.T) {
	plugin, err := integrations.Get("generic-webhook")
	require.NoError(t, err)

	configurable := plugin.(integrations.ConfigurableInboundPlugin)
	cfg := json.RawMessage(`{"title":"title","dedup_key":"id"}`)

	_, err = configurable.ParseAlertWithConfig([]byte(`{`), http.Header{}, cfg)
	require.Error(t, err)
}

func TestParseAlertWithConfigMapsResolvedEvent(t *testing.T) {
	plugin, err := integrations.Get("generic-webhook")
	require.NoError(t, err)

	configurable := plugin.(integrations.ConfigurableInboundPlugin)
	cfg := json.RawMessage(`{"title":"title","dedup_key":"id","event_type":"status"}`)
	payload := []byte(`{"title":"Recovered","id":"cpu-42","status":"resolved"}`)

	alert, err := configurable.ParseAlertWithConfig(payload, http.Header{}, cfg)
	require.NoError(t, err)
	require.Equal(t, integrations.EventResolved, alert.EventType)
}

func TestParseAllUsesIntegrationKeyConfig(t *testing.T) {
	plugin, err := integrations.Get("generic-webhook")
	require.NoError(t, err)

	cfg := json.RawMessage(`{"title":"title","dedup_key":"id"}`)
	payload := []byte(`{"title":"Hello","id":"abc"}`)

	alerts, err := integrations.ParseAll(plugin, payload, http.Header{}, cfg)
	require.NoError(t, err)
	require.Len(t, alerts, 1)
	require.Equal(t, "Hello", alerts[0].Summary)
}
