package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

const resetTokenBytes = 32

// NewPasswordResetToken returns a URL-safe plaintext token and its SHA-256 hex hash for storage.
func NewPasswordResetToken() (plaintext string, hash string, err error) {
	raw := make([]byte, resetTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("generate reset token: %w", err)
	}

	plaintext = base64.RawURLEncoding.EncodeToString(raw)
	hash = HashPasswordResetToken(plaintext)
	return plaintext, hash, nil
}

// HashPasswordResetToken returns the SHA-256 hex digest of a plaintext reset token.
func HashPasswordResetToken(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}
