package auth

import "strings"

// ParseBearerToken extracts the token from an Authorization: Bearer header.
// The second return value is false when the header is missing or malformed.
func ParseBearerToken(header string) (token string, ok bool) {
	header = strings.TrimSpace(header)
	if header == "" {
		return "", false
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}

	token = strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}
	return token, true
}
