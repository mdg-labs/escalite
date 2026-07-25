package webhook

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mdg-labs/escalite/services/engine/channels"
)

const name = "webhook"

type channel struct{}

// New returns the outbound webhook notification channel plugin.
func New() channels.NotificationChannel {
	return &channel{}
}

func (c *channel) Name() string {
	return name
}

func (c *channel) Send(_ context.Context, _ channels.SendParams) error {
	return fmt.Errorf("channel %s: send not implemented", name)
}

func (c *channel) ValidateConfig(cfg json.RawMessage) error {
	return channels.ValidateObjectConfig(cfg, map[string]func(string) error{
		"url": channels.RequireHTTPURL,
	})
}

func (c *channel) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"url": {
				"type": "string",
				"format": "uri",
				"title": "Webhook URL"
			}
		},
		"required": ["url"],
		"additionalProperties": false
	}`)
}
