package integrations

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
	return fmt.Sprintf("unknown inbound plugin %q", e.Name)
}

var registry = map[string]InboundPlugin{}

// Register adds an inbound plugin to the compile-time registry.
func Register(plugin InboundPlugin) {
	if plugin == nil {
		panic("integrations: Register(nil)")
	}
	name := strings.TrimSpace(plugin.Name())
	if name == "" {
		panic("integrations: Register with empty name")
	}
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("integrations: duplicate registration for %q", name))
	}
	registry[name] = plugin
}

// Get returns the inbound plugin for name.
func Get(name string) (InboundPlugin, error) {
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

// List returns registered inbound plugins in name order.
func List() []InboundPlugin {
	names := Names()
	plugins := make([]InboundPlugin, 0, len(names))
	for _, name := range names {
		plugins = append(plugins, registry[name])
	}
	return plugins
}

// ValidateConfig validates cfg for the named inbound plugin.
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

// ConfigSchema returns the JSON Schema for the named inbound plugin.
func ConfigSchema(name string) (json.RawMessage, error) {
	plugin, err := Get(name)
	if err != nil {
		return nil, err
	}
	return plugin.ConfigSchema(), nil
}

func init() {
	// Register inbound plugins here — one import + Register call per plugin package.
}
