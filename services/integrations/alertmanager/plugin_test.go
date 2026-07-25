package alertmanager_test

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/integrations"
	_ "github.com/mdg-labs/escalite/services/integrations/alertmanager"
)

func TestPluginRegistered(t *testing.T) {
	plugin, err := integrations.Get("prometheus-alertmanager")
	require.NoError(t, err)
	require.Equal(t, "prometheus-alertmanager", plugin.Name())
}

func TestParseFiringFixture(t *testing.T) {
	raw := loadFixture(t, "firing_v4.json")

	plugin, err := integrations.Get("prometheus-alertmanager")
	require.NoError(t, err)

	alerts, err := integrations.ParseAll(plugin, raw, http.Header{})
	require.NoError(t, err)
	require.Len(t, alerts, 1)

	alert := alerts[0]
	require.Equal(t, integrations.EventTriggered, alert.EventType)
	require.Equal(t, "1a30ba71cca2921f", alert.DedupKey)
	require.Equal(t, "BlackBox Probe Failure: https://server.example.org", alert.Summary)
	require.Contains(t, alert.Description, "has been down for over 5m")
	require.Equal(t, "high", alert.Priority)
	require.Equal(t, "webhook:prometheus-alertmanager", alert.Source)
}

func TestParseResolvedFixture(t *testing.T) {
	raw := loadFixture(t, "resolved_v4.json")

	plugin, err := integrations.Get("prometheus-alertmanager")
	require.NoError(t, err)

	alerts, err := integrations.ParseAll(plugin, raw, http.Header{})
	require.NoError(t, err)
	require.Len(t, alerts, 1)

	alert := alerts[0]
	require.Equal(t, integrations.EventResolved, alert.EventType)
	require.Equal(t, "1a30ba71cca2921f", alert.DedupKey)
}

func TestParseMultiAlertFixture(t *testing.T) {
	raw := loadFixture(t, "multi_alert_v4.json")

	plugin, err := integrations.Get("prometheus-alertmanager")
	require.NoError(t, err)

	alerts, err := integrations.ParseAll(plugin, raw, http.Header{})
	require.NoError(t, err)
	require.Len(t, alerts, 2)

	require.Equal(t, "a8c518a872e3149a", alerts[0].DedupKey)
	require.Equal(t, integrations.EventTriggered, alerts[0].EventType)
	require.Equal(t, "high", alerts[0].Priority)

	require.Equal(t, "8b581f60e6fd0de1", alerts[1].DedupKey)
	require.Equal(t, integrations.EventTriggered, alerts[1].EventType)
	require.Equal(t, "low", alerts[1].Priority)
}

func TestParseAlertRejectsInvalidJSON(t *testing.T) {
	plugin, err := integrations.Get("prometheus-alertmanager")
	require.NoError(t, err)

	_, err = plugin.ParseAlert([]byte(`{`), http.Header{})
	require.Error(t, err)
}

func TestParseAlertRejectsEmptyAlerts(t *testing.T) {
	plugin, err := integrations.Get("prometheus-alertmanager")
	require.NoError(t, err)

	_, err = plugin.ParseAlert([]byte(`{"version":"4","alerts":[]}`), http.Header{})
	require.Error(t, err)
}

func TestParseAlertRejectsUnsupportedVersion(t *testing.T) {
	plugin, err := integrations.Get("prometheus-alertmanager")
	require.NoError(t, err)

	_, err = plugin.ParseAlert([]byte(`{"version":"3","alerts":[{"status":"firing","fingerprint":"abc"}]}`), http.Header{})
	require.Error(t, err)
}

func TestParseAlertRejectsMissingFingerprint(t *testing.T) {
	plugin, err := integrations.Get("prometheus-alertmanager")
	require.NoError(t, err)

	_, err = plugin.ParseAlert([]byte(`{"version":"4","alerts":[{"status":"firing"}]}`), http.Header{})
	require.Error(t, err)
}

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return raw
}
