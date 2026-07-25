package channels

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

func ValidateObjectConfig(cfg json.RawMessage, allowedFields map[string]func(string) error) error {
	if len(cfg) == 0 {
		cfg = json.RawMessage(`{}`)
	}

	var values map[string]json.RawMessage
	if err := json.Unmarshal(cfg, &values); err != nil {
		return fmt.Errorf("config must be a JSON object")
	}

	for key := range values {
		if _, ok := allowedFields[key]; !ok && len(allowedFields) > 0 {
			// Allow only known fields when schema defines properties.
			if key != "" {
				return fmt.Errorf("unknown config field %q", key)
			}
		}
	}

	for field, validate := range allowedFields {
		raw, ok := values[field]
		if !ok {
			return fmt.Errorf("%s is required", field)
		}
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return fmt.Errorf("%s must be a string", field)
		}
		if err := validate(value); err != nil {
			return fmt.Errorf("%s: %w", field, err)
		}
	}

	return nil
}

func RequireNonEmpty(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("must not be empty")
	}
	return nil
}

func RequireHTTPURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("must be a valid URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("must use http or https")
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return fmt.Errorf("must be a valid URL")
	}
	return nil
}
