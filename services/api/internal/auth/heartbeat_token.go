package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

const heartbeatTokenBytes = 32

// NewHeartbeatToken returns a URL-safe plaintext token (≥128-bit entropy) and its SHA-256 hex hash for storage.
func NewHeartbeatToken() (plaintext string, hash string, prefix string, err error) {
	raw := make([]byte, heartbeatTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", "", "", fmt.Errorf("generate heartbeat token: %w", err)
	}

	plaintext = base64.RawURLEncoding.EncodeToString(raw)
	hash = HashPasswordResetToken(plaintext)
	if len(plaintext) >= 8 {
		prefix = plaintext[:8]
	} else {
		prefix = plaintext
	}
	return plaintext, hash, prefix, nil
}
