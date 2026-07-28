package grafana_test

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/integrations"
	_ "github.com/mdg-labs/escalite/services/integrations/grafana"
)

func TestPluginRegistered(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("grafana")
		require.NoError(a, err)
		require.Equal(a, "grafana", plugin.Name())
	})
}

func TestParseFiringFixture(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		raw := loadFixture(t, "firing.json")

		plugin, err := integrations.Get("grafana")
		require.NoError(a, err)

		alerts, err := integrations.ParseAll(plugin, raw, http.Header{}, nil)
		require.NoError(a, err)
		require.Len(a, alerts, 1)

		alert := alerts[0]
		require.Equal(a, integrations.EventTriggered, alert.EventType)
		require.Equal(a, "c6eadffa33fcdf37", alert.DedupKey)
		require.Equal(a, "High memory usage on db-primary-01", alert.Summary)
		require.Contains(a, alert.Description, "Memory usage on db-primary-01 has been above 90%")
		require.Equal(a, "high", alert.Priority)
		require.Equal(a, "webhook:grafana", alert.Source)
	})
}

func TestParseResolvedFixture(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		raw := loadFixture(t, "resolved.json")

		plugin, err := integrations.Get("grafana")
		require.NoError(a, err)

		alerts, err := integrations.ParseAll(plugin, raw, http.Header{}, nil)
		require.NoError(a, err)
		require.Len(a, alerts, 1)

		alert := alerts[0]
		require.Equal(a, integrations.EventResolved, alert.EventType)
		require.Equal(a, "c6eadffa33fcdf37", alert.DedupKey)
	})
}

func TestParseMultiAlertFixture(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		raw := loadFixture(t, "multi_alert.json")

		plugin, err := integrations.Get("grafana")
		require.NoError(a, err)

		alerts, err := integrations.ParseAll(plugin, raw, http.Header{}, nil)
		require.NoError(a, err)
		require.Len(a, alerts, 2)

		require.Equal(a, "a8c518a872e3149a", alerts[0].DedupKey)
		require.Equal(a, integrations.EventTriggered, alerts[0].EventType)
		require.Equal(a, "high", alerts[0].Priority)

		require.Equal(a, "8b581f60e6fd0de1", alerts[1].DedupKey)
		require.Equal(a, integrations.EventTriggered, alerts[1].EventType)
		require.Equal(a, "low", alerts[1].Priority)
	})
}

func TestParseAlertRejectsInvalidJSON(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("grafana")
		require.NoError(a, err)

		_, err = plugin.ParseAlert([]byte(`{`), http.Header{})
		require.Error(a, err)
	})
}

func TestParseAlertRejectsEmptyAlerts(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("grafana")
		require.NoError(a, err)

		_, err = plugin.ParseAlert([]byte(`{"version":"1","alerts":[]}`), http.Header{})
		require.Error(a, err)
	})
}

func TestParseAlertRejectsUnsupportedVersion(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("grafana")
		require.NoError(a, err)

		_, err = plugin.ParseAlert([]byte(`{"version":"4","alerts":[{"status":"firing","fingerprint":"abc"}]}`), http.Header{})
		require.Error(a, err)
	})
}

func TestParseAlertRejectsMissingFingerprint(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("grafana")
		require.NoError(a, err)

		_, err = plugin.ParseAlert([]byte(`{"version":"1","alerts":[{"status":"firing"}]}`), http.Header{})
		require.Error(a, err)
	})
}

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return raw
}
