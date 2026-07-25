package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

const integrationKeyTokenBytes = 32

// NewIntegrationKeyToken returns a URL-safe plaintext token (≥128-bit entropy),
// its SHA-256 hex hash for storage, and a display-safe prefix for logging.
func NewIntegrationKeyToken() (plaintext string, hash string, prefix string, err error) {
	raw := make([]byte, integrationKeyTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", "", "", fmt.Errorf("generate integration key token: %w", err)
	}

	plaintext = base64.RawURLEncoding.EncodeToString(raw)
	hash = HashPasswordResetToken(plaintext)
	prefix = TokenPrefix(plaintext)
	return plaintext, hash, prefix, nil
}
