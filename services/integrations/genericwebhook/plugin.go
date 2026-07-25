// Package genericwebhook implements the generic-webhook inbound plugin.
package genericwebhook

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/tidwall/gjson"

	"github.com/mdg-labs/escalite/services/integrations"
	"github.com/mdg-labs/escalite/services/integrations/internal/jsonschema"
)

const pluginName = "generic-webhook"

func init() {
	integrations.Register(plugin{})
}

type plugin struct{}

func (plugin) Name() string { return pluginName }

func (plugin) ParseAlert([]byte, http.Header) (integrations.AlertCreate, error) {
	return integrations.AlertCreate{}, errConfigRequired
}

func (p plugin) ParseAlertWithConfig(raw []byte, headers http.Header, cfg json.RawMessage) (integrations.AlertCreate, error) {
	_ = headers

	mapping, err := parseMapping(cfg)
	if err != nil {
		return integrations.AlertCreate{}, err
	}

	if !json.Valid(raw) {
		return integrations.AlertCreate{}, errInvalidPayload
	}

	title, err := extractRequired(raw, mapping.Title, "title")
	if err != nil {
		return integrations.AlertCreate{}, err
	}
	dedupKey, err := extractRequired(raw, mapping.DedupKey, "dedup_key")
	if err != nil {
		return integrations.AlertCreate{}, err
	}

	description, err := extractOptional(raw, mapping.Body, "body")
	if err != nil {
		return integrations.AlertCreate{}, err
	}

	priority := "high"
	if mapping.Priority != "" {
		priority, err = extractRequired(raw, mapping.Priority, "priority")
		if err != nil {
			return integrations.AlertCreate{}, err
		}
	}

	eventType := integrations.EventTriggered
	if mapping.EventType != "" {
		value, err := extractRequired(raw, mapping.EventType, "event_type")
		if err != nil {
			return integrations.AlertCreate{}, err
		}
		eventType, err = mapEventType(value)
		if err != nil {
			return integrations.AlertCreate{}, err
		}
	}

	return integrations.AlertCreate{
		EventType:   eventType,
		DedupKey:    dedupKey,
		Summary:     title,
		Description: description,
		Priority:    mapPriority(priority),
		Source:      "webhook:" + pluginName,
	}, nil
}

func (plugin) ValidateConfig(cfg json.RawMessage) error {
	return jsonschema.ValidateDocument(configSchema(), cfg)
}

func (plugin) ConfigSchema() json.RawMessage {
	return configSchema()
}

func configSchema() json.RawMessage {
	return json.RawMessage(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"additionalProperties": false,
		"properties": {
			"title": {
				"type": "string",
				"minLength": 1,
				"title": "Title JSON path"
			},
			"body": {
				"type": "string",
				"minLength": 1,
				"title": "Body JSON path"
			},
			"dedup_key": {
				"type": "string",
				"minLength": 1,
				"title": "Dedup key JSON path"
			},
			"priority": {
				"type": "string",
				"minLength": 1,
				"title": "Priority JSON path"
			},
			"event_type": {
				"type": "string",
				"minLength": 1,
				"title": "Event type JSON path"
			}
		},
		"required": ["title", "dedup_key"]
	}`)
}

type mappingConfig struct {
	Title     string `json:"title"`
	Body      string `json:"body"`
	DedupKey  string `json:"dedup_key"`
	Priority  string `json:"priority"`
	EventType string `json:"event_type"`
}

func parseMapping(cfg json.RawMessage) (mappingConfig, error) {
	if len(cfg) == 0 {
		cfg = json.RawMessage(`{}`)
	}
	if err := jsonschema.ValidateDocument(configSchema(), cfg); err != nil {
		return mappingConfig{}, errInvalidConfig
	}

	var mapping mappingConfig
	if err := json.Unmarshal(cfg, &mapping); err != nil {
		return mappingConfig{}, errInvalidConfig
	}
	return mapping, nil
}

func extractRequired(raw []byte, path, field string) (string, error) {
	value, err := extractOptional(raw, path, field)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", missingMappedFieldError{field: field}
	}
	return value, nil
}

func extractOptional(raw []byte, path, field string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}

	result := gjson.GetBytes(raw, path)
	if !result.Exists() || result.Type == gjson.Null {
		return "", missingMappedFieldError{field: field}
	}

	value := strings.TrimSpace(result.String())
	if value == "" {
		return "", missingMappedFieldError{field: field}
	}
	return value, nil
}

func mapEventType(value string) (integrations.EventType, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "triggered", "firing", "alert", "open", "active":
		return integrations.EventTriggered, nil
	case "resolved", "ok", "recovery", "closed", "inactive":
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

type configRequiredError struct{}

func (configRequiredError) Error() string { return "generic-webhook requires integration key mapping config" }

type invalidConfigError struct{}

func (invalidConfigError) Error() string { return "invalid generic-webhook mapping config" }

type invalidPayloadError struct{}

func (invalidPayloadError) Error() string { return "invalid generic-webhook payload" }

type missingMappedFieldError struct {
	field string
}

func (e missingMappedFieldError) Error() string {
	return "missing mapped field " + e.field
}

type invalidEventTypeError struct{}

func (invalidEventTypeError) Error() string { return "invalid mapped event_type value" }

var (
	errConfigRequired   = configRequiredError{}
	errInvalidConfig    = invalidConfigError{}
	errInvalidPayload   = invalidPayloadError{}
	errInvalidEventType = invalidEventTypeError{}
)
