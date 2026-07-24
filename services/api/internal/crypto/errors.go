package crypto

import (
	"errors"
	"fmt"
)

// KeyIDMismatchError is returned when ciphertext was encrypted with a different key id.
type KeyIDMismatchError struct {
	Configured string
	Stored     string
}

func (e *KeyIDMismatchError) Error() string {
	return fmt.Sprintf(
		"encryption key id mismatch: secret encrypted with key id %q but this server has %q configured; re-encrypt the credential or restore ESCALITE_ENCRYPTION_KEY for key id %q (see docs/specs/07-security-and-auth.md)",
		e.Stored,
		e.Configured,
		e.Stored,
	)
}

// IsKeyIDMismatch reports whether err is a key id mismatch.
func IsKeyIDMismatch(err error) bool {
	var mismatch *KeyIDMismatchError
	return errors.As(err, &mismatch)
}
