package emailtoalert_test

import (
	"encoding/json"
	"net/http"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/integrations"
	_ "github.com/mdg-labs/escalite/services/integrations/emailtoalert"
)

func TestPluginRegistered(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("email-to-alert")
		require.NoError(a, err)
		require.Equal(a, "email-to-alert", plugin.Name())
	})
}

func TestValidateConfigRequiresMappingFields(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("email-to-alert")
		require.NoError(a, err)

		err = plugin.ValidateConfig(json.RawMessage(`{}`))
		require.Error(a, err)

		err = plugin.ValidateConfig(json.RawMessage(`{"title":"subject","dedup_key":"message_id"}`))
		require.NoError(a, err)

		err = plugin.ValidateConfig(json.RawMessage(`{"title":"subject","dedup_key":"message_id","unknown":"x"}`))
		require.Error(a, err)
	})
}

func TestParseAlertWithConfigMapsEmailFields(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("email-to-alert")
		require.NoError(a, err)

		configurable, ok := plugin.(integrations.ConfigurableInboundPlugin)
		require.True(a, ok)

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
		require.NoError(a, err)
		require.Equal(a, integrations.EventTriggered, alert.EventType)
		require.Equal(a, "<abc@mail.example.com>", alert.DedupKey)
		require.Equal(a, "low: CPU usage high", alert.Summary)
		require.Equal(a, "CPU usage is 95%", alert.Description)
		require.Equal(a, "high", alert.Priority)
		require.Equal(a, "email", alert.Source)
	})
}

func TestParseAlertWithConfigMapsResolvedEvent(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("email-to-alert")
		require.NoError(a, err)

		configurable := plugin.(integrations.ConfigurableInboundPlugin)
		cfg := json.RawMessage(`{"title":"subject","dedup_key":"message_id","event_type":"text"}`)
		payload := []byte(`{
			"subject": "Recovered",
			"text": "resolved",
			"message_id": "<abc@mail.example.com>"
		}`)

		alert, err := configurable.ParseAlertWithConfig(payload, http.Header{}, cfg)
		require.NoError(a, err)
		require.Equal(a, integrations.EventResolved, alert.EventType)
	})
}

func TestParseAlertWithConfigMissingMappedField(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("email-to-alert")
		require.NoError(a, err)

		configurable := plugin.(integrations.ConfigurableInboundPlugin)
		cfg := json.RawMessage(`{"title":"subject","dedup_key":"message_id"}`)
		payload := []byte(`{"subject":"Only subject present"}`)

		_, err = configurable.ParseAlertWithConfig(payload, http.Header{}, cfg)
		require.Error(a, err)
		require.Contains(a, err.Error(), "dedup_key")
	})
}
