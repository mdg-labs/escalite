package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

const heartbeatTokenBytes = 32

// TokenPrefix returns the display-safe prefix for logging (doc 07).
func TokenPrefix(token string) string {
	if len(token) >= 8 {
		return token[:8]
	}
	return token
}

// NewHeartbeatToken returns a URL-safe plaintext token (≥128-bit entropy) and its SHA-256 hex hash for storage.
func NewHeartbeatToken() (plaintext string, hash string, prefix string, err error) {
	raw := make([]byte, heartbeatTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", "", "", fmt.Errorf("generate heartbeat token: %w", err)
	}

	plaintext = base64.RawURLEncoding.EncodeToString(raw)
	hash = HashPasswordResetToken(plaintext)
	prefix = TokenPrefix(plaintext)
	return plaintext, hash, prefix, nil
}
