// Package emailtoalert implements the email-to-alert inbound plugin.
package emailtoalert

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/mdg-labs/escalite/services/integrations"
	"github.com/mdg-labs/escalite/services/integrations/internal/jsonschema"
)

const pluginName = "email-to-alert"

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

	var payload emailPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return integrations.AlertCreate{}, errInvalidPayload
	}

	title, err := extractRequired(payload, mapping.Title, "title")
	if err != nil {
		return integrations.AlertCreate{}, err
	}
	dedupKey, err := extractRequired(payload, mapping.DedupKey, "dedup_key")
	if err != nil {
		return integrations.AlertCreate{}, err
	}

	description, err := extractOptional(payload, mapping.Body, "body")
	if err != nil {
		return integrations.AlertCreate{}, err
	}

	priority := "high"
	if mapping.Priority != "" {
		priority, err = extractRequired(payload, mapping.Priority, "priority")
		if err != nil {
			return integrations.AlertCreate{}, err
		}
	}

	eventType := integrations.EventTriggered
	if mapping.EventType != "" {
		value, err := extractRequired(payload, mapping.EventType, "event_type")
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
		Source:      "email",
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
				"enum": ["subject", "text", "html", "message_id", "from"],
				"title": "Email field used for alert title"
			},
			"body": {
				"type": "string",
				"enum": ["subject", "text", "html", "message_id", "from"],
				"title": "Email field used for alert body"
			},
			"dedup_key": {
				"type": "string",
				"enum": ["subject", "text", "html", "message_id", "from"],
				"title": "Email field used for dedup key"
			},
			"priority": {
				"type": "string",
				"enum": ["subject", "text", "html", "message_id", "from"],
				"title": "Email field used for priority"
			},
			"event_type": {
				"type": "string",
				"enum": ["subject", "text", "html", "message_id", "from"],
				"title": "Email field used for event type"
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

type emailPayload struct {
	To          string `json:"to"`
	From        string `json:"from"`
	Subject     string `json:"subject"`
	Text        string `json:"text"`
	HTML        string `json:"html"`
	MessageID   string `json:"message_id"`
	Authenticated bool `json:"authenticated"`
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

func extractRequired(payload emailPayload, field, name string) (string, error) {
	value, err := extractOptional(payload, field, name)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", missingMappedFieldError{field: name}
	}
	return value, nil
}

func extractOptional(payload emailPayload, field, name string) (string, error) {
	field = strings.TrimSpace(field)
	if field == "" {
		return "", nil
	}

	value := strings.TrimSpace(emailFieldValue(payload, field))
	if value == "" {
		return "", missingMappedFieldError{field: name}
	}
	return value, nil
}

func emailFieldValue(payload emailPayload, field string) string {
	switch field {
	case "subject":
		return payload.Subject
	case "text":
		return payload.Text
	case "html":
		return payload.HTML
	case "message_id":
		return payload.MessageID
	case "from":
		return payload.From
	default:
		return ""
	}
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

func (configRequiredError) Error() string { return "email-to-alert requires integration key mapping config" }

type invalidConfigError struct{}

func (invalidConfigError) Error() string { return "invalid email-to-alert mapping config" }

type invalidPayloadError struct{}

func (invalidPayloadError) Error() string { return "invalid email-to-alert payload" }

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
