package genericrest_test

import (
	"encoding/json"
	"net/http"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/integrations"
	_ "github.com/mdg-labs/escalite/services/integrations/genericrest"
)

func TestPluginRegistered(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("generic-rest-api")
		require.NoError(a, err)
		require.Equal(a, "generic-rest-api", plugin.Name())
	})
}

func TestParseAlertMapsFields(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("generic-rest-api")
		require.NoError(a, err)

		payload := []byte(`{
			"summary": "Disk usage high",
			"description": "Volume /data is 95% full",
			"dedup_key": "host-1-disk",
			"priority": "low"
		}`)

		alert, err := plugin.ParseAlert(payload, http.Header{})
		require.NoError(a, err)
		require.Equal(a, integrations.EventTriggered, alert.EventType)
		require.Equal(a, "host-1-disk", alert.DedupKey)
		require.Equal(a, "Disk usage high", alert.Summary)
		require.Equal(a, "Volume /data is 95% full", alert.Description)
		require.Equal(a, "low", alert.Priority)
		require.Equal(a, "api:generic-rest-api", alert.Source)
	})
}

func TestParseAlertRequiresSummaryAndDedupKey(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("generic-rest-api")
		require.NoError(a, err)

		_, err = plugin.ParseAlert([]byte(`{"summary":"Only summary"}`), http.Header{})
		require.Error(a, err)

		_, err = plugin.ParseAlert([]byte(`{"dedup_key":"only-key"}`), http.Header{})
		require.Error(a, err)
	})
}

func TestParseAlertMapsResolvedEvent(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("generic-rest-api")
		require.NoError(a, err)

		payload := []byte(`{
			"summary": "Recovered",
			"dedup_key": "host-1-disk",
			"event_type": "resolved"
		}`)

		alert, err := plugin.ParseAlert(payload, http.Header{})
		require.NoError(a, err)
		require.Equal(a, integrations.EventResolved, alert.EventType)
	})
}

func TestValidateConfigAcceptsEmptyObject(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("generic-rest-api")
		require.NoError(a, err)

		require.NoError(a, plugin.ValidateConfig(json.RawMessage(`{}`)))
	})
}
