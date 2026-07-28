package uptimekuma_test

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/integrations"
	_ "github.com/mdg-labs/escalite/services/integrations/uptimekuma"
)

func TestPluginRegistered(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("uptime-kuma")
		require.NoError(a, err)
		require.Equal(a, "uptime-kuma", plugin.Name())
	})
}

func TestParseDownFixture(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		raw := loadFixture(t, "down.json")

		plugin, err := integrations.Get("uptime-kuma")
		require.NoError(a, err)

		alerts, err := integrations.ParseAll(plugin, raw, http.Header{}, nil)
		require.NoError(a, err)
		require.Len(a, alerts, 1)

		alert := alerts[0]
		require.Equal(a, integrations.EventTriggered, alert.EventType)
		require.Equal(a, "42", alert.DedupKey)
		require.Equal(a, "checkout-api", alert.Summary)
		require.Contains(a, alert.Description, "Request failed with status code 502")
		require.Equal(a, "high", alert.Priority)
		require.Equal(a, "webhook:uptime-kuma", alert.Source)
	})
}

func TestParseUpFixture(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		raw := loadFixture(t, "up.json")

		plugin, err := integrations.Get("uptime-kuma")
		require.NoError(a, err)

		alerts, err := integrations.ParseAll(plugin, raw, http.Header{}, nil)
		require.NoError(a, err)
		require.Len(a, alerts, 1)

		alert := alerts[0]
		require.Equal(a, integrations.EventResolved, alert.EventType)
		require.Equal(a, "42", alert.DedupKey)
	})
}

func TestParseAlertRejectsInvalidJSON(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("uptime-kuma")
		require.NoError(a, err)

		_, err = plugin.ParseAlert([]byte(`{`), http.Header{})
		require.Error(a, err)
	})
}

func TestParseAlertRejectsMissingMonitorID(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("uptime-kuma")
		require.NoError(a, err)

		_, err = plugin.ParseAlert([]byte(`{"heartbeat":{"status":0},"monitor":{}}`), http.Header{})
		require.Error(a, err)
	})
}

func TestParseAlertRejectsMissingHeartbeatStatus(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("uptime-kuma")
		require.NoError(a, err)

		_, err = plugin.ParseAlert([]byte(`{"heartbeat":{},"monitor":{"id":1}}`), http.Header{})
		require.Error(a, err)
	})
}

func TestValidateConfigAcceptsEmptyObject(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		err := integrations.ValidateConfig("uptime-kuma", []byte(`{}`))
		require.NoError(a, err)
	})
}

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return raw
}
