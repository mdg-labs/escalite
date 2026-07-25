package graph

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/mdg-labs/escalite/services/engine/channels"
)

func validateRegisteredNotificationChannel(channelName string) error {
	name := strings.TrimSpace(channelName)
	if name == "" {
		return fmt.Errorf("channel is required")
	}
	if _, err := channels.Get(name); err != nil {
		var unknown channels.ErrUnknownChannel
		if errors.As(err, &unknown) {
			return fmt.Errorf("unknown notification channel %q", unknown.Name)
		}
		return err
	}
	return nil
}

func validateNotificationChannelConfig(channelName string, config map[string]any) error {
	name := strings.TrimSpace(channelName)
	if name == "" {
		return fmt.Errorf("channel is required")
	}

	raw, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("config must be a JSON object")
	}

	if err := channels.ValidateConfig(name, raw); err != nil {
		var unknown channels.ErrUnknownChannel
		if errors.As(err, &unknown) {
			return fmt.Errorf("unknown notification channel %q", unknown.Name)
		}
		return err
	}

	return nil
}

func configMapToRaw(config map[string]any) ([]byte, error) {
	if config == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(config)
}
