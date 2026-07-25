// Package datadog implements the datadog inbound plugin.
package datadog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/mdg-labs/escalite/services/integrations"
	"github.com/mdg-labs/escalite/services/integrations/internal/jsonschema"
	"github.com/mdg-labs/escalite/services/integrations/internal/webhookauth"
)

const pluginName = "datadog"

func init() {
	integrations.Register(plugin{})
}

type plugin struct{}

func (plugin) Name() string { return pluginName }

func (plugin) ParseAlert(raw []byte, headers http.Header) (integrations.AlertCreate, error) {
	return plugin{}.ParseAlertWithConfig(raw, headers, json.RawMessage(`{}`))
}

func (p plugin) ParseAlertWithConfig(raw []byte, headers http.Header, cfg json.RawMessage) (integrations.AlertCreate, error) {
	if err := p.AuthenticateRequest(raw, headers, cfg); err != nil {
		return integrations.AlertCreate{}, err
	}

	event, err := parseWebhookEvent(raw)
	if err != nil {
		return integrations.AlertCreate{}, err
	}
	return event.toAlertCreate()
}

func (p plugin) AuthenticateRequest(raw []byte, headers http.Header, cfg json.RawMessage) error {
	config, err := parseConfig(cfg)
	if err != nil {
		return err
	}
	return webhookauth.Verify(config.SignatureSecret, raw, headers)
}

func (plugin) ValidateConfig(cfg json.RawMessage) error {
	return jsonschema.ValidateDocument(configSchema(), cfg)
}

func (plugin) ConfigSchema() json.RawMessage {
	return configSchema()
}

type pluginConfig struct {
	SignatureSecret string `json:"signature_secret,omitempty"`
}

type webhookEvent struct {
	Title           string          `json:"title"`
	EventTitle      string          `json:"event_title"`
	Msg             string          `json:"msg"`
	Text            string          `json:"text"`
	EventMsg        string          `json:"event_msg"`
	TextOnlyMsg     string          `json:"text_only_msg"`
	AlertType       string          `json:"alert_type"`
	AlertTransition string          `json:"alert_transition"`
	AlertStatus     string          `json:"alert_status"`
	Priority        string          `json:"priority"`
	MonitorID       json.RawMessage `json:"monitor_id"`
	ID              string          `json:"id"`
	Tags            json.RawMessage `json:"tags"`
	Link            string          `json:"link"`
}

func (e webhookEvent) toAlertCreate() (integrations.AlertCreate, error) {
	monitorID, err := monitorIDFromEvent(e)
	if err != nil {
		return integrations.AlertCreate{}, err
	}

	tags, err := tagsFromEvent(e)
	if err != nil {
		return integrations.AlertCreate{}, err
	}

	summary := firstNonEmpty(
		e.Title,
		e.EventTitle,
		"Datadog monitor alert",
	)
	description := firstNonEmpty(
		e.Msg,
		e.Text,
		e.EventMsg,
		e.TextOnlyMsg,
		e.Link,
	)

	return integrations.AlertCreate{
		EventType:   mapEventType(e.AlertTransition, e.AlertType, e.AlertStatus),
		DedupKey:    buildDedupKey(monitorID, tags),
		Summary:     summary,
		Description: description,
		Priority:    mapPriority(e.Priority, e.AlertType),
		Source:      "webhook:" + pluginName,
	}, nil
}

func parseWebhookEvent(raw []byte) (webhookEvent, error) {
	if !json.Valid(raw) {
		return webhookEvent{}, errInvalidPayload
	}

	var event webhookEvent
	if err := json.Unmarshal(raw, &event); err != nil {
		return webhookEvent{}, errInvalidPayload
	}
	return event, nil
}

func monitorIDFromEvent(event webhookEvent) (string, error) {
	if len(event.MonitorID) > 0 && string(event.MonitorID) != "null" {
		var asNumber int64
		if err := json.Unmarshal(event.MonitorID, &asNumber); err == nil {
			return strconv.FormatInt(asNumber, 10), nil
		}
		var asString string
		if err := json.Unmarshal(event.MonitorID, &asString); err == nil {
			if trimmed := strings.TrimSpace(asString); trimmed != "" {
				return trimmed, nil
			}
		}
	}

	if fallback := strings.TrimSpace(event.ID); fallback != "" {
		return fallback, nil
	}

	return "", errMissingMonitorID
}

func tagsFromEvent(event webhookEvent) ([]string, error) {
	if len(event.Tags) == 0 || string(event.Tags) == "null" {
		return nil, nil
	}

	var tags []string
	if err := json.Unmarshal(event.Tags, &tags); err == nil {
		return normalizeTags(tags), nil
	}

	var tagString string
	if err := json.Unmarshal(event.Tags, &tagString); err == nil {
		return normalizeTags(strings.Split(tagString, ",")), nil
	}

	return nil, errInvalidTags
}

func normalizeTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			out = append(out, tag)
		}
	}
	sort.Strings(out)
	return out
}

func buildDedupKey(monitorID string, tags []string) string {
	return fmt.Sprintf("monitor:%s|tags:%s", monitorID, strings.Join(tags, ","))
}

func mapEventType(transition, alertType, alertStatus string) integrations.EventType {
	for _, value := range []string{transition, alertType, alertStatus} {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "recovered", "recovery", "ok", "resolved", "success":
			return integrations.EventResolved
		}
	}
	return integrations.EventTriggered
}

func mapPriority(priority, alertType string) string {
	switch strings.ToLower(strings.TrimSpace(priority)) {
	case "low", "normal":
		return "low"
	}
	switch strings.ToLower(strings.TrimSpace(alertType)) {
	case "warning", "info":
		return "low"
	default:
		return "high"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func parseConfig(cfg json.RawMessage) (pluginConfig, error) {
	if len(cfg) == 0 {
		cfg = json.RawMessage(`{}`)
	}
	if err := jsonschema.ValidateDocument(configSchema(), cfg); err != nil {
		return pluginConfig{}, errInvalidConfig
	}

	var config pluginConfig
	if err := json.Unmarshal(cfg, &config); err != nil {
		return pluginConfig{}, errInvalidConfig
	}
	return config, nil
}

func configSchema() json.RawMessage {
	return json.RawMessage(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"additionalProperties": false,
		"properties": {
			"signature_secret": {
				"type": "string",
				"minLength": 1,
				"title": "Webhook signature secret"
			}
		}
	}`)
}


type invalidConfigError struct{}

func (invalidConfigError) Error() string { return "invalid datadog config" }

type invalidPayloadError struct{}

func (invalidPayloadError) Error() string { return "invalid datadog webhook payload" }

type missingMonitorIDError struct{}

func (missingMonitorIDError) Error() string { return "datadog webhook missing monitor id" }

type invalidTagsError struct{}

func (invalidTagsError) Error() string { return "invalid datadog tags" }

var (
	errInvalidConfig    = invalidConfigError{}
	errInvalidPayload   = invalidPayloadError{}
	errMissingMonitorID = missingMonitorIDError{}
	errInvalidTags      = invalidTagsError{}
)
