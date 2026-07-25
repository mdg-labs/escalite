package uptimekuma_test

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/integrations"
	_ "github.com/mdg-labs/escalite/services/integrations/uptimekuma"
)

func TestPluginRegistered(t *testing.T) {
	plugin, err := integrations.Get("uptime-kuma")
	require.NoError(t, err)
	require.Equal(t, "uptime-kuma", plugin.Name())
}

func TestParseDownFixture(t *testing.T) {
	raw := loadFixture(t, "down.json")

	plugin, err := integrations.Get("uptime-kuma")
	require.NoError(t, err)

	alerts, err := integrations.ParseAll(plugin, raw, http.Header{}, nil)
	require.NoError(t, err)
	require.Len(t, alerts, 1)

	alert := alerts[0]
	require.Equal(t, integrations.EventTriggered, alert.EventType)
	require.Equal(t, "42", alert.DedupKey)
	require.Equal(t, "checkout-api", alert.Summary)
	require.Contains(t, alert.Description, "Request failed with status code 502")
	require.Equal(t, "high", alert.Priority)
	require.Equal(t, "webhook:uptime-kuma", alert.Source)
}

func TestParseUpFixture(t *testing.T) {
	raw := loadFixture(t, "up.json")

	plugin, err := integrations.Get("uptime-kuma")
	require.NoError(t, err)

	alerts, err := integrations.ParseAll(plugin, raw, http.Header{}, nil)
	require.NoError(t, err)
	require.Len(t, alerts, 1)

	alert := alerts[0]
	require.Equal(t, integrations.EventResolved, alert.EventType)
	require.Equal(t, "42", alert.DedupKey)
}

func TestParseAlertRejectsInvalidJSON(t *testing.T) {
	plugin, err := integrations.Get("uptime-kuma")
	require.NoError(t, err)

	_, err = plugin.ParseAlert([]byte(`{`), http.Header{})
	require.Error(t, err)
}

func TestParseAlertRejectsMissingMonitorID(t *testing.T) {
	plugin, err := integrations.Get("uptime-kuma")
	require.NoError(t, err)

	_, err = plugin.ParseAlert([]byte(`{"heartbeat":{"status":0},"monitor":{}}`), http.Header{})
	require.Error(t, err)
}

func TestParseAlertRejectsMissingHeartbeatStatus(t *testing.T) {
	plugin, err := integrations.Get("uptime-kuma")
	require.NoError(t, err)

	_, err = plugin.ParseAlert([]byte(`{"heartbeat":{},"monitor":{"id":1}}`), http.Header{})
	require.Error(t, err)
}

func TestValidateConfigAcceptsEmptyObject(t *testing.T) {
	err := integrations.ValidateConfig("uptime-kuma", []byte(`{}`))
	require.NoError(t, err)
}

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return raw
}
