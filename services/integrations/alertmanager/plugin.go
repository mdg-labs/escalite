// Package alertmanager implements the prometheus-alertmanager inbound plugin.
package alertmanager

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/mdg-labs/escalite/services/integrations"
)

const pluginName = "prometheus-alertmanager"

func init() {
	integrations.Register(plugin{})
}

type plugin struct{}

func (plugin) Name() string { return pluginName }

func (p plugin) ParseAlert(raw []byte, headers http.Header) (integrations.AlertCreate, error) {
	alerts, err := p.ParseAlerts(raw, headers)
	if err != nil {
		return integrations.AlertCreate{}, err
	}
	if len(alerts) == 0 {
		return integrations.AlertCreate{}, errNoAlerts
	}
	return alerts[0], nil
}

func (p plugin) ParseAlerts(raw []byte, headers http.Header) ([]integrations.AlertCreate, error) {
	_ = headers

	var msg webhookMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil, errInvalidPayload
	}
	if len(msg.Alerts) == 0 {
		return nil, errNoAlerts
	}
	if msg.Version != "" && msg.Version != "4" {
		return nil, errUnsupportedVersion
	}

	out := make([]integrations.AlertCreate, 0, len(msg.Alerts))
	for _, alert := range msg.Alerts {
		parsed, err := alert.toAlertCreate()
		if err != nil {
			return nil, err
		}
		out = append(out, parsed)
	}
	return out, nil
}

func (plugin) ValidateConfig(json.RawMessage) error { return nil }

func (plugin) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","additionalProperties":false}`)
}

type webhookMessage struct {
	Version string         `json:"version"`
	Status  string         `json:"status"`
	Alerts  []webhookAlert `json:"alerts"`
}

type webhookAlert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	Fingerprint  string            `json:"fingerprint"`
	GeneratorURL string            `json:"generatorURL"`
}

func (a webhookAlert) toAlertCreate() (integrations.AlertCreate, error) {
	fingerprint := strings.TrimSpace(a.Fingerprint)
	if fingerprint == "" {
		return integrations.AlertCreate{}, errMissingFingerprint
	}

	eventType := integrations.EventTriggered
	if strings.EqualFold(strings.TrimSpace(a.Status), "resolved") {
		eventType = integrations.EventResolved
	}

	summary := strings.TrimSpace(a.Annotations["summary"])
	if summary == "" {
		summary = strings.TrimSpace(a.Labels["alertname"])
	}
	if summary == "" {
		summary = "Alertmanager alert"
	}

	description := strings.TrimSpace(a.Annotations["description"])
	if description == "" && a.GeneratorURL != "" {
		description = a.GeneratorURL
	}

	return integrations.AlertCreate{
		EventType:   eventType,
		DedupKey:    fingerprint,
		Summary:     summary,
		Description: description,
		Priority:    mapSeverity(a.Labels["severity"]),
		Source:      "webhook:" + pluginName,
	}, nil
}

func mapSeverity(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical", "error", "warning", "page":
		return "high"
	default:
		return "low"
	}
}

type invalidPayloadError struct{}

func (invalidPayloadError) Error() string { return "invalid alertmanager webhook payload" }

type noAlertsError struct{}

func (noAlertsError) Error() string { return "alertmanager webhook contains no alerts" }

type unsupportedVersionError struct{}

func (unsupportedVersionError) Error() string { return "unsupported alertmanager webhook version" }

type missingFingerprintError struct{}

func (missingFingerprintError) Error() string { return "alertmanager alert missing fingerprint" }

var (
	errInvalidPayload     = invalidPayloadError{}
	errNoAlerts           = noAlertsError{}
	errUnsupportedVersion = unsupportedVersionError{}
	errMissingFingerprint = missingFingerprintError{}
)
