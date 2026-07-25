package genericrest_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/integrations"
	_ "github.com/mdg-labs/escalite/services/integrations/genericrest"
)

func TestPluginRegistered(t *testing.T) {
	plugin, err := integrations.Get("generic-rest-api")
	require.NoError(t, err)
	require.Equal(t, "generic-rest-api", plugin.Name())
}

func TestParseAlertMapsFields(t *testing.T) {
	plugin, err := integrations.Get("generic-rest-api")
	require.NoError(t, err)

	payload := []byte(`{
		"summary": "Disk usage high",
		"description": "Volume /data is 95% full",
		"dedup_key": "host-1-disk",
		"priority": "low"
	}`)

	alert, err := plugin.ParseAlert(payload, http.Header{})
	require.NoError(t, err)
	require.Equal(t, integrations.EventTriggered, alert.EventType)
	require.Equal(t, "host-1-disk", alert.DedupKey)
	require.Equal(t, "Disk usage high", alert.Summary)
	require.Equal(t, "Volume /data is 95% full", alert.Description)
	require.Equal(t, "low", alert.Priority)
	require.Equal(t, "api:generic-rest-api", alert.Source)
}

func TestParseAlertRequiresSummaryAndDedupKey(t *testing.T) {
	plugin, err := integrations.Get("generic-rest-api")
	require.NoError(t, err)

	_, err = plugin.ParseAlert([]byte(`{"summary":"Only summary"}`), http.Header{})
	require.Error(t, err)

	_, err = plugin.ParseAlert([]byte(`{"dedup_key":"only-key"}`), http.Header{})
	require.Error(t, err)
}

func TestParseAlertMapsResolvedEvent(t *testing.T) {
	plugin, err := integrations.Get("generic-rest-api")
	require.NoError(t, err)

	payload := []byte(`{
		"summary": "Recovered",
		"dedup_key": "host-1-disk",
		"event_type": "resolved"
	}`)

	alert, err := plugin.ParseAlert(payload, http.Header{})
	require.NoError(t, err)
	require.Equal(t, integrations.EventResolved, alert.EventType)
}

func TestValidateConfigAcceptsEmptyObject(t *testing.T) {
	plugin, err := integrations.Get("generic-rest-api")
	require.NoError(t, err)

	require.NoError(t, plugin.ValidateConfig(json.RawMessage(`{}`)))
}
