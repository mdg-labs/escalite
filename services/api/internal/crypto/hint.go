package crypto

// SecretHint returns the last four characters of a secret for UI display after save.
// Full secrets must never be rendered back to clients (doc 07).
func SecretHint(secret string) string {
	runes := []rune(secret)
	if len(runes) <= 4 {
		return secret
	}
	return string(runes[len(runes)-4:])
}
