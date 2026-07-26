package auth

// NewScimBearerToken returns a URL-safe plaintext token (≥128-bit entropy),
// its SHA-256 hex hash for storage, and a display-safe prefix.
func NewScimBearerToken() (plaintext string, hash string, prefix string, err error) {
	const scimTokenBytes = 32
	plaintext, hash, err = newHashedToken(scimTokenBytes)
	if err != nil {
		return "", "", "", err
	}
	prefix = TokenPrefix(plaintext)
	return plaintext, hash, prefix, nil
}

// HashScimBearerToken returns the SHA-256 hex digest of a plaintext SCIM bearer token.
func HashScimBearerToken(plaintext string) string {
	return hashOpaqueToken(plaintext)
}

// IsUserActive reports whether the user may authenticate or hold an active session.
func IsUserActive(deprovisionedAtValid bool) bool {
	return !deprovisionedAtValid
}
