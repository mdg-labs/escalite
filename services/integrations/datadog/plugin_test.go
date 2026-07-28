package datadog_test

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/integrations"
	"github.com/mdg-labs/escalite/services/integrations/internal/webhookauth"
	_ "github.com/mdg-labs/escalite/services/integrations/datadog"
)

func TestPluginRegistered(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("datadog")
		require.NoError(a, err)
		require.Equal(a, "datadog", plugin.Name())
	})
}

func TestParseTriggeredFixture(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		raw := loadFixture(t, "triggered.json")

		plugin, err := integrations.Get("datadog")
		require.NoError(a, err)

		alerts, err := integrations.ParseAll(plugin, raw, http.Header{}, nil)
		require.NoError(a, err)
		require.Len(a, alerts, 1)

		alert := alerts[0]
		require.Equal(a, integrations.EventTriggered, alert.EventType)
		require.Equal(a, "monitor:12345|tags:env:prod,service:api", alert.DedupKey)
		require.Equal(a, "CPU usage above 90%", alert.Summary)
		require.Equal(a, "CPU is at 94% on host web-01", alert.Description)
		require.Equal(a, "low", alert.Priority)
		require.Equal(a, "webhook:datadog", alert.Source)
	})
}

func TestParseRecoveredFixture(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		raw := loadFixture(t, "recovered.json")

		plugin, err := integrations.Get("datadog")
		require.NoError(a, err)

		alerts, err := integrations.ParseAll(plugin, raw, http.Header{}, nil)
		require.NoError(a, err)
		require.Len(a, alerts, 1)

		alert := alerts[0]
		require.Equal(a, integrations.EventResolved, alert.EventType)
		require.Equal(a, "monitor:12345|tags:env:prod,service:api", alert.DedupKey)
	})
}

func TestAuthenticateRequestRejectsInvalidSignature(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("datadog")
		require.NoError(a, err)

		auth, ok := plugin.(integrations.AuthenticatableInboundPlugin)
		require.True(a, ok)

		raw := loadFixture(t, "triggered.json")
		cfg := json.RawMessage(`{"signature_secret":"top-secret"}`)
		headers := http.Header{}
		headers.Set("X-Escalite-Signature", "invalid")

		err = auth.AuthenticateRequest(raw, headers, cfg)
		require.Error(a, err)

		var sigErr integrations.ErrInvalidSignature
		require.ErrorAs(a, err, &sigErr)
	})
}

func TestAuthenticateRequestAcceptsValidSignature(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("datadog")
		require.NoError(a, err)

		auth := plugin.(integrations.AuthenticatableInboundPlugin)

		raw := loadFixture(t, "triggered.json")
		secret := "top-secret"
		cfg := json.RawMessage(`{"signature_secret":"top-secret"}`)
		headers := http.Header{}
		headers.Set("X-Escalite-Signature", webhookauth.SignBody(secret, raw))

		err = auth.AuthenticateRequest(raw, headers, cfg)
		require.NoError(a, err)
	})
}

func TestParseAlertRejectsInvalidJSON(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("datadog")
		require.NoError(a, err)

		_, err = plugin.ParseAlert([]byte(`{`), http.Header{})
		require.Error(a, err)
	})
}

func TestParseAlertRejectsMissingMonitorID(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("datadog")
		require.NoError(a, err)

		_, err = plugin.ParseAlert([]byte(`{"title":"CPU high"}`), http.Header{})
		require.Error(a, err)
	})
}

func TestValidateConfigRejectsUnknownField(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		plugin, err := integrations.Get("datadog")
		require.NoError(a, err)

		err = plugin.ValidateConfig(json.RawMessage(`{"unknown":"x"}`))
		require.Error(a, err)
	})
}

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return raw
}
