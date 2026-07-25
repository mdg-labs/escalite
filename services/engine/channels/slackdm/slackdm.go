package slackdm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mdg-labs/escalite/services/engine/channels"
)

const name = "slack-dm"

type channel struct{}

// New returns the Slack DM notification channel plugin.
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
		"slack_user_id": channels.RequireNonEmpty,
	})
}

func (c *channel) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"slack_user_id": {
				"type": "string",
				"minLength": 1,
				"title": "Slack user ID"
			}
		},
		"required": ["slack_user_id"],
		"additionalProperties": false
	}`)
}
