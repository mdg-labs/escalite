package integrations

import (
	"encoding/json"
	"net/http"
)

// EventType classifies an inbound alert event.
type EventType string

const (
	// EventTriggered signals a new or ongoing alert condition.
	EventTriggered EventType = "triggered"
	// EventResolved signals recovery; the matching open alert (by dedup key) is auto-closed.
	EventResolved EventType = "resolved"
)

// AlertCreate is the normalized alert payload produced by inbound plugins.
type AlertCreate struct {
	EventType   EventType `json:"event_type"`
	DedupKey    string    `json:"dedup_key"`
	Summary     string    `json:"summary"`
	Description string    `json:"description,omitempty"`
	Priority    string    `json:"priority,omitempty"`
	Source      string    `json:"source,omitempty"`
}

// InboundPlugin parses inbound integration payloads into AlertCreate values.
type InboundPlugin interface {
	Name() string

	// ParseAlert converts a raw inbound payload into an AlertCreate.
	// EventType may be EventTriggered or EventResolved; resolved events auto-close
	// the matching open alert by dedup key instead of creating duplicate noise.
	ParseAlert(raw []byte, headers http.Header) (AlertCreate, error)

	ValidateConfig(cfg json.RawMessage) error
	ConfigSchema() json.RawMessage
}

// MultiAlertInboundPlugin parses payloads that may contain multiple alerts.
type MultiAlertInboundPlugin interface {
	InboundPlugin
	ParseAlerts(raw []byte, headers http.Header) ([]AlertCreate, error)
}

// ParseAll returns every alert in a payload, using ParseAlerts when implemented.
func ParseAll(plugin InboundPlugin, raw []byte, headers http.Header) ([]AlertCreate, error) {
	if batch, ok := plugin.(MultiAlertInboundPlugin); ok {
		return batch.ParseAlerts(raw, headers)
	}
	alert, err := plugin.ParseAlert(raw, headers)
	if err != nil {
		return nil, err
	}
	return []AlertCreate{alert}, nil
}
