package scim

import (
	"fmt"
	"strings"
)

// ParseUserNameFilter extracts the userName value from a SCIM filter expression.
// Supports: userName eq "value" (case-insensitive operator).
func ParseUserNameFilter(filter string) (string, bool) {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return "", false
	}

	parts := strings.Fields(filter)
	if len(parts) != 3 {
		return "", false
	}
	if !strings.EqualFold(parts[0], "userName") || !strings.EqualFold(parts[1], "eq") {
		return "", false
	}

	value := strings.Trim(parts[2], `"'`)
	if value == "" {
		return "", false
	}
	return value, true
}

// ParseDisplayNameFilter extracts the displayName value from a SCIM filter expression.
func ParseDisplayNameFilter(filter string) (string, bool) {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return "", false
	}

	parts := strings.Fields(filter)
	if len(parts) != 3 {
		return "", false
	}
	if !strings.EqualFold(parts[0], "displayName") || !strings.EqualFold(parts[1], "eq") {
		return "", false
	}

	value := strings.Trim(parts[2], `"'`)
	if value == "" {
		return "", false
	}
	return value, true
}

// MemberUserID extracts a user ID from a SCIM member patch value.
func MemberUserID(value any) (string, error) {
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return "", fmt.Errorf("empty member value")
		}
		return strings.TrimSpace(typed), nil
	case map[string]any:
		raw, ok := typed["value"].(string)
		if !ok || strings.TrimSpace(raw) == "" {
			return "", fmt.Errorf("invalid member value")
		}
		return strings.TrimSpace(raw), nil
	default:
		return "", fmt.Errorf("unsupported member value type")
	}
}

// ActiveFromPatchValue coerces a SCIM PATCH active value to bool.
func ActiveFromPatchValue(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	default:
		return false, false
	}
}
