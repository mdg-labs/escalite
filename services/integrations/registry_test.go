package integrations_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/integrations"
	_ "github.com/mdg-labs/escalite/services/integrations/datadog"
	_ "github.com/mdg-labs/escalite/services/integrations/grafana"
	_ "github.com/mdg-labs/escalite/services/integrations/uptimekuma"
)

func TestEventTypeConstants(t *testing.T) {
	require.Equal(t, integrations.EventType("triggered"), integrations.EventTriggered)
	require.Equal(t, integrations.EventType("resolved"), integrations.EventResolved)
}

func TestGetUnknownPlugin(t *testing.T) {
	_, err := integrations.Get("nonexistent-plugin")
	require.Error(t, err)

	var unknown integrations.ErrUnknownPlugin
	require.ErrorAs(t, err, &unknown)
	require.Equal(t, "nonexistent-plugin", unknown.Name)
}

func TestValidateConfigUnknownPlugin(t *testing.T) {
	err := integrations.ValidateConfig("newrelic", json.RawMessage(`{}`))
	require.Error(t, err)

	var unknown integrations.ErrUnknownPlugin
	require.ErrorAs(t, err, &unknown)
	require.Equal(t, "newrelic", unknown.Name)
}

func TestValidateConfigGrafana(t *testing.T) {
	err := integrations.ValidateConfig("grafana", json.RawMessage(`{}`))
	require.NoError(t, err)
}

func TestValidateConfigUptimeKuma(t *testing.T) {
	err := integrations.ValidateConfig("uptime-kuma", json.RawMessage(`{}`))
	require.NoError(t, err)
}

func TestConfigSchemaDatadogPlugin(t *testing.T) {
	schema, err := integrations.ConfigSchema("datadog")
	require.NoError(t, err)
	require.Contains(t, string(schema), "signature_secret")
}

func TestInboundPluginInterfaceDocumentsResolvedEvents(t *testing.T) {
	var plugin integrations.InboundPlugin = stubPlugin{}
	alert, err := plugin.ParseAlert([]byte(`{}`), http.Header{})
	require.NoError(t, err)
	require.Equal(t, integrations.EventResolved, alert.EventType)
}

type stubPlugin struct{}

func (stubPlugin) Name() string { return "stub" }

func (stubPlugin) ParseAlert(_ []byte, _ http.Header) (integrations.AlertCreate, error) {
	return integrations.AlertCreate{
		EventType: integrations.EventResolved,
		DedupKey:  "example",
		Summary:   "resolved",
	}, nil
}

func (stubPlugin) ValidateConfig(json.RawMessage) error { return nil }

func (stubPlugin) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object"}`)
}
