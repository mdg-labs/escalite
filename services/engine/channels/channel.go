package channels

import (
	"context"
	"encoding/json"
)

// Target is the resolved delivery destination for a notification attempt.
type Target struct {
	Type   string `json:"type"`
	UserID string `json:"user_id,omitempty"`
	Email  string `json:"email,omitempty"`
}

// Alert is the alert payload passed to channel Send implementations.
type Alert struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	ServiceID      string `json:"service_id"`
	Summary        string `json:"summary"`
	Description    string `json:"description,omitempty"`
	Priority       string `json:"priority"`
	Status         string `json:"status"`
}

// SendParams bundles delivery context for a channel plugin.
type SendParams struct {
	Target Target
	Alert  Alert
	Config json.RawMessage
}

// NotificationChannel is the outbound mirror of inbound integration plugins.
type NotificationChannel interface {
	Name() string
	Send(ctx context.Context, params SendParams) error
	ValidateConfig(cfg json.RawMessage) error
	ConfigSchema() json.RawMessage
}
