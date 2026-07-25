// Package genericrest implements the generic-rest-api inbound plugin.
package genericrest

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/mdg-labs/escalite/services/integrations"
	"github.com/mdg-labs/escalite/services/integrations/internal/jsonschema"
)

const pluginName = "generic-rest-api"

func init() {
	integrations.Register(plugin{})
}

type plugin struct{}

func (plugin) Name() string { return pluginName }

func (p plugin) ParseAlert(raw []byte, headers http.Header) (integrations.AlertCreate, error) {
	_ = headers

	if !json.Valid(raw) {
		return integrations.AlertCreate{}, errInvalidPayload
	}

	var payload alertPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return integrations.AlertCreate{}, errInvalidPayload
	}

	if err := jsonschema.ValidateDocument(payloadSchema(), raw); err != nil {
		return integrations.AlertCreate{}, errInvalidPayload
	}

	summary := strings.TrimSpace(payload.Summary)
	dedupKey := strings.TrimSpace(payload.DedupKey)
	if summary == "" || dedupKey == "" {
		return integrations.AlertCreate{}, errInvalidPayload
	}

	eventType := integrations.EventTriggered
	if payload.EventType != "" {
		mapped, err := mapEventType(payload.EventType)
		if err != nil {
			return integrations.AlertCreate{}, err
		}
		eventType = mapped
	}

	priority := "high"
	if payload.Priority != "" {
		priority = mapPriority(payload.Priority)
	}

	description := strings.TrimSpace(payload.Description)

	return integrations.AlertCreate{
		EventType:   eventType,
		DedupKey:    dedupKey,
		Summary:     summary,
		Description: description,
		Priority:    priority,
		Source:      "api:" + pluginName,
	}, nil
}

func (plugin) ValidateConfig(json.RawMessage) error { return nil }

func (plugin) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","additionalProperties":false}`)
}

type alertPayload struct {
	Summary     string `json:"summary"`
	Description string `json:"description,omitempty"`
	DedupKey    string `json:"dedup_key"`
	Priority    string `json:"priority,omitempty"`
	EventType   string `json:"event_type,omitempty"`
}

func payloadSchema() json.RawMessage {
	return json.RawMessage(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"additionalProperties": false,
		"properties": {
			"summary": {
				"type": "string",
				"minLength": 1
			},
			"description": {
				"type": "string"
			},
			"dedup_key": {
				"type": "string",
				"minLength": 1
			},
			"priority": {
				"type": "string",
				"enum": ["low", "high"]
			},
			"event_type": {
				"type": "string",
				"enum": ["triggered", "resolved"]
			}
		},
		"required": ["summary", "dedup_key"]
	}`)
}

func mapEventType(value string) (integrations.EventType, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "triggered":
		return integrations.EventTriggered, nil
	case "resolved":
		return integrations.EventResolved, nil
	default:
		return "", errInvalidEventType
	}
}

func mapPriority(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "low":
		return "low"
	default:
		return "high"
	}
}

type invalidPayloadError struct{}

func (invalidPayloadError) Error() string { return "invalid generic-rest-api payload" }

type invalidEventTypeError struct{}

func (invalidEventTypeError) Error() string { return "invalid event_type value" }

var (
	errInvalidPayload   = invalidPayloadError{}
	errInvalidEventType = invalidEventTypeError{}
)
