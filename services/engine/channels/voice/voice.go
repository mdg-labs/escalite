package voice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/mdg-labs/escalite/services/engine/channels"
	"github.com/mdg-labs/escalite/services/engine/internal/smsprovider"
)

const name = "voice"

var provider smsprovider.Provider

// SetProvider configures the SMS/voice provider used by SMS and voice channels.
func SetProvider(p smsprovider.Provider) {
	provider = p
}

type channel struct{}

// New returns the voice notification channel plugin.
func New() channels.NotificationChannel {
	return &channel{}
}

func (c *channel) Name() string {
	return name
}

func (c *channel) Send(ctx context.Context, params channels.SendParams) error {
	if provider == nil {
		return smsprovider.ErrNotConfigured
	}

	cfg, err := parseConfig(params.Config)
	if err != nil {
		return err
	}

	return provider.MakeVoiceCall(ctx, smsprovider.VoiceParams{
		To:      cfg.PhoneNumber,
		Message: buildVoiceMessage(params.Alert),
	})
}

func (c *channel) ValidateConfig(cfg json.RawMessage) error {
	return channels.ValidateObjectConfig(cfg, map[string]func(string) error{
		"phone_number": validatePhoneNumber,
	})
}

func (c *channel) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"phone_number": {
				"type": "string",
				"minLength": 1,
				"title": "Phone number",
				"description": "E.164 phone number for voice call delivery"
			}
		},
		"required": ["phone_number"],
		"additionalProperties": false
	}`)
}

type channelConfig struct {
	PhoneNumber string `json:"phone_number"`
}

func parseConfig(raw json.RawMessage) (channelConfig, error) {
	if len(raw) == 0 {
		return channelConfig{}, errors.New("phone_number is required")
	}
	var cfg channelConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return channelConfig{}, errors.New("config must be a JSON object")
	}
	if err := validatePhoneNumber(cfg.PhoneNumber); err != nil {
		return channelConfig{}, fmt.Errorf("phone_number: %w", err)
	}
	return cfg, nil
}

func validatePhoneNumber(value string) error {
	phone := strings.TrimSpace(value)
	if phone == "" {
		return errors.New("must not be empty")
	}
	if !strings.HasPrefix(phone, "+") {
		return errors.New("must be in E.164 format")
	}
	return nil
}

func buildVoiceMessage(alert channels.Alert) string {
	service := strings.TrimSpace(alert.ServiceName)
	if service == "" {
		service = strings.TrimSpace(alert.ServiceID)
	}
	if service == "" {
		return alert.Summary
	}
	return fmt.Sprintf("%s on %s", alert.Summary, service)
}
