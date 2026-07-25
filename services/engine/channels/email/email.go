package email

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mdg-labs/escalite/services/engine/channels"
)

const name = "email"

type channel struct{}

// New returns the email notification channel plugin.
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
	if len(cfg) == 0 {
		return nil
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal(cfg, &values); err != nil {
		return errors.New("config must be a JSON object")
	}
	if len(values) > 0 {
		return errors.New("email channel does not accept config fields")
	}
	return nil
}

func (c *channel) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"additionalProperties": false
	}`)
}
