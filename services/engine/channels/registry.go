package channels

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// ErrUnknownChannel is returned when a channel name is not registered.
type ErrUnknownChannel struct {
	Name string
}

func (e ErrUnknownChannel) Error() string {
	return fmt.Sprintf("unknown notification channel %q", e.Name)
}

var registry = map[string]NotificationChannel{}

// Register adds a notification channel plugin to the compile-time registry.
func Register(channel NotificationChannel) {
	if channel == nil {
		panic("channels: Register(nil)")
	}
	name := strings.TrimSpace(channel.Name())
	if name == "" {
		panic("channels: Register with empty name")
	}
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("channels: duplicate registration for %q", name))
	}
	registry[name] = channel
}

// Get returns the channel plugin for name.
func Get(name string) (NotificationChannel, error) {
	channel, ok := registry[strings.TrimSpace(name)]
	if !ok {
		return nil, ErrUnknownChannel{Name: name}
	}
	return channel, nil
}

// Names returns registered channel names in stable order.
func Names() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// List returns registered channel plugins in name order.
func List() []NotificationChannel {
	names := Names()
	channels := make([]NotificationChannel, 0, len(names))
	for _, name := range names {
		channels = append(channels, registry[name])
	}
	return channels
}

// ValidateConfig validates cfg for the named channel plugin.
func ValidateConfig(name string, cfg json.RawMessage) error {
	channel, err := Get(name)
	if err != nil {
		return err
	}
	if len(cfg) == 0 {
		cfg = json.RawMessage(`{}`)
	}
	return channel.ValidateConfig(cfg)
}

// ConfigSchema returns the JSON Schema for the named channel plugin.
func ConfigSchema(name string) (json.RawMessage, error) {
	channel, err := Get(name)
	if err != nil {
		return nil, err
	}
	return channel.ConfigSchema(), nil
}
