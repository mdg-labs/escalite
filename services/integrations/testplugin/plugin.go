// Package testplugin registers a stub inbound plugin for integration tests.
package testplugin

import (
	"encoding/json"
	"net/http"

	"github.com/mdg-labs/escalite/services/integrations"
)

func init() {
	integrations.Register(stubPlugin{})
}

type stubPlugin struct{}

func (stubPlugin) Name() string { return "test-plugin" }

func (stubPlugin) ParseAlert(raw []byte, _ http.Header) (integrations.AlertCreate, error) {
	if !json.Valid(raw) {
		return integrations.AlertCreate{}, errInvalidJSON
	}
	return integrations.AlertCreate{
		EventType: integrations.EventTriggered,
		DedupKey:  "test",
		Summary:   "test alert",
	}, nil
}

func (stubPlugin) ValidateConfig(json.RawMessage) error { return nil }

func (stubPlugin) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object"}`)
}

type invalidJSONError struct{}

func (invalidJSONError) Error() string { return "invalid JSON body" }

var errInvalidJSON = invalidJSONError{}
