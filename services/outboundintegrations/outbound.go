package outboundintegrations

import (
	"context"
	"encoding/json"
)

// Incident is the incident context passed to outbound ticketing plugins.
type Incident struct {
	ID             string
	OrganizationID string
	TeamID         string
	Title          string
}

// Ticket is the external ticket created by an outbound plugin.
type Ticket struct {
	URL string
}

// OutboundPlugin creates external tickets when incidents are declared.
type OutboundPlugin interface {
	Name() string
	CreateTicket(ctx context.Context, incident Incident, cfg json.RawMessage, apiToken string) (Ticket, error)
	ValidateConfig(cfg json.RawMessage) error
	ConfigSchema() json.RawMessage
}
