package auth

import (
	"github.com/alexedwards/argon2id"
)

// HashPassword returns an Argon2id hash suitable for storage in users.password_hash.
// Plain-text passwords must never be logged or persisted.
func HashPassword(password string) (string, error) {
	return argon2id.CreateHash(password, argon2id.DefaultParams)
}

// VerifyPassword checks a plain-text password against a stored Argon2id hash.
func VerifyPassword(password, encodedHash string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, encodedHash)
}
