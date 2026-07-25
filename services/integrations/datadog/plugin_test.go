package datadog_test

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/integrations"
	"github.com/mdg-labs/escalite/services/integrations/internal/webhookauth"
	_ "github.com/mdg-labs/escalite/services/integrations/datadog"
)

func TestPluginRegistered(t *testing.T) {
	plugin, err := integrations.Get("datadog")
	require.NoError(t, err)
	require.Equal(t, "datadog", plugin.Name())
}

func TestParseTriggeredFixture(t *testing.T) {
	raw := loadFixture(t, "triggered.json")

	plugin, err := integrations.Get("datadog")
	require.NoError(t, err)

	alerts, err := integrations.ParseAll(plugin, raw, http.Header{}, nil)
	require.NoError(t, err)
	require.Len(t, alerts, 1)

	alert := alerts[0]
	require.Equal(t, integrations.EventTriggered, alert.EventType)
	require.Equal(t, "monitor:12345|tags:env:prod,service:api", alert.DedupKey)
	require.Equal(t, "CPU usage above 90%", alert.Summary)
	require.Equal(t, "CPU is at 94% on host web-01", alert.Description)
	require.Equal(t, "low", alert.Priority)
	require.Equal(t, "webhook:datadog", alert.Source)
}

func TestParseRecoveredFixture(t *testing.T) {
	raw := loadFixture(t, "recovered.json")

	plugin, err := integrations.Get("datadog")
	require.NoError(t, err)

	alerts, err := integrations.ParseAll(plugin, raw, http.Header{}, nil)
	require.NoError(t, err)
	require.Len(t, alerts, 1)

	alert := alerts[0]
	require.Equal(t, integrations.EventResolved, alert.EventType)
	require.Equal(t, "monitor:12345|tags:env:prod,service:api", alert.DedupKey)
}

func TestAuthenticateRequestRejectsInvalidSignature(t *testing.T) {
	plugin, err := integrations.Get("datadog")
	require.NoError(t, err)

	auth, ok := plugin.(integrations.AuthenticatableInboundPlugin)
	require.True(t, ok)

	raw := loadFixture(t, "triggered.json")
	cfg := json.RawMessage(`{"signature_secret":"top-secret"}`)
	headers := http.Header{}
	headers.Set("X-Escalite-Signature", "invalid")

	err = auth.AuthenticateRequest(raw, headers, cfg)
	require.Error(t, err)

	var sigErr integrations.ErrInvalidSignature
	require.ErrorAs(t, err, &sigErr)
}

func TestAuthenticateRequestAcceptsValidSignature(t *testing.T) {
	plugin, err := integrations.Get("datadog")
	require.NoError(t, err)

	auth := plugin.(integrations.AuthenticatableInboundPlugin)

	raw := loadFixture(t, "triggered.json")
	secret := "top-secret"
	cfg := json.RawMessage(`{"signature_secret":"top-secret"}`)
	headers := http.Header{}
	headers.Set("X-Escalite-Signature", webhookauth.SignBody(secret, raw))

	err = auth.AuthenticateRequest(raw, headers, cfg)
	require.NoError(t, err)
}

func TestParseAlertRejectsInvalidJSON(t *testing.T) {
	plugin, err := integrations.Get("datadog")
	require.NoError(t, err)

	_, err = plugin.ParseAlert([]byte(`{`), http.Header{})
	require.Error(t, err)
}

func TestParseAlertRejectsMissingMonitorID(t *testing.T) {
	plugin, err := integrations.Get("datadog")
	require.NoError(t, err)

	_, err = plugin.ParseAlert([]byte(`{"title":"CPU high"}`), http.Header{})
	require.Error(t, err)
}

func TestValidateConfigRejectsUnknownField(t *testing.T) {
	plugin, err := integrations.Get("datadog")
	require.NoError(t, err)

	err = plugin.ValidateConfig(json.RawMessage(`{"unknown":"x"}`))
	require.Error(t, err)
}

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return raw
}
