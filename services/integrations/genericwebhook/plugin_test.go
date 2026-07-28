package genericwebhook_test

import (
	"encoding/json"
	"net/http"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/integrations"
	_ "github.com/mdg-labs/escalite/services/integrations/genericwebhook"
)

func TestPluginRegistered(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("generic-webhook")
		require.NoError(a, err)
		require.Equal(a, "generic-webhook", plugin.Name())
	})
}

func TestValidateConfigRequiresMappingPaths(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("generic-webhook")
		require.NoError(a, err)

		err = plugin.ValidateConfig(json.RawMessage(`{}`))
		require.Error(a, err)

		err = plugin.ValidateConfig(json.RawMessage(`{"title":"title","dedup_key":"id"}`))
		require.NoError(a, err)

		err = plugin.ValidateConfig(json.RawMessage(`{"title":"title","dedup_key":"id","unknown":"x"}`))
		require.Error(a, err)
	})
}

func TestParseAlertWithConfigMapsFields(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("generic-webhook")
		require.NoError(a, err)

		configurable, ok := plugin.(integrations.ConfigurableInboundPlugin)
		require.True(a, ok)

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
		require.NoError(a, err)
		require.Equal(a, integrations.EventTriggered, alert.EventType)
		require.Equal(a, "host-1-disk", alert.DedupKey)
		require.Equal(a, "Disk usage high", alert.Summary)
		require.Equal(a, "Volume /data is 95% full", alert.Description)
		require.Equal(a, "low", alert.Priority)
		require.Equal(a, "webhook:generic-webhook", alert.Source)
	})
}

func TestParseAlertWithConfigResolvesNestedPaths(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("generic-webhook")
		require.NoError(a, err)

		configurable := plugin.(integrations.ConfigurableInboundPlugin)
		cfg := json.RawMessage(`{"title":"alert.summary","dedup_key":"alert.id"}`)
		payload := []byte(`{"alert":{"summary":"CPU high","id":"cpu-42"}}`)

		alert, err := configurable.ParseAlertWithConfig(payload, http.Header{}, cfg)
		require.NoError(a, err)
		require.Equal(a, "CPU high", alert.Summary)
		require.Equal(a, "cpu-42", alert.DedupKey)
	})
}

func TestParseAlertWithConfigMissingMappedField(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("generic-webhook")
		require.NoError(a, err)

		configurable := plugin.(integrations.ConfigurableInboundPlugin)
		cfg := json.RawMessage(`{"title":"title","dedup_key":"id"}`)
		payload := []byte(`{"title":"Only title present"}`)

		_, err = configurable.ParseAlertWithConfig(payload, http.Header{}, cfg)
		require.Error(a, err)
		require.Contains(a, err.Error(), "dedup_key")
	})
}

func TestParseAlertWithConfigRejectsInvalidJSON(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("generic-webhook")
		require.NoError(a, err)

		configurable := plugin.(integrations.ConfigurableInboundPlugin)
		cfg := json.RawMessage(`{"title":"title","dedup_key":"id"}`)

		_, err = configurable.ParseAlertWithConfig([]byte(`{`), http.Header{}, cfg)
		require.Error(a, err)
	})
}

func TestParseAlertWithConfigMapsResolvedEvent(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("generic-webhook")
		require.NoError(a, err)

		configurable := plugin.(integrations.ConfigurableInboundPlugin)
		cfg := json.RawMessage(`{"title":"title","dedup_key":"id","event_type":"status"}`)
		payload := []byte(`{"title":"Recovered","id":"cpu-42","status":"resolved"}`)

		alert, err := configurable.ParseAlertWithConfig(payload, http.Header{}, cfg)
		require.NoError(a, err)
		require.Equal(a, integrations.EventResolved, alert.EventType)
	})
}

func TestParseAllUsesIntegrationKeyConfig(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("generic-webhook")
		require.NoError(a, err)

		cfg := json.RawMessage(`{"title":"title","dedup_key":"id"}`)
		payload := []byte(`{"title":"Hello","id":"abc"}`)

		alerts, err := integrations.ParseAll(plugin, payload, http.Header{}, cfg)
		require.NoError(a, err)
		require.Len(a, alerts, 1)
		require.Equal(a, "Hello", alerts[0].Summary)
	})
}
