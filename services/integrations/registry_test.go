package integrations_test

import (
	"encoding/json"
	"net/http"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/integrations"
	_ "github.com/mdg-labs/escalite/services/integrations/datadog"
	_ "github.com/mdg-labs/escalite/services/integrations/grafana"
	_ "github.com/mdg-labs/escalite/services/integrations/uptimekuma"
)

func TestEventTypeConstants(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		require.Equal(a, integrations.EventType("triggered"), integrations.EventTriggered)
		require.Equal(a, integrations.EventType("resolved"), integrations.EventResolved)
	})
}

func TestGetUnknownPlugin(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		_, err := integrations.Get("nonexistent-plugin")
		require.Error(a, err)

		var unknown integrations.ErrUnknownPlugin
		require.ErrorAs(a, err, &unknown)
		require.Equal(a, "nonexistent-plugin", unknown.Name)
	})
}

func TestValidateConfigUnknownPlugin(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		err := integrations.ValidateConfig("newrelic", json.RawMessage(`{}`))
		require.Error(a, err)

		var unknown integrations.ErrUnknownPlugin
		require.ErrorAs(a, err, &unknown)
		require.Equal(a, "newrelic", unknown.Name)
	})
}

func TestValidateConfigGrafana(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		err := integrations.ValidateConfig("grafana", json.RawMessage(`{}`))
		require.NoError(a, err)
	})
}

func TestValidateConfigUptimeKuma(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		err := integrations.ValidateConfig("uptime-kuma", json.RawMessage(`{}`))
		require.NoError(a, err)
	})
}

func TestConfigSchemaDatadogPlugin(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		schema, err := integrations.ConfigSchema("datadog")
		require.NoError(a, err)
		require.Contains(a, string(schema), "signature_secret")
	})
}

func TestInboundPluginInterfaceDocumentsResolvedEvents(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		var plugin integrations.InboundPlugin = stubPlugin{}
		alert, err := plugin.ParseAlert([]byte(`{}`), http.Header{})
		require.NoError(a, err)
		require.Equal(a, integrations.EventResolved, alert.EventType)
	})
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
