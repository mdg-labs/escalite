package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"
)

const (
	refreshTokenBytes   = 32
	mobileAuthCodeBytes = 32
	// DefaultRefreshTokenTTL is how long a mobile device refresh token remains valid.
	DefaultRefreshTokenTTL = 90 * 24 * time.Hour
	// DefaultMobileAuthCodeTTL is how long a one-time mobile auth code remains valid.
	DefaultMobileAuthCodeTTL = 5 * time.Minute
)

// NewRefreshToken returns a URL-safe plaintext token and its SHA-256 hex hash for storage.
func NewRefreshToken() (plaintext string, hash string, err error) {
	return newHashedToken(refreshTokenBytes)
}

// HashRefreshToken returns the SHA-256 hex digest of a plaintext refresh token.
func HashRefreshToken(plaintext string) string {
	return hashOpaqueToken(plaintext)
}

// NewMobileAuthCode returns a URL-safe plaintext code and its SHA-256 hex hash for storage.
func NewMobileAuthCode() (plaintext string, hash string, err error) {
	return newHashedToken(mobileAuthCodeBytes)
}

// HashMobileAuthCode returns the SHA-256 hex digest of a plaintext mobile auth code.
func HashMobileAuthCode(plaintext string) string {
	return hashOpaqueToken(plaintext)
}

func newHashedToken(size int) (plaintext string, hash string, err error) {
	raw := make([]byte, size)
	if _, err := rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("generate token: %w", err)
	}

	plaintext = base64.RawURLEncoding.EncodeToString(raw)
	hash = hashOpaqueToken(plaintext)
	return plaintext, hash, nil
}

func hashOpaqueToken(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}
