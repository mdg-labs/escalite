package outboundintegrations

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"
)

// RequireNonEmpty rejects blank strings.
func RequireNonEmpty(value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("value is required")
	}
	return nil
}

// RequireHTTPURL rejects values that are not http(s) URLs.
func RequireHTTPURL(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("URL is required")
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return errors.New("URL must be a valid http or https URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("URL must use http or https")
	}
	return nil
}

// ValidateObjectConfig validates cfg against required field validators.
func ValidateObjectConfig(cfg json.RawMessage, fields map[string]func(string) error) error {
	if len(cfg) == 0 {
		for field, validate := range fields {
			if err := validate(""); err != nil {
				return errors.New(field + " is required")
			}
		}
		return nil
	}

	var values map[string]json.RawMessage
	if err := json.Unmarshal(cfg, &values); err != nil {
		return errors.New("config must be a JSON object")
	}

	for field, validate := range fields {
		raw, ok := values[field]
		if !ok {
			if err := validate(""); err != nil {
				return errors.New(field + " is required")
			}
			continue
		}

		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return errors.New(field + " must be a string")
		}
		if err := validate(value); err != nil {
			return err
		}
	}

	return nil
}
