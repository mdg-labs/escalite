package push

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mdg-labs/escalite/services/engine/channels"
)

const name = "push"

type channel struct{}

// New returns the push notification channel plugin.
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
		"expo_push_token": channels.RequireNonEmpty,
	})
}

func (c *channel) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"expo_push_token": {
				"type": "string",
				"minLength": 1,
				"title": "Expo push token"
			}
		},
		"required": ["expo_push_token"],
		"additionalProperties": false
	}`)
}
