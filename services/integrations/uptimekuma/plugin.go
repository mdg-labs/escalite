// Package uptimekuma implements the uptime-kuma inbound plugin.
package uptimekuma

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/mdg-labs/escalite/services/integrations"
)

const pluginName = "uptime-kuma"

func init() {
	integrations.Register(plugin{})
}

type plugin struct{}

func (plugin) Name() string { return pluginName }

func (plugin) ParseAlert(raw []byte, headers http.Header) (integrations.AlertCreate, error) {
	_ = headers

	event, err := parseWebhookEvent(raw)
	if err != nil {
		return integrations.AlertCreate{}, err
	}
	return event.toAlertCreate()
}

func (plugin) ValidateConfig(json.RawMessage) error { return nil }

func (plugin) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","additionalProperties":false}`)
}

type webhookEvent struct {
	Msg       string         `json:"msg"`
	Heartbeat heartbeatEvent `json:"heartbeat"`
	Monitor   monitorEvent   `json:"monitor"`
}

type heartbeatEvent struct {
	MonitorID json.RawMessage `json:"monitorID"`
	Status    json.RawMessage `json:"status"`
	Msg       string          `json:"msg"`
}

type monitorEvent struct {
	ID   json.RawMessage `json:"id"`
	Name string          `json:"name"`
	URL  string          `json:"url"`
	Type string          `json:"type"`
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

func (e webhookEvent) toAlertCreate() (integrations.AlertCreate, error) {
	monitorID, err := monitorIDFromEvent(e)
	if err != nil {
		return integrations.AlertCreate{}, err
	}

	status, err := heartbeatStatusFromEvent(e.Heartbeat)
	if err != nil {
		return integrations.AlertCreate{}, err
	}

	summary := strings.TrimSpace(e.Monitor.Name)
	if summary == "" {
		summary = strings.TrimSpace(e.Msg)
	}
	if summary == "" {
		summary = "Uptime Kuma monitor alert"
	}

	description := firstNonEmpty(
		strings.TrimSpace(e.Msg),
		strings.TrimSpace(e.Heartbeat.Msg),
		strings.TrimSpace(e.Monitor.URL),
	)

	return integrations.AlertCreate{
		EventType:   mapEventType(status),
		DedupKey:    monitorID,
		Summary:     summary,
		Description: description,
		Priority:    mapPriority(status),
		Source:      "webhook:" + pluginName,
	}, nil
}

func monitorIDFromEvent(event webhookEvent) (string, error) {
	if id, err := parseID(event.Monitor.ID); err == nil && id != "" {
		return id, nil
	}
	if id, err := parseID(event.Heartbeat.MonitorID); err == nil && id != "" {
		return id, nil
	}
	return "", errMissingMonitorID
}

func heartbeatStatusFromEvent(heartbeat heartbeatEvent) (int, error) {
	if len(heartbeat.Status) == 0 || string(heartbeat.Status) == "null" {
		return 0, errMissingHeartbeatStatus
	}

	var asNumber int
	if err := json.Unmarshal(heartbeat.Status, &asNumber); err == nil {
		return asNumber, nil
	}

	var asString string
	if err := json.Unmarshal(heartbeat.Status, &asString); err == nil {
		switch strings.ToLower(strings.TrimSpace(asString)) {
		case "down", "0":
			return 0, nil
		case "up", "1":
			return 1, nil
		case "pending", "2":
			return 2, nil
		case "maintenance", "3":
			return 3, nil
		}
	}

	return 0, errInvalidHeartbeatStatus
}

func parseID(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", errMissingMonitorID
	}

	var asNumber int64
	if err := json.Unmarshal(raw, &asNumber); err == nil {
		return strconv.FormatInt(asNumber, 10), nil
	}

	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		if trimmed := strings.TrimSpace(asString); trimmed != "" {
			return trimmed, nil
		}
	}

	return "", errMissingMonitorID
}

func mapEventType(status int) integrations.EventType {
	if status == 1 {
		return integrations.EventResolved
	}
	return integrations.EventTriggered
}

func mapPriority(status int) string {
	if status == 0 {
		return "high"
	}
	return "low"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

type invalidPayloadError struct{}

func (invalidPayloadError) Error() string { return "invalid uptime-kuma webhook payload" }

type missingMonitorIDError struct{}

func (missingMonitorIDError) Error() string { return "uptime-kuma webhook missing monitor id" }

type missingHeartbeatStatusError struct{}

func (missingHeartbeatStatusError) Error() string { return "uptime-kuma webhook missing heartbeat status" }

type invalidHeartbeatStatusError struct{}

func (invalidHeartbeatStatusError) Error() string { return "invalid uptime-kuma heartbeat status" }

var (
	errInvalidPayload          = invalidPayloadError{}
	errMissingMonitorID        = missingMonitorIDError{}
	errMissingHeartbeatStatus  = missingHeartbeatStatusError{}
	errInvalidHeartbeatStatus  = invalidHeartbeatStatusError{}
)
