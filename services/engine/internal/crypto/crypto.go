package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
)

const (
	DefaultKeyID = "v1"

	keySize   = 32
	nonceSize = 12
)

// Encrypted holds provider secret ciphertext and the encryption key id for DB storage.
type Encrypted struct {
	KeyID      string
	Ciphertext []byte
}

// Box encrypts and decrypts provider credentials with AES-256-GCM.
type Box struct {
	aead  cipher.AEAD
	keyID string
}

// NewBox creates an encryptor from a 32-byte AES-256 key.
func NewBox(key []byte) (*Box, error) {
	if len(key) != keySize {
		return nil, fmt.Errorf("encryption key must be %d bytes for AES-256-GCM", keySize)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	return &Box{aead: aead, keyID: DefaultKeyID}, nil
}

// KeyID returns the key identifier used for newly encrypted secrets.
func (b *Box) KeyID() string {
	return b.keyID
}

// Decrypt opens ciphertext encrypted by this box.
func (b *Box) Decrypt(enc Encrypted) ([]byte, error) {
	if enc.KeyID != b.keyID {
		return nil, fmt.Errorf(
			"encryption key id mismatch: secret encrypted with key id %q but this server has %q configured",
			enc.KeyID,
			b.keyID,
		)
	}

	if len(enc.Ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short to contain nonce")
	}

	nonce := enc.Ciphertext[:nonceSize]
	ciphertext := enc.Ciphertext[nonceSize:]

	plaintext, err := b.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt secret: %w", err)
	}

	return plaintext, nil
}
