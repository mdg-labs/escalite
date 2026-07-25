package smsprovider

import (
	"fmt"
	"strings"
)

// ErrUnknownProvider is returned when a provider name is not registered.
type ErrUnknownProvider struct {
	Name string
}

func (e ErrUnknownProvider) Error() string {
	return fmt.Sprintf("unknown sms/voice provider %q", e.Name)
}

var registry = map[string]Provider{}

// Register adds an SMS/voice provider to the compile-time registry.
func Register(provider Provider) {
	if provider == nil {
		panic("smsprovider: Register(nil)")
	}
	name := strings.TrimSpace(provider.Name())
	if name == "" {
		panic("smsprovider: Register with empty name")
	}
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("smsprovider: duplicate registration for %q", name))
	}
	registry[name] = provider
}

// Get returns the provider for name.
func Get(name string) (Provider, error) {
	provider, ok := registry[strings.TrimSpace(name)]
	if !ok {
		return nil, ErrUnknownProvider{Name: name}
	}
	return provider, nil
}
