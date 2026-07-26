package outboundintegrations

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// ErrUnknownPlugin is returned when a plugin name is not registered.
type ErrUnknownPlugin struct {
	Name string
}

func (e ErrUnknownPlugin) Error() string {
	return fmt.Sprintf("unknown outbound integration %q", e.Name)
}

var registry = map[string]OutboundPlugin{}

// Register adds an outbound integration plugin to the compile-time registry.
func Register(plugin OutboundPlugin) {
	if plugin == nil {
		panic("outboundintegrations: Register(nil)")
	}
	name := strings.TrimSpace(plugin.Name())
	if name == "" {
		panic("outboundintegrations: Register with empty name")
	}
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("outboundintegrations: duplicate registration for %q", name))
	}
	registry[name] = plugin
}

// Get returns the outbound plugin for name.
func Get(name string) (OutboundPlugin, error) {
	plugin, ok := registry[strings.TrimSpace(name)]
	if !ok {
		return nil, ErrUnknownPlugin{Name: name}
	}
	return plugin, nil
}

// Names returns registered plugin names in stable order.
func Names() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ValidateConfig validates cfg for the named outbound plugin.
func ValidateConfig(name string, cfg json.RawMessage) error {
	plugin, err := Get(name)
	if err != nil {
		return err
	}
	if len(cfg) == 0 {
		cfg = json.RawMessage(`{}`)
	}
	return plugin.ValidateConfig(cfg)
}

// ConfigSchema returns the JSON Schema for the named outbound plugin.
func ConfigSchema(name string) (json.RawMessage, error) {
	plugin, err := Get(name)
	if err != nil {
		return nil, err
	}
	return plugin.ConfigSchema(), nil
}
